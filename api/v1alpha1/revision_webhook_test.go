package v1alpha1

import (
	"testing"
	"time"

	testify "github.com/stretchr/testify/assert"
	"gomodules.xyz/jsonpatch/v2"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type revisionBuilder struct {
	*Revision
	portCounter int32
}

func (r *revisionBuilder) internalAddPort(target string, name string, mode PortMode, ingresses ...string) *revisionBuilder {
	r.portCounter += 1
	r.ensureTarget(target)
	for i := range r.Spec.Targets {
		if r.Spec.Targets[i].Name == target {
			r.Spec.Targets[i].Ports = append(r.Spec.Targets[i].Ports, PortInfo{
				Name:          name,
				IngressPort:   r.portCounter,
				Port:          r.portCounter,
				ContainerPort: r.portCounter,
				Protocol:      "TCP",
				Mode:          mode,
				Ingresses:     ingresses,
			})
			return r
		}
	}
	panic("Bug")
}

func (r *revisionBuilder) addPortWithMode(target, name string, mode PortMode) *revisionBuilder {
	r.ensureTarget(target)
	return r.internalAddPort(target, name, mode)
}

func (r *revisionBuilder) addPort(target, name string, ingresses ...string) *revisionBuilder {
	r.ensureTarget(target)
	return r.internalAddPort(target, name, "", ingresses...)
}

func (r *revisionBuilder) addIngressDefaultPort(target, ingress, port string) *revisionBuilder {
	r.ensureTarget(target)
	for i := range r.Spec.Targets {
		if r.Spec.Targets[i].Name == target {
			if r.Spec.Targets[i].DefaultIngressPorts == nil {
				r.Spec.Targets[i].DefaultIngressPorts = map[string]string{}
			}
			r.Spec.Targets[i].DefaultIngressPorts[ingress] = port
			break
		}
	}
	return r
}

func (r *revisionBuilder) setRate(target string, scaling LinearScaling) *revisionBuilder {
	r.ensureTarget(target)
	for i := range r.Spec.Targets {
		if r.Spec.Targets[i].Name == target {
			r.Spec.Targets[i].Release.ScalingStrategy = ScalingStrategyLinear
			r.Spec.Targets[i].Release.LinearScaling = scaling
			break
		}
	}
	return r
}

func (r *revisionBuilder) ensureTarget(name string) *revisionBuilder {
	for _, target := range r.Spec.Targets {
		if target.Name == name {
			return r
		}
	}
	r.Spec.Targets = append(r.Spec.Targets, RevisionTarget{
		Name: name,
		Release: ReleaseInfo{
			Eligible:         false,
			Max:              0,
			ScalingStrategy:  ScalingStrategyLinear,
			GeometricScaling: GeometricScaling{},
			LinearScaling:    LinearScaling{},
			Schedule:         "",
			TTL:              0,
		},
	})
	return r
}

func (r *revisionBuilder) build() *Revision {
	return r.Revision
}

func newRevisionBuilder() *revisionBuilder {
	return &revisionBuilder{
		Revision: &Revision{
			ObjectMeta: meta.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: RevisionSpec{
				App: RevisionApp{
					Name:  "name",
					Ref:   "ref",
					Tag:   "tag",
					Image: "image",
				},
			},
		},
	}
}

func TestMutate(t *testing.T) {
	assert := testify.New(t)
	mutator := Revision{}

	ts := []struct {
		name     string
		rev      *Revision
		expected []jsonpatch.JsonPatchOperation
	}{
		{
			name: "SetHTTPAsDefaultPort",
			rev: newRevisionBuilder().
				addPortWithMode("production", "http", PortPrivate).
				addPortWithMode("production", "grpc", PortPublic).
				addPortWithMode("production", "status", PortLocal).
				build(),
			expected: []jsonpatch.JsonPatchOperation{
				{
					Operation: "add",
					Path:      "/spec/targets/0/ports/0/ingresses",
					Value:     []string{"private"},
				},
				{
					Operation: "add",
					Path:      "/spec/targets/0/ports/1/ingresses",
					Value:     []string{"private", "public"},
				},
				{
					Operation: "add",
					Path:      "/spec/targets/0/defaultIngressPorts",
					Value:     map[string]string{"private": "http", "public": "grpc"},
				},
			},
		},
		{
			name: "DontSetHTTPAsDefaultPort",
			rev: newRevisionBuilder().
				addPort("production", "http", "private").
				addPort("production", "grpc", "private").
				addPort("production", "status").
				addIngressDefaultPort("production", "private", "grpc").
				build(),
			expected: nil,
		},
		{
			name: "DontSetLinearScalingProperties",
			rev: newRevisionBuilder().
				setRate("production", LinearScaling{
					Increment: 10,
					Delay:     &meta.Duration{Duration: time.Duration(20) * time.Second},
				}).
				build(),
			expected: nil,
		},
		{
			name: "DontSetHTTPAsDefaultPort",
			rev: newRevisionBuilder().
				addPort("production", "http", "private").
				addPort("production", "grpc", "private").
				addPort("production", "status").
				addIngressDefaultPort("production", "private", "grpc").
				build(),
			expected: nil,
		},
		{
			name: "SpitModes",
			rev: newRevisionBuilder().
				addPort("production", "grpc", "private").
				addPort("production", "http", "public", "private").
				addPort("production", "status").
				addIngressDefaultPort("production", "private", "grpc").
				build(),
			expected: nil,
		},
		{
			name: "NoHTTP",
			rev: newRevisionBuilder().
				addPort("production", "grpc", "public", "private").
				addPort("production", "blah", "public", "private").
				addPort("production", "status").
				build(),
			expected: nil,
		},
		{
			name: "SinglePort",
			rev: newRevisionBuilder().
				addPort("production", "grpc", "public", "private").
				addPort("production", "status").
				build(),
			expected: nil,
		},
		{
			name: "SingleAndHTTP",
			rev: newRevisionBuilder().
				addPort("production", "grpc", "public", "private").
				addPort("production", "http", "private").
				addPort("production", "status").
				build(),
			expected: nil,
		},
		{
			name: "NoDefaultOnLocal",
			rev: newRevisionBuilder().
				addPort("production", "grpc").
				addPort("production", "http").
				addPort("production", "status").
				build(),
			expected: nil,
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, func(t *testing.T) {
			assert.ElementsMatch(tc.expected, mutator.getPatches())
		})
	}
}

func TestValidate(t *testing.T) {
	assert := testify.New(t)
	//validator := Revision{}

	ts := []struct {
		name     string
		rev      *Revision
		expected []string
	}{
		{
			name: "DefaultsSet",
			rev: newRevisionBuilder().
				addPort("production", "http", "private").
				addPort("production", "grpc", "public", "private").
				addPort("production", "status").
				addIngressDefaultPort("production", "private", "http").
				addIngressDefaultPort("production", "public", "grpc").
				build(),
			expected: nil,
		},
		{
			name: "NoDefault",
			rev: newRevisionBuilder().
				addPort("production", "http", "private").
				addPort("production", "grpc", "private").
				addPort("production", "status").
				build(),
			expected: []string{"production"},
		},
		{
			name: "DefaultLocal",
			rev: newRevisionBuilder().
				addPort("production", "status").
				addIngressDefaultPort("production", "local", "status").
				build(),
			expected: []string{"production"},
		},
		{
			name: "NoPorts",
			rev: newRevisionBuilder().
				build(),
			expected: nil,
		},
		{
			name: "MultiTargets",
			rev: newRevisionBuilder().
				addPort("staging", "http", "private").
				addPort("staging", "grpc", "public", "private").
				addPort("staging", "status").
				addIngressDefaultPort("staging", "private", "grpc").
				addIngressDefaultPort("staging", "public", "grpc").
				addPort("production", "http", "private").
				addPort("production", "grpc", "private").
				addPort("production", "status").
				addIngressDefaultPort("production", "private", "httpz").
				build(),
			expected: []string{"production"},
		},
		{
			name: "MultiTargetFailures",
			rev: newRevisionBuilder().
				addPort("staging", "http", "private").
				addPort("staging", "grpc", "public", "private").
				addPort("staging", "status").
				addPort("production", "http", "private").
				addPort("production", "grpc", "private").
				addPort("production", "status").
				build(),
			expected: []string{"production", "staging"},
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, func(t *testing.T) {
			failures := tc.rev.validate()
			assert.ElementsMatch(failures, nil)
		})
	}
}
