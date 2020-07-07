package revision

import (
	testify "github.com/stretchr/testify/assert"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"gomodules.xyz/jsonpatch/v2"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type revisionBuilder struct {
	*picchu.Revision
	portCounter int32
}

func (r *revisionBuilder) addPort(target string, name string, mode picchu.PortMode, dflt bool) *revisionBuilder {
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
				Default:       dflt,
			})
			return r
		}
	}
	panic("Bug")
}

func (r *revisionBuilder) ensureTarget(name string) *revisionBuilder {
	for _, target := range r.Spec.Targets {
		if target.Name == name {
			return r
		}
	}
	r.Spec.Targets = append(r.Spec.Targets, picchu.RevisionTarget{
		Name: name,
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
				addPort("production", "http", picchu.PortPrivate, false).
				addPort("production", "grpc", picchu.PortPrivate, false).
				addPort("production", "status", picchu.PortLocal, false).
				build(),
			expected: []jsonpatch.JsonPatchOperation{{
				Operation: "add",
				Path:      "/spec/targets/0/ports/0/default",
				Value:     true,
			}},
		},
		{
			name: "DontSetHTTPAsDefaultPort",
			rev: newRevisionBuilder().
				addPort("production", "http", picchu.PortPrivate, false).
				addPort("production", "grpc", picchu.PortPrivate, true).
				addPort("production", "status", picchu.PortLocal, false).
				build(),
			expected: nil,
		},
		{
			name: "SpitModes",
			rev: newRevisionBuilder().
				addPort("production", "grpc", picchu.PortPublic, false).
				addPort("production", "http", picchu.PortPublic, false).
				addPort("production", "http", picchu.PortPrivate, false).
				addPort("production", "grpc", picchu.PortPrivate, true).
				addPort("production", "status", picchu.PortLocal, false).
				build(),
			expected: []jsonpatch.JsonPatchOperation{{
				Operation: "add",
				Path:      "/spec/targets/0/ports/1/default",
				Value:     true,
			}},
		},
		{
			name: "NoHTTP",
			rev: newRevisionBuilder().
				addPort("production", "grpc", picchu.PortPublic, false).
				addPort("production", "blah", picchu.PortPublic, false).
				addPort("production", "blah", picchu.PortPrivate, false).
				addPort("production", "grpc", picchu.PortPrivate, false).
				addPort("production", "status", picchu.PortLocal, false).
				build(),
			expected: nil,
		},
		{
			name: "SinglePort",
			rev: newRevisionBuilder().
				addPort("production", "grpc", picchu.PortPublic, false).
				addPort("production", "grpc", picchu.PortPrivate, false).
				addPort("production", "status", picchu.PortLocal, false).
				build(),
			expected: []jsonpatch.JsonPatchOperation{
				{
					Operation: "add",
					Path:      "/spec/targets/0/ports/0/default",
					Value:     true,
				},
				{
					Operation: "add",
					Path:      "/spec/targets/0/ports/1/default",
					Value:     true,
				},
			},
		},
		{
			name: "SingleAndHTTP",
			rev: newRevisionBuilder().
				addPort("production", "grpc", picchu.PortPublic, false).
				addPort("production", "grpc", picchu.PortPrivate, false).
				addPort("production", "http", picchu.PortPrivate, false).
				addPort("production", "status", picchu.PortLocal, false).
				build(),
			expected: []jsonpatch.JsonPatchOperation{
				{
					Operation: "add",
					Path:      "/spec/targets/0/ports/0/default",
					Value:     true,
				},
				{
					Operation: "add",
					Path:      "/spec/targets/0/ports/2/default",
					Value:     true,
				},
			},
		},
		{
			name: "NoDefaultOnLocal",
			rev: newRevisionBuilder().
				addPort("production", "grpc", picchu.PortLocal, false).
				addPort("production", "http", picchu.PortLocal, false).
				addPort("production", "status", picchu.PortLocal, false).
				build(),
			expected: nil,
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, func(t *testing.T) {
			patches := mutator.getPatches(tc.rev)
			assert.EqualValues(tc.expected, patches)
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
				addPort("production", "http", picchu.PortPrivate, true).
				addPort("production", "grpc", picchu.PortPublic, true).
				addPort("production", "status", picchu.PortLocal, false).
				build(),
			expected: nil,
		},
		{
			name: "NoDefault",
			rev: newRevisionBuilder().
				addPort("production", "http", picchu.PortPrivate, false).
				addPort("production", "grpc", picchu.PortPrivate, false).
				addPort("production", "status", picchu.PortLocal, false).
				build(),
			expected: []string{"production"},
		},
		{
			name: "DefaultLocal",
			rev: newRevisionBuilder().
				addPort("production", "status", picchu.PortLocal, true).
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
				addPort("staging", "http", picchu.PortPrivate, true).
				addPort("staging", "grpc", picchu.PortPublic, true).
				addPort("staging", "status", picchu.PortLocal, false).
				addPort("production", "http", picchu.PortPrivate, false).
				addPort("production", "grpc", picchu.PortPrivate, false).
				addPort("production", "status", picchu.PortLocal, false).
				build(),
			expected: []string{"production"},
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, func(t *testing.T) {
			failures := validator.invalidTargets(tc.rev)
			assert.EqualValues(tc.expected, failures)
		})
	}
}
