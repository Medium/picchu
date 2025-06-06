package controllers

import (
	ttesting "testing"
	"time"

	picchu "go.medium.engineering/picchu/api/v1alpha1"

	"github.com/stretchr/testify/assert"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIncarnation_getExternalTestStatus(t *ttesting.T) {
	testIncarnation := createTestIncarnation("test", testing, 10)
	recentLastUpdated := meta.NewTime(time.Now().Add(-5 * time.Minute))
	oldLastUpdated := meta.NewTime(time.Now().Add(-15 * time.Minute))
	timeout := meta.Duration{Duration: 10 * time.Minute}

	// ExternalTestDisabled
	assertExternalTestStatus(t, testIncarnation, picchu.ExternalTest{
		Enabled:     false,
		Timeout:     &timeout,
		Started:     false,
		Completed:   false,
		Succeeded:   false,
		LastUpdated: nil,
	}, ExternalTestDisabled, false)

	// ExternalTestPending
	assertExternalTestStatus(t, testIncarnation, picchu.ExternalTest{
		Enabled:     true,
		Timeout:     &timeout,
		Started:     false,
		Completed:   false,
		Succeeded:   false,
		LastUpdated: &recentLastUpdated,
	}, ExternalTestPending, false)

	// ExternalTestStarted
	assertExternalTestStatus(t, testIncarnation, picchu.ExternalTest{
		Enabled:     true,
		Timeout:     &timeout,
		Started:     true,
		Completed:   false,
		Succeeded:   false,
		LastUpdated: &recentLastUpdated,
	}, ExternalTestStarted, false)

	// Timing out
	assertExternalTestStatus(t, testIncarnation, picchu.ExternalTest{
		Enabled:     true,
		Timeout:     &timeout,
		Started:     true,
		Completed:   false,
		Succeeded:   false,
		LastUpdated: &oldLastUpdated,
	}, ExternalTestStarted, true)

	// ExternalTestFailed
	assertExternalTestStatus(t, testIncarnation, picchu.ExternalTest{
		Enabled:     true,
		Timeout:     &timeout,
		Started:     true,
		Completed:   true,
		Succeeded:   false,
		LastUpdated: &recentLastUpdated,
	}, ExternalTestFailed, false)

	// ExternalTestSucceeded
	assertExternalTestStatus(t, testIncarnation, picchu.ExternalTest{
		Enabled:     true,
		Timeout:     &timeout,
		Started:     true,
		Completed:   true,
		Succeeded:   true,
		LastUpdated: &oldLastUpdated,
	}, ExternalTestSucceeded, false)
}

func assertExternalTestStatus(
	t *ttesting.T,
	testIncarnation *Incarnation,
	externalTest picchu.ExternalTest,
	expectedStatus ExternalTestStatus,
	expectedTimeout bool,
) {
	testIncarnation.revision.Spec.Targets[0].ExternalTest = externalTest
	assert.Equal(t, expectedStatus, testIncarnation.getExternalTestStatus())
	assert.Equal(t, expectedTimeout, testIncarnation.isTimingOut())
}

func TestIncarnation_targetScale(t *ttesting.T) {
	testIncarnation := createTestIncarnation("test", releasing, 10)
	testIncarnation.revision.Spec.Targets[0].Release.LinearScaling.Increment = 10
	testIncarnation.status.CurrentPercent = 0
	assert.Equal(t, 1.0, testIncarnation.targetScale())
	testIncarnation.status.CurrentPercent = 10
	assert.Equal(t, 0.5, testIncarnation.targetScale())
	testIncarnation.status.CurrentPercent = 90
	assert.Equal(t, 0.9, testIncarnation.targetScale())
	testIncarnation.status.CurrentPercent = 100
	assert.Equal(t, 1.0, testIncarnation.targetScale())
}

func TestIncarnation_divideReplicas(t *ttesting.T) {
	assert := assert.New(t)

	for _, test := range []struct {
		Name             string
		ScalingStrategy  string
		LinearScaling    picchu.LinearScaling
		GeometricScaling picchu.GeometricScaling
		CurrentPercent   int
		Replicas         int32
		ExpectedReplicas int32
	}{
		{
			Name:            "TestSecondRamp",
			ScalingStrategy: picchu.ScalingStrategyGeometric,
			GeometricScaling: picchu.GeometricScaling{
				Start:  10,
				Factor: 2,
				Delay:  &meta.Duration{},
			},
			CurrentPercent:   10,
			Replicas:         30,
			ExpectedReplicas: 6,
		},
		{
			Name:            "TestThirdRamp",
			ScalingStrategy: picchu.ScalingStrategyGeometric,
			GeometricScaling: picchu.GeometricScaling{
				Start:  10,
				Factor: 2,
				Delay:  &meta.Duration{},
			},
			CurrentPercent:   20,
			Replicas:         30,
			ExpectedReplicas: 12,
		},
		{
			Name:            "TestFourthRamp",
			ScalingStrategy: picchu.ScalingStrategyGeometric,
			GeometricScaling: picchu.GeometricScaling{
				Start:  10,
				Factor: 2,
				Delay:  &meta.Duration{},
			},
			CurrentPercent:   40,
			Replicas:         30,
			ExpectedReplicas: 24,
		},
		{
			Name:            "TestFifthRamp",
			ScalingStrategy: picchu.ScalingStrategyGeometric,
			GeometricScaling: picchu.GeometricScaling{
				Start:  10,
				Factor: 2,
				Delay:  &meta.Duration{},
			},
			CurrentPercent:   80,
			Replicas:         30,
			ExpectedReplicas: 30,
		},
	} {
		t.Run(test.Name, func(t *ttesting.T) {
			i := createTestIncarnation("test", "releasing", test.CurrentPercent)
			i.revision.Spec.Targets[0].Release.ScalingStrategy = test.ScalingStrategy
			i.revision.Spec.Targets[0].Release.LinearScaling = test.LinearScaling
			i.revision.Spec.Targets[0].Release.GeometricScaling = test.GeometricScaling
			assert.Equal(test.ExpectedReplicas, i.divideReplicas(test.Replicas))
		})
	}
}

func TestIncarnation_eventDrivenDisabled(t *ttesting.T) {
	for _, test := range []struct {
		Name                string
		EventDriven         bool
		EventDrivenDisabled bool
		Expected            int32
		ScalingFactor       float64
		Count               int32
		Percent             int32
	}{
		{
			Name:                "EventDrivenTrue + EventDrivenDisabledTrue",
			EventDriven:         true,
			EventDrivenDisabled: true,
			Expected:            0,
			ScalingFactor:       0.0,
			Count:               0,
			Percent:             100,
		},
		{
			Name:                "EventDrivenFalse + EventDrivenDisabledTrue",
			EventDriven:         false,
			EventDrivenDisabled: true,
			Expected:            5,
			ScalingFactor:       1.0,
			Count:               5,
			Percent:             100,
		},
		{
			Name:                "EventDrivenFalse + EventDrivenDisabledFalse",
			EventDriven:         false,
			EventDrivenDisabled: false,
			Expected:            10,
			ScalingFactor:       1.0,
			Count:               10,
			Percent:             100,
		},
	} {
		t.Run(test.Name, func(t *ttesting.T) {
			assert := assert.New(t)
			i := &Incarnation{
				revision: &picchu.Revision{
					Spec: picchu.RevisionSpec{
						EventDriven: test.EventDriven,
					},
				},
				controller: &IncarnationController{
					clusterInfo: ClusterInfoList{
						{
							Name:          "test-cluster",
							Live:          true,
							ScalingFactor: test.ScalingFactor,
						},
					},
				},
			}
			assert.Equal(test.Expected, i.controller.expectedTotalReplicas(test.Count, test.Percent))
		})
	}
}

func TestIncarnation_divideReplicasNoAutoscale(t *ttesting.T) {
	for _, test := range []struct {
		Name         string
		Clusters     int
		ScaleDefault int32
		Expected     int32
	}{
		{
			Name:         "One Cluster, One Instance",
			Clusters:     1,
			ScaleDefault: 1,
			Expected:     1,
		},
		{
			Name:         "One Cluster, Eight Instance",
			Clusters:     1,
			ScaleDefault: 8,
			Expected:     8,
		},
		{
			Name:         "Two Cluster, Eight Instance",
			Clusters:     2,
			ScaleDefault: 8,
			Expected:     4,
		},
		{
			Name:         "Three Cluster, Eight Instance",
			Clusters:     3,
			ScaleDefault: 8,
			Expected:     3,
		},
	} {
		t.Run(test.Name, func(t *ttesting.T) {
			assert := assert.New(t)
			i := createTestIncarnation(
				"test",
				"deploying",
				0,
				&testClusters{Clusters: test.Clusters},
			)
			assert.Equal(test.Expected, i.divideReplicasNoAutoscale(test.ScaleDefault))
		})
	}
}

func Test_IsExpired(t *ttesting.T) {
	for _, test := range []struct {
		Name              string
		TTL               time.Duration
		CreationTimestamp time.Time
		Expected          bool
	}{
		{
			Name:              "expired",
			TTL:               time.Second,
			CreationTimestamp: time.Now().Add(-time.Second * 2),
			Expected:          true,
		},
		{
			Name:              "almost expired",
			TTL:               time.Second * 4,
			CreationTimestamp: time.Now().Add(-time.Second * 2),
			Expected:          true,
		},
	} {
		t.Run(test.Name, func(t *ttesting.T) {
			assert := assert.New(t)
			i := createTestIncarnation("test", "deployed", 0, &testClusters{Clusters: 1})
			i.target().Release.TTL = int64(test.TTL / time.Second)
			i.revision.CreationTimestamp = meta.Time{Time: test.CreationTimestamp}
			assert.Equal(test.Expected, i.isExpired())
		})
	}
}
