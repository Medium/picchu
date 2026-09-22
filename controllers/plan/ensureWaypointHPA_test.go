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

	// ScaleUp is pinned to the literal Kubernetes defaults rather than left
	// nil. Scaling up terminates no connections, so there's nothing to damp --
	// but leaving this nil and relying on API-server defaulting reintroduces
	// PICCHU-INV-MESH-9's follow-up bug (see TestEnsureWaypointHPAIdempotent
	// AfterServerDefaulting): the object picchu constructs would never equal
	// the object actually persisted, so CreateOrUpdate would rewrite every
	// waypoint HPA fleet-wide on every ~15s reconcile forever.
	scaleUp := behavior.ScaleUp
	assert.NotNil(scaleUp)
	assert.Equal(int32(waypointScaleUpStabilizationSeconds), *scaleUp.StabilizationWindowSeconds)
	assert.Equal(autoscaling.MaxChangePolicySelect, *scaleUp.SelectPolicy)
	assert.Len(scaleUp.Policies, 2)
	scaleUpByType := map[autoscaling.HPAScalingPolicyType]autoscaling.HPAScalingPolicy{}
	for _, p := range scaleUp.Policies {
		scaleUpByType[p.Type] = p
	}
	scaleUpPods, ok := scaleUpByType[autoscaling.PodsScalingPolicy]
	assert.True(ok, "expected a pods scale-up policy")
	assert.Equal(int32(waypointScaleUpPods), scaleUpPods.Value)
	assert.Equal(int32(waypointScaleUpPeriodSeconds), scaleUpPods.PeriodSeconds)
	scaleUpPercent, ok := scaleUpByType[autoscaling.PercentScalingPolicy]
	assert.True(ok, "expected a percent scale-up policy")
	assert.Equal(int32(waypointScaleUpPercent), scaleUpPercent.Value)
	assert.Equal(int32(waypointScaleUpPeriodSeconds), scaleUpPercent.PeriodSeconds)

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

// TestEnsureWaypointHPAIdempotentAfterServerDefaulting reproduces PICCHU-INV-MESH-9's
// follow-up bug: leaving Behavior.ScaleUp nil relied on the Kubernetes API server to
// default it on write, but picchu's CreateOrUpdate (controllerutil.CreateOrUpdate) compares
// the object it just built -- ScaleUp nil -- against a DeepCopy of what Get returned --
// ScaleUp populated by that same server-side defaulting on the *previous* write. The two
// were never equal, so picchu re-issued an Update on every single reconcile forever,
// fleet-wide, racing the live HPA controller's own concurrent status writes on the same
// object ("the object has been modified; please apply your changes to the latest version
// and try again").
//
// fake.NewClientBuilder does not run real API-server admission/defaulting, so this test
// applies that defaulting by hand between the two Apply calls -- reproducing exactly what
// a real cluster does on every write -- then asserts a second reconcile with an unchanged
// spec does not touch the object at all. Before the fix (ScaleUp left nil), this test fails:
// the resourceVersion changes on every call.
func TestEnsureWaypointHPAIdempotentAfterServerDefaulting(t *testing.T) {
	assert := testify.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	log := test.MustNewLogger()
	cli := fakeClient()
	key := types.NamespacedName{Name: waypointHPAName, Namespace: "namespace"}

	spec := &picchuv1alpha1.WaypointHPASpec{
		MinReplicas:                    6,
		MaxReplicas:                    100,
		TargetCPUUtilizationPercentage: 50,
	}
	en := &EnsureWaypointHPA{Namespace: "namespace", HPA: spec}
	assert.NoError(en.Apply(ctx, cli, cluster, log))

	hpa := &autoscaling.HorizontalPodAutoscaler{}
	assert.NoError(cli.Get(ctx, key, hpa))

	// Simulate the API server's own defaulting of the side of Behavior we don't
	// otherwise set, which a fake client never does on its own.
	hpa.Spec.Behavior.ScaleUp = &autoscaling.HPAScalingRules{
		StabilizationWindowSeconds: func() *int32 { v := int32(0); return &v }(),
		SelectPolicy:               func() *autoscaling.ScalingPolicySelect { v := autoscaling.MaxChangePolicySelect; return &v }(),
		Policies: []autoscaling.HPAScalingPolicy{
			{Type: autoscaling.PodsScalingPolicy, Value: 4, PeriodSeconds: 15},
			{Type: autoscaling.PercentScalingPolicy, Value: 100, PeriodSeconds: 15},
		},
	}
	assert.NoError(cli.Update(ctx, hpa))
	resourceVersionBeforeReconcile := hpa.ResourceVersion

	// Reconciling again with the exact same spec has to be a true no-op. If it
	// isn't, picchu is fighting API-server defaulting and will rewrite this
	// object -- and every waypoint HPA fleet-wide -- on every reconcile forever.
	assert.NoError(en.Apply(ctx, cli, cluster, log))

	after := &autoscaling.HorizontalPodAutoscaler{}
	assert.NoError(cli.Get(ctx, key, after))
	assert.Equal(resourceVersionBeforeReconcile, after.ResourceVersion,
		"reconciling with an unchanged spec must not write to the HPA -- a resourceVersion "+
			"bump here means picchu is rewriting this object (and every waypoint HPA "+
			"fleet-wide) on every ~15s sync-period tick, forever")
}
