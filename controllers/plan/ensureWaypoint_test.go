package plan

import (
	"context"
	"strings"
	"testing"

	testify "github.com/stretchr/testify/assert"
	ktest "go.medium.engineering/kubernetes/pkg/test"
	"go.medium.engineering/picchu/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "go.medium.engineering/kubernetes/pkg/test/core/v1"
)

func TestBuildWaypointDeploymentOverlayWithoutResources(t *testing.T) {
	assert := testify.New(t)
	overlay, err := buildWaypointDeploymentOverlay(nil)
	assert.NoError(err)
	assert.Equal(waypointDeploymentOverlayBase, overlay)
}

func TestBuildWaypointDeploymentOverlayWithResources(t *testing.T) {
	assert := testify.New(t)
	resources := &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}
	overlay, err := buildWaypointDeploymentOverlay(resources)
	assert.NoError(err)
	assert.True(strings.HasPrefix(overlay, waypointDeploymentOverlayBase))
	assert.Contains(overlay, "containers:")
	assert.Contains(overlay, "name: istio-proxy")
	assert.Contains(overlay, "requests:")
	assert.Contains(overlay, "cpu: 500m")
	assert.Contains(overlay, "memory: 128Mi")
	assert.Contains(overlay, "limits:")
	assert.Contains(overlay, "cpu: 1")
	assert.Contains(overlay, "memory: 256Mi")
}

func TestEnsureWaypointOptionsWithResources(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	cli := fakeClient()

	resources := &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("250m"),
		},
	}
	overlay, err := buildWaypointDeploymentOverlay(resources)
	assert.NoError(err)

	en := &EnsureWaypointOptions{
		Namespace: "namespace",
		Resources: resources,
	}
	assert.NoError(en.Apply(ctx, cli, cluster, log))

	expected := &corev1.ConfigMap{
		ObjectMeta: meta.ObjectMeta{
			Name:      waypointOptionsConfigMap,
			Namespace: "namespace",
		},
		Data: map[string]string{
			"deployment": overlay,
		},
	}
	ktest.AssertMatch(ctx, t, cli, expected)
}
