package plan

import (
	"context"
	"testing"

	testify "github.com/stretchr/testify/assert"
	ktest "go.medium.engineering/kubernetes/pkg/test"
	"go.medium.engineering/picchu/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	_ "go.medium.engineering/kubernetes/pkg/test/core/v1"
)

func TestBuildWaypointDeploymentOverlayWithoutResources(t *testing.T) {
	assert := testify.New(t)
	overlay, err := buildWaypointDeploymentOverlay("my-app", "production", nil)
	assert.NoError(err)
	// Unified service tags are stamped for the app the waypoint fronts (PLT-3301).
	assert.Contains(overlay, `ad.datadoghq.com/tags: '{"env":"production","service":"my-app"}'`)
	assert.NotContains(overlay, ddUnifiedServiceTagsPlaceholder)
	// The istio-proxy autodiscovery check and its %%host%% variable must survive rendering.
	assert.Contains(overlay, "ad.datadoghq.com/istio-proxy.checks")
	assert.Contains(overlay, "%%host%%")
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
	overlay, err := buildWaypointDeploymentOverlay("my-app", "production", resources)
	assert.NoError(err)
	assert.Contains(overlay, `ad.datadoghq.com/tags: '{"env":"production","service":"my-app"}'`)
	assert.Contains(overlay, "containers:")
	assert.Contains(overlay, "name: istio-proxy")
	assert.Contains(overlay, "requests:")
	assert.Contains(overlay, "cpu: 500m")
	assert.Contains(overlay, "memory: 128Mi")
	assert.Contains(overlay, "limits:")
	assert.Contains(overlay, "cpu: 1")
	assert.Contains(overlay, "memory: 256Mi")
}

// TestBuildWaypointDeploymentOverlayIsValidYAML round-trips the rendered overlay through the
// Deployment type (PICCHU-INV-MESH-5) and asserts the unified service tags land on the pod template.
func TestBuildWaypointDeploymentOverlayIsValidYAML(t *testing.T) {
	assert := testify.New(t)
	resources := &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("250m")},
	}
	overlay, err := buildWaypointDeploymentOverlay("my-app", "production", resources)
	assert.NoError(err)

	var deployment appsv1.Deployment
	assert.NoError(yaml.Unmarshal([]byte(overlay), &deployment))
	assert.Equal(`{"env":"production","service":"my-app"}`, deployment.Spec.Template.Annotations["ad.datadoghq.com/tags"])
	assert.Contains(deployment.Spec.Template.Annotations, "ad.datadoghq.com/istio-proxy.checks")
	assert.Len(deployment.Spec.Template.Spec.Containers, 1)
	assert.Equal("istio-proxy", deployment.Spec.Template.Spec.Containers[0].Name)
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
	overlay, err := buildWaypointDeploymentOverlay("my-app", "production", resources)
	assert.NoError(err)

	en := &EnsureWaypointOptions{
		Namespace: "namespace",
		App:       "my-app",
		Env:       "production",
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
