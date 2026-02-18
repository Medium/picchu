package plan

import (
	"context"
	"testing"

	testify "github.com/stretchr/testify/assert"
	ktest "go.medium.engineering/kubernetes/pkg/test"
	picchu "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/test"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	_ "go.medium.engineering/kubernetes/pkg/test/core/v1"
)

var (
	namespace = &core.Namespace{
		ObjectMeta: meta.ObjectMeta{
			Name: "namespace",
			Labels: map[string]string{
				"istio-injection":     "enabled",
				picchu.LabelOwnerName: "",
				picchu.LabelOwnerType: "",
			},
		},
	}
)

func TestCreatesNamespace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	scheme := runtime.NewScheme()
	assert.NoError(core.AddToScheme(scheme))
	cli := fakeClient()

	en := &EnsureNamespace{
		Name: "namespace",
	}
	assert.NoError(en.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertMatch(ctx, t, cli, namespace)
}

func TestIgnoreNamespace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	scheme := runtime.NewScheme()
	assert.NoError(core.AddToScheme(scheme))
	cli := fakeClient(namespace)

	en := &EnsureNamespace{
		Name: "namespace",
	}
	assert.NoError(en.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertMatch(ctx, t, cli, namespace)
}

func TestUpdatesNamespace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	copy := namespace.DeepCopy()
	copy.Labels[picchu.LabelOwnerName] = "bob"
	cli := fakeClient(copy)

	en := &EnsureNamespace{
		Name: "namespace",
	}
	assert.NoError(en.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertMatch(ctx, t, cli, namespace)
}

func TestEnsureNamespaceWithAmbientMesh(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	scheme := runtime.NewScheme()
	assert.NoError(core.AddToScheme(scheme))
	cli := fakeClient()

	en := &EnsureNamespace{
		Name:        "namespace",
		OwnerName:   "rm",
		OwnerType:   picchu.OwnerReleaseManager,
		AmbientMesh: true,
	}
	assert.NoError(en.Apply(ctx, cli, cluster, log), "Shouldn't return error.")

	expected := &core.Namespace{
		ObjectMeta: meta.ObjectMeta{
			Name: "namespace",
			Labels: map[string]string{
				"istio.io/dataplane-mode": "ambient",
				picchu.LabelOwnerName:     "rm",
				picchu.LabelOwnerType:     picchu.OwnerReleaseManager,
			},
		},
	}
	ktest.AssertMatch(ctx, t, cli, expected)
}

func TestEnsureNamespaceTransitionToSidecar(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	scheme := runtime.NewScheme()
	assert.NoError(core.AddToScheme(scheme))
	cli := fakeClient()

	en := &EnsureNamespace{
		Name:                "namespace",
		OwnerName:           "rm",
		OwnerType:           picchu.OwnerReleaseManager,
		AmbientMesh:         false,
		TransitionToSidecar: true,
	}
	assert.NoError(en.Apply(ctx, cli, cluster, log), "Shouldn't return error.")

	expected := &core.Namespace{
		ObjectMeta: meta.ObjectMeta{
			Name: "namespace",
			Labels: map[string]string{
				"istio-injection":         "enabled",
				"istio.io/dataplane-mode": "ambient",
				picchu.LabelOwnerName:     "rm",
				picchu.LabelOwnerType:     picchu.OwnerReleaseManager,
			},
		},
	}
	ktest.AssertMatch(ctx, t, cli, expected)
}
