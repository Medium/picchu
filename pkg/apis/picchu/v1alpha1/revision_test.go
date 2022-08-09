package v1alpha1

import (
	"testing"
	"time"

	istio "istio.io/api/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFailRevision(t *testing.T) {
	r := Revision{}
	r.Fail()
	failedAt, ok := r.Annotations[AnnotationFailedAt]
	assert.True(t, ok, "FailedAt not set")
	d := r.SinceFailed()
	assert.NotEqual(t, d, time.Duration(0), "SinceFailed reports 0")
	r.Fail()
	failedAtNow := r.Annotations[AnnotationFailedAt]
	assert.Equal(t, failedAt, failedAtNow, "FailedAt shouldn't update on subsequent calls")
}

func TestExternalTestPending(t *testing.T) {
	target := &RevisionTarget{}
	assert.False(t, target.IsExternalTestPending())

	target.ExternalTest.Enabled = true
	assert.True(t, target.IsExternalTestPending())

	target.ExternalTest.Completed = true
	assert.False(t, target.IsExternalTestPending())
}

func TestExternalTestFailed(t *testing.T) {
	target := &RevisionTarget{}
	target.ExternalTest.Enabled = true

	target.ExternalTest.Completed = true
	assert.False(t, target.IsExternalTestSuccessful())

	target.ExternalTest.Succeeded = true
	assert.True(t, target.IsExternalTestSuccessful())
}

func TestCanaryTestPending(t *testing.T) {
	target := &RevisionTarget{}
	dt := metav1.Time{}
	assert.False(t, target.IsCanaryPending(&dt))

	target.Canary.Percent = 1
	assert.False(t, target.IsCanaryPending(&dt))

	target.Canary.TTL = 1
	now := metav1.Now()
	assert.True(t, target.IsCanaryPending(&now))

	lastSecond := metav1.Time{Time: time.Now().Add(-time.Second)}
	assert.False(t, target.IsCanaryPending(&lastSecond))
}

func TestTrafficPolicyDeepCopy(t *testing.T) {
	assert := assert.New(t)
	r := &Revision{
		Spec: RevisionSpec{
			Targets: []RevisionTarget{
				{
					Istio: &Istio{TrafficPolicy: &istio.TrafficPolicy{
						PortLevelSettings: []*istio.TrafficPolicy_PortTrafficPolicy{
							{
								Port: &istio.PortSelector{Number: 8282},
								ConnectionPool: &istio.ConnectionPoolSettings{
									Http: &istio.ConnectionPoolSettings_HTTPSettings{
										Http2MaxRequests: 10,
									},
								},
							},
						},
					}},
				},
			},
		},
	}
	o := r.DeepCopy()
	portPolicy := o.Spec.Targets[0].Istio.TrafficPolicy.PortLevelSettings[0]
	assert.EqualValues(8282, portPolicy.Port.Number)
	assert.EqualValues(10, portPolicy.ConnectionPool.Http.Http2MaxRequests)
}

func TestRevisionDefaults(t *testing.T) {
	assert := assert.New(t)
	r := &Revision{
		Spec: RevisionSpec{
			Targets: []RevisionTarget{{}},
		},
	}
	scheme := runtime.NewScheme()
	AddToScheme(scheme)
	RegisterDefaults(scheme)
	scheme.Default(r)
	assert.EqualValues(defaultReleaseGcTTLSeconds, r.Spec.Targets[0].Release.TTL)
}

func TestScaleInfo_HasAutoscaler(t *testing.T) {
	for _, test := range []struct {
		Name                              string
		TargetCPUUtilizationPercentage    *int32
		TargetMemoryUtilizationPercentage *int32
		TargetRequestsRate                *string
		Worker                            *WorkerScaleInfo
		Expected                          bool
	}{
		{
			Name:     "No Autoscaler Defined",
			Expected: false,
		},
		{
			Name:                           "CPU Utilization Defined",
			TargetCPUUtilizationPercentage: pointer.Int32Ptr(90),
			Expected:                       true,
		},
		{
			Name:                              "Memory Utilization Defined",
			TargetMemoryUtilizationPercentage: pointer.Int32Ptr(90),
			Expected:                          true,
		},
		{
			Name:               "Requests Rate Defined",
			TargetRequestsRate: pointer.StringPtr("1"),
			Expected:           true,
		},
		{
			Name:     "Worker Defined",
			Worker:   &WorkerScaleInfo{},
			Expected: true,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			assert := assert.New(t)
			s := &ScaleInfo{
				TargetCPUUtilizationPercentage:    test.TargetCPUUtilizationPercentage,
				TargetMemoryUtilizationPercentage: test.TargetMemoryUtilizationPercentage,
				TargetRequestsRate:                test.TargetRequestsRate,
				Worker:                            test.Worker,
			}
			assert.Equal(test.Expected, s.HasAutoscaler())
		})
	}
}
