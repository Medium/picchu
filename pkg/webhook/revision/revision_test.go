package revision

import (
	testify "github.com/stretchr/testify/assert"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"gomodules.xyz/jsonpatch/v2"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"testing"
)

type revisionBuilder struct {
	*picchu.Revision
	portCounter int32
}

func (r *revisionBuilder) internalAddPort(target string, name string, mode picchu.PortMode, ingresses ...string) *revisionBuilder {
	r.portCounter += 1
	r.ensureTarget(target)
	for i := range r.Spec.Targets {
		if r.Spec.Targets[i].Name == target {
			r.Spec.Targets[i].Ports = append(r.Spec.Targets[i].Ports, picchu.PortInfo{
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

func (r *revisionBuilder) addPortWithMode(target, name string, mode picchu.PortMode) *revisionBuilder {
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

func (r *revisionBuilder) setRate(target string, rate picchu.RateInfo) *revisionBuilder {
	r.ensureTarget(target)
	for i := range r.Spec.Targets {
		if r.Spec.Targets[i].Name == target {
			r.Spec.Targets[i].Release.Rate = rate
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
	r.Spec.Targets = append(r.Spec.Targets, picchu.RevisionTarget{
		Name: name,
		Release: picchu.ReleaseInfo{
			Eligible:         false,
			Max:              0,
			ScalingStrategy:  picchu.ScalingStrategyLinear,
			GeometricScaling: picchu.GeometricScaling{},
			LinearScaling:    picchu.LinearScaling{},
			Rate:             picchu.RateInfo{},
			Schedule:         "",
			TTL:              0,
		},
	})
	return r
}

func (r *revisionBuilder) build() *picchu.Revision {
	return r.Revision
}

func newRevisionBuilder() *revisionBuilder {
	return &revisionBuilder{
		Revision: &picchu.Revision{
			ObjectMeta: meta.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: picchu.RevisionSpec{
				App: picchu.RevisionApp{
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
	mutator := revisionMutator{}

	ts := []struct {
		name     string
		rev      *picchu.Revision
		expected []jsonpatch.JsonPatchOperation
	}{
		{
			name: "SetHTTPAsDefaultPort",
			rev: newRevisionBuilder().
				addPortWithMode("production", "http", picchu.PortPrivate).
				addPortWithMode("production", "grpc", picchu.PortPublic).
				addPortWithMode("production", "status", picchu.PortLocal).
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
				setRate("production", picchu.RateInfo{
					Increment:    10,
					DelaySeconds: pointer.Int64Ptr(20),
				}).
				build(),
			expected: []jsonpatch.JsonPatchOperation{},
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
			expected: []jsonpatch.JsonPatchOperation{{
				Operation: "add",
				Path:      "/spec/targets/0/defaultIngressPorts[public]",
				Value:     "http",
			}},
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
			expected: []jsonpatch.JsonPatchOperation{
				{
					Operation: "add",
					Path:      "/spec/targets/0/defaultIngressPorts",
					Value:     map[string]string{"public": "grpc", "private": "grpc"},
				},
			},
		},
		{
			name: "SingleAndHTTP",
			rev: newRevisionBuilder().
				addPort("production", "grpc", "public", "private").
				addPort("production", "http", "private").
				addPort("production", "status").
				build(),
			expected: []jsonpatch.JsonPatchOperation{
				{
					Operation: "add",
					Path:      "/spec/targets/0/defaultIngressPorts",
					Value:     map[string]string{"public": "grpc", "private": "http"},
				},
			},
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
			assert.ElementsMatch(tc.expected, mutator.getPatches(tc.rev))
		})
	}
}

func TestValidate(t *testing.T) {
	assert := testify.New(t)
	validator := revisionValidator{}

	ts := []struct {
		name     string
		rev      *picchu.Revision
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
			failures := validator.failures(tc.rev)
			failedTargetsMap := map[string]bool{}
			for _, failure := range failures {
				failedTargetsMap[failure.target] = true
			}
			var failedTargets []string
			for target := range failedTargetsMap {
				failedTargets = append(failedTargets, target)
			}

			assert.ElementsMatch(tc.expected, failedTargets, failures)
		})
	}
}
