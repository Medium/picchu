package controllers

import (
	ttesting "testing"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"

	"github.com/stretchr/testify/assert"
)

// testReleaseManagerStatus sets ReleaseManager.Status.Revisions (must run after testClusters).
type testReleaseManagerStatus struct {
	Revisions []picchuv1alpha1.ReleaseManagerRevisionStatus
}

func (s testReleaseManagerStatus) Apply(i *Incarnation, currentPercent int) {
	i.controller.getReleaseManager().Status.Revisions = s.Revisions
}

// testTargetRequestsRate enables Scale.HasAutoscaler() for proactive min path.
type testTargetRequestsRate struct {
	Rate string
}

func (s testTargetRequestsRate) Apply(i *Incarnation, currentPercent int) {
	i.revision.Spec.Targets[0].Scale.TargetRequestsRate = &s.Rate
}

func TestComputeFleetReplicasRequiredForRamp_normalizedOtherRevisions(t *ttesting.T) {
	// 4 live clusters @ 1.0 scaling; other revision at 90% / 36 fleet pods => baseCapacity 40.
	// PeakPercent < 100 so we use normalized capacity (not the "unscaled from 100%" branch).
	// createTestIncarnation uses linear increment 20%, so next traffic step from 10% is 30%.
	// expectedTotalReplicas(40, 30) => 12 fleet; ceil(12 * 0.9) = 11.
	i := createTestIncarnation(
		"new",
		releasing,
		10,
		testClusters{Clusters: 4},
		testReleaseManagerStatus{
			Revisions: []picchuv1alpha1.ReleaseManagerRevisionStatus{
				{Tag: "old", CurrentPercent: 90, PeakPercent: 90, Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{Current: 36}},
			},
		},
	)

	sta := ScalableTargetAdapter{Incarnation: *i}
	c, ok := sta.computeFleetReplicasRequiredForRamp(30)
	assert.True(t, ok)
	assert.NotNil(t, c)
	assert.EqualValues(t, 11, c.expectedReplicas)
	assert.EqualValues(t, 36, c.totalPods)
	assert.EqualValues(t, 90, c.totalTrafficPercent)
	assert.EqualValues(t, 40, c.baseCapacity)
}

func TestComputeFleetReplicasRequiredForRamp_noOtherRevisionsUsesScaleMin(t *ttesting.T) {
	i := createTestIncarnation("new", releasing, 10, testClusters{Clusters: 4})

	sta := ScalableTargetAdapter{Incarnation: *i}
	c, ok := sta.computeFleetReplicasRequiredForRamp(20)
	assert.True(t, ok)
	assert.NotNil(t, c)
	// baseCapacity = Scale.Min = 1; expectedTotalReplicas spreads 1 @ 20% then threshold.
	assert.EqualValues(t, 1, c.baseCapacity)
	assert.Greater(t, c.expectedReplicas, int32(0))
}

func TestComputeFleetReplicasRequiredForRamp_nilScaleMin(t *ttesting.T) {
	i := createTestIncarnation("new", releasing, 10)
	i.revision.Spec.Targets[0].Scale.Min = nil

	sta := ScalableTargetAdapter{Incarnation: *i}
	c, ok := sta.computeFleetReplicasRequiredForRamp(20)
	assert.False(t, ok)
	assert.Nil(t, c)
}

func TestCanRampTo_enoughReplicas(t *ttesting.T) {
	i := createTestIncarnation(
		"new",
		releasing,
		10,
		testClusters{Clusters: 4},
		testReleaseManagerStatus{
			Revisions: []picchuv1alpha1.ReleaseManagerRevisionStatus{
				{Tag: "old", CurrentPercent: 90, PeakPercent: 90, Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{Current: 36}},
			},
		},
	)
	i.status.Scale.Current = 11

	sta := ScalableTargetAdapter{Incarnation: *i}
	assert.True(t, sta.CanRampTo(30))
}

func TestCanRampTo_notEnoughReplicas(t *ttesting.T) {
	i := createTestIncarnation(
		"new",
		releasing,
		10,
		testClusters{Clusters: 4},
		testReleaseManagerStatus{
			Revisions: []picchuv1alpha1.ReleaseManagerRevisionStatus{
				{Tag: "old", CurrentPercent: 90, PeakPercent: 90, Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{Current: 36}},
			},
		},
	)
	i.status.Scale.Current = 4

	sta := ScalableTargetAdapter{Incarnation: *i}
	assert.False(t, sta.CanRampTo(30))
}

func TestProactiveRampMinReplicas_raisesMinToMeetFleetRequirement(t *ttesting.T) {
	i := createTestIncarnation(
		"new",
		releasing,
		10,
		testClusters{Clusters: 4},
		testTargetRequestsRate{Rate: "12"},
		testReleaseManagerStatus{
			Revisions: []picchuv1alpha1.ReleaseManagerRevisionStatus{
				{Tag: "old", CurrentPercent: 90, PeakPercent: 90, Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{Current: 36}},
			},
		},
	)
	i.revision.Spec.Targets[0].Scale.Max = 25

	// NextIncrement = 30%; fleet required = 11; per-cluster min = divideReplicas(11, 100) = 3 on 4 clusters.
	assert.EqualValues(t, 3, i.proactiveRampMinReplicas())
}

func TestProactiveRampMinReplicas_cappedByPerClusterMax(t *ttesting.T) {
	i := createTestIncarnation(
		"new",
		releasing,
		10,
		testClusters{Clusters: 4},
		testTargetRequestsRate{Rate: "12"},
		testReleaseManagerStatus{
			Revisions: []picchuv1alpha1.ReleaseManagerRevisionStatus{
				{Tag: "old", CurrentPercent: 90, PeakPercent: 90, Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{Current: 36}},
			},
		},
	)
	i.revision.Spec.Targets[0].Scale.Max = 1

	// proactive wants 3 per cluster but max divides to 1 per cluster.
	assert.EqualValues(t, 1, i.proactiveRampMinReplicas())
}

func TestProactiveRampMinReplicas_skipsWhenNotRamping(t *ttesting.T) {
	i := createTestIncarnation(
		"new",
		released,
		100,
		testClusters{Clusters: 4},
		testTargetRequestsRate{Rate: "12"},
		testReleaseManagerStatus{
			Revisions: []picchuv1alpha1.ReleaseManagerRevisionStatus{
				{Tag: "old", CurrentPercent: 0, Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{}},
			},
		},
	)
	assert.EqualValues(t, 0, i.proactiveRampMinReplicas())
}

func TestProactiveRampMinReplicas_skipsWorker(t *ttesting.T) {
	i := createTestIncarnation(
		"new",
		releasing,
		10,
		testClusters{Clusters: 4},
		testReleaseManagerStatus{
			Revisions: []picchuv1alpha1.ReleaseManagerRevisionStatus{
				{Tag: "old", CurrentPercent: 90, PeakPercent: 90, Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{Current: 36}},
			},
		},
	)
	i.revision.Spec.Targets[0].Scale.Worker = &picchuv1alpha1.WorkerScaleInfo{QueueURI: "q"}

	assert.EqualValues(t, 0, i.proactiveRampMinReplicas())
}

func TestProactiveRampMinReplicas_skipsWhenReleasedEvenIfIsRamping(t *ttesting.T) {
	// Simulates newIncarnationCollection marking isRamping on the old released revision when a
	// newer revision is canarying (first non-canary in releasable order).
	i := createTestIncarnation(
		"old",
		released,
		99,
		testClusters{Clusters: 4},
		testTargetRequestsRate{Rate: "12"},
		testReleaseManagerStatus{
			Revisions: []picchuv1alpha1.ReleaseManagerRevisionStatus{
				{Tag: "new", CurrentPercent: 1, PeakPercent: 1, Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{Current: 4}},
			},
		},
	)
	i.isRamping = true

	assert.EqualValues(t, 0, i.proactiveRampMinReplicas())
}

func TestProactiveRampMinReplicas_skipsWithoutAutoscaler(t *ttesting.T) {
	i := createTestIncarnation(
		"new",
		releasing,
		10,
		testClusters{Clusters: 4},
		testReleaseManagerStatus{
			Revisions: []picchuv1alpha1.ReleaseManagerRevisionStatus{
				{Tag: "old", CurrentPercent: 90, PeakPercent: 90, Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{Current: 36}},
			},
		},
	)
	// No CPU/memory/RPS/worker — HasAutoscaler() false.
	assert.EqualValues(t, 0, i.proactiveRampMinReplicas())
}
