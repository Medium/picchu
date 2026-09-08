package plan

import (
	"context"
	"testing"

	testify "github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/test"
	autoscaling "k8s.io/api/autoscaling/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func applyWaypointHPA(t *testing.T, spec *picchuv1alpha1.WaypointHPASpec) (*autoscaling.HorizontalPodAutoscaler, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	log := test.MustNewLogger()
	cli := fakeClient()

	en := &EnsureWaypointHPA{Namespace: "namespace", HPA: spec}
	if err := en.Apply(ctx, cli, cluster, log); err != nil {
		return nil, err
	}

	hpa := &autoscaling.HorizontalPodAutoscaler{}
	key := types.NamespacedName{Name: waypointHPAName, Namespace: "namespace"}
	err := cli.Get(ctx, key, hpa)
	return hpa, err
}

// TestEnsureWaypointHPAScaleDownBehavior pins the scale-down damping. Without an explicit
// behavior the Kubernetes defaults permit removing 100% of the waypoint pods in one 15s step,
// and each removed pod terminates the gRPC connections routed through it.
func TestEnsureWaypointHPAScaleDownBehavior(t *testing.T) {
	assert := testify.New(t)

	hpa, err := applyWaypointHPA(t, &picchuv1alpha1.WaypointHPASpec{
		MinReplicas:                    6,
		MaxReplicas:                    100,
		TargetCPUUtilizationPercentage: 50,
	})
	assert.NoError(err)

	assert.Equal(int32(6), *hpa.Spec.MinReplicas)
	assert.Equal(int32(100), hpa.Spec.MaxReplicas)

	behavior := hpa.Spec.Behavior
	assert.NotNil(behavior)

	// ScaleUp stays on the Kubernetes defaults: scaling up terminates no
	// connections, and damping it would only slow the response to a spike.
	assert.Nil(behavior.ScaleUp)

	scaleDown := behavior.ScaleDown
	assert.NotNil(scaleDown)
	assert.Equal(int32(waypointScaleDownStabilizationSeconds), *scaleDown.StabilizationWindowSeconds)
	assert.Equal(autoscaling.MaxChangePolicySelect, *scaleDown.SelectPolicy)

	// Both policies are required. The percent policy damps at higher replica
	// counts; the pods policy keeps the HPA able to make progress at low counts,
	// where 25% rounds down to zero pods and a percent-only rule would pin the
	// waypoint above its floor indefinitely.
	assert.Len(scaleDown.Policies, 2)
	byType := map[autoscaling.HPAScalingPolicyType]autoscaling.HPAScalingPolicy{}
	for _, p := range scaleDown.Policies {
		byType[p.Type] = p
	}

	percent, ok := byType[autoscaling.PercentScalingPolicy]
	assert.True(ok, "expected a percent scale-down policy")
	assert.Equal(int32(waypointScaleDownPercent), percent.Value)
	assert.Equal(int32(waypointScaleDownPeriodSeconds), percent.PeriodSeconds)

	pods, ok := byType[autoscaling.PodsScalingPolicy]
	assert.True(ok, "expected a pods scale-down policy")
	assert.Equal(int32(waypointScaleDownPods), pods.Value)
	assert.Equal(int32(waypointScaleDownPeriodSeconds), pods.PeriodSeconds)
}

// TestEnsureWaypointHPADefaults covers the fallbacks, including PICCHU-INV-MESH-3:
// MinReplicas below 2 is raised to 2 so the waypoint PDB's minAvailable = min-1
// still allows Karpenter to evict one pod.
func TestEnsureWaypointHPADefaults(t *testing.T) {
	assert := testify.New(t)

	hpa, err := applyWaypointHPA(t, &picchuv1alpha1.WaypointHPASpec{MinReplicas: 1})
	assert.NoError(err)

	assert.Equal(int32(waypointDefaultMin), *hpa.Spec.MinReplicas)
	assert.Equal(int32(waypointDefaultMax), hpa.Spec.MaxReplicas)
	assert.Len(hpa.Spec.Metrics, 1)
	assert.Equal(int32(waypointDefaultCPU), *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)

	// Damping applies to defaulted specs too, not just explicitly tuned ones.
	assert.NotNil(hpa.Spec.Behavior)
	assert.NotNil(hpa.Spec.Behavior.ScaleDown)
	assert.Len(hpa.Spec.Behavior.ScaleDown.Policies, 2)
}

func TestEnsureWaypointHPANilSpecIsNoop(t *testing.T) {
	assert := testify.New(t)

	_, err := applyWaypointHPA(t, nil)
	assert.True(apierrors.IsNotFound(err), "expected no HPA to be created, got %v", err)
}
