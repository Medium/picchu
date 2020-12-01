package releasemanager

import (
	"fmt"
	"math"
	tt "testing"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	rmplan "go.medium.engineering/picchu/pkg/controller/releasemanager/plan"
	"go.medium.engineering/picchu/pkg/test"
)

const (
	testCanaryIncrement        uint32 = 10
	testReleaseIncrement       uint32 = 20
	testReleaseScalingStrategy        = picchuv1alpha1.ScalingStrategyLinear
)

type testIncarnationOption interface {
	Apply(i *Incarnation, percent int)
}

type testScale struct {
	Current int32
	Desired int32
}

func (s testScale) Apply(i *Incarnation, currentPercent int) {
	i.status.Scale.Current = s.Current
	i.status.Scale.Desired = s.Desired
	*i.target().Scale.Min = int32(math.Ceil(float64(s.Desired) / (float64(currentPercent) / 100.0)))
}

type testClusters struct {
	Clusters int
}

func (t testClusters) Apply(i *Incarnation, currentPercent int) {
	if t.Clusters > 0 {
		var clusters []ClusterInfo
		for i := 0; i < t.Clusters; i++ {
			clusters = append(clusters, ClusterInfo{
				Name:          fmt.Sprintf("cluster-%d", i),
				Live:          true,
				ScalingFactor: 1.0,
			})
		}
		i.controller = &IncarnationController{
			clusterInfo: clusters,
			log:         test.MustNewLogger(),
			releaseManager: &picchuv1alpha1.ReleaseManager{
				Spec: picchuv1alpha1.ReleaseManagerSpec{
					Target: i.tag,
				},
			},
		}
	}
}

func createTestIncarnation(tag string, currentState State, currentPercent int, options ...testIncarnationOption) *Incarnation {
	scaleMin := int32(1)
	incarnation := &Incarnation{
		tag: tag,
		log: test.MustNewLogger(),
		revision: &picchuv1alpha1.Revision{
			Spec: picchuv1alpha1.RevisionSpec{
				Targets: []picchuv1alpha1.RevisionTarget{
					{
						Name: tag,
						Canary: picchuv1alpha1.Canary{
							Percent: testCanaryIncrement,
						},
						Release: picchuv1alpha1.ReleaseInfo{
							ScalingStrategy: testReleaseScalingStrategy,
							Max:             100,
							LinearScaling: picchuv1alpha1.LinearScaling{
								Increment: testReleaseIncrement,
								Delay:     &meta.Duration{},
							},
						},
						Scale: picchuv1alpha1.ScaleInfo{
							Min: &scaleMin,
						},
					},
				},
			},
		},
		status: &picchuv1alpha1.ReleaseManagerRevisionStatus{
			// GitTimestamp:   &metav1.Time{gitTimestamp},
			CurrentPercent: uint32(currentPercent),
			Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{
				Current: int32(currentPercent),
				Desired: int32(currentPercent),
			},
			State: picchuv1alpha1.ReleaseManagerRevisionStateStatus{
				Current: string(currentState),
			},
		},
		controller: &IncarnationController{
			clusterInfo: ClusterInfoList{{
				Name:          "cluster-0",
				Live:          true,
				ScalingFactor: 1.0,
			}},
			log: test.MustNewLogger(),
			releaseManager: &picchuv1alpha1.ReleaseManager{
				Spec: picchuv1alpha1.ReleaseManagerSpec{
					Target: tag,
				},
			},
		},
		isRamping: currentState == releasing,
	}

	for _, opt := range options {
		opt.Apply(incarnation, currentPercent)
	}

	return incarnation
}

func logIncarnations(t *tt.T, name string, incarnations []*Incarnation) {
	t.Log(name)
	for _, i := range incarnations {
		t.Logf(
			"%s - {CurrentPercent: %v}",
			i.tag,
			i.status.CurrentPercent,
		)
	}
}

func createTestIncarnations(ctrl *gomock.Controller) (m *MockIncarnations) {
	deployedIncarnations := []*Incarnation{
		createTestIncarnation("deployed", released, 100),
	}
	var unreleasableIncarnations []*Incarnation
	var alertableIncarnations []*Incarnation

	m = NewMockIncarnations(ctrl)
	m.
		EXPECT().
		deployed().
		Return(deployedIncarnations).
		AnyTimes()
	m.
		EXPECT().
		unreleasable().
		Return(unreleasableIncarnations).
		AnyTimes()
	m.
		EXPECT().
		alertable().
		Return(alertableIncarnations).
		AnyTimes()

	return
}

func assertIncarnationPercent(
	t *tt.T,
	incarnations []*Incarnation,
	revisions []rmplan.Revision,
	assertPercents []int,
) {
	t.Logf("expected - %v", assertPercents)
	t.Logf("revisions - %v", revisions)
	logIncarnations(t, "incarnations", incarnations)

	incarnationTagMap := map[string]int{}

	for i, assertPercent := range assertPercents {
		assert.Equal(t, assertPercent, int(incarnations[i].status.CurrentPercent))
		incarnationTagMap[incarnations[i].tag] = assertPercent
	}

	for _, rev := range revisions {
		assertPercent := incarnationTagMap[rev.Tag]
		assert.Equal(t, assertPercent, int(rev.Weight))
	}
}

func TestPrepareRevisionsAndRulesBadAddition(t *tt.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := createTestIncarnations(ctrl)
	testResourceSyncer := &ResourceSyncer{
		incarnations: m,
		log:          test.MustNewLogger(),
	}

	releasableIncarnations := []*Incarnation{
		// sorted by GitTimestamp, newest first
		// note: does not add up to 100
		createTestIncarnation("test1 incarnation0", canarying, 10),
		createTestIncarnation("test1 incarnation1", canarying, 10),
		createTestIncarnation("test1 incarnation2", releasing, 10),
		createTestIncarnation("test1 incarnation3", releasing, 10),
		createTestIncarnation("test1 incarnation4", released, 40),
	}
	m.
		EXPECT().
		releasable().
		Return(releasableIncarnations).
		AnyTimes()

	// testing when revision percents don't add up to 100
	// revisions should add up after running prepareRevisions() once
	revisions := testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{10, 10, 30, 10, 40})

	// testing "normal" test case
	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{10, 10, 50, 10, 20})

	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{10, 10, 70, 10, 0})

	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{10, 10, 80, 0, 0})

	// canary will end on it's own
	// will stop getting returned from releasable() when it transitions to canaried
	// which happens in the state machine after ttl expires
	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{10, 10, 80, 0, 0})
}

func TestPrepareRevisionsAndRulesNormalCase(t *tt.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := createTestIncarnations(ctrl)
	testResourceSyncer := &ResourceSyncer{
		incarnations: m,
		log:          test.MustNewLogger(),
	}

	releasableIncarnations := []*Incarnation{
		createTestIncarnation("test2 incarnation0", canarying, 10),
		createTestIncarnation("test2 incarnation1", releasing, 10),
		createTestIncarnation("test2 incarnation2", releasing, 50),
		createTestIncarnation("test2 incarnation3", released, 30),
	}
	m.
		EXPECT().
		releasable().
		Return(releasableIncarnations).
		AnyTimes()

	revisions := testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{10, 30, 50, 10})

	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{10, 50, 40, 0})

	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{10, 70, 20, 0})

	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{10, 90, 0, 0})

	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{10, 90, 0, 0})
}

func TestPrepareRevisionsAndRulesIllegalStates(t *tt.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := createTestIncarnations(ctrl)
	testResourceSyncer := &ResourceSyncer{
		incarnations: m,
		log:          test.MustNewLogger(),
	}

	releasableIncarnations := []*Incarnation{
		createTestIncarnation("test3 incarnation0", releasing, 10),
		createTestIncarnation("test3 incarnation1", canaried, 10), // illegal state
		createTestIncarnation("test3 incarnation2", canarying, 10),
		createTestIncarnation("test3 incarnation3", pendingrelease, 10), // illegal state
		createTestIncarnation("test3 incarnation4", releasing, 20),
		createTestIncarnation("test3 incarnation5", released, 30),
		createTestIncarnation("test3 incarnation6", retiring, 10),
	}
	m.
		EXPECT().
		releasable().
		Return(releasableIncarnations).
		AnyTimes()

	revisions := testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{30, 10, 10, 10, 20, 20, 0})

	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{50, 10, 10, 10, 20, 0, 0})

	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{70, 10, 10, 10, 0, 0, 0})

	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{90, 10, 0, 0, 0, 0, 0})

	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{100, 0, 0, 0, 0, 0, 0})
}

func TestPrepareRevisionsAndRulesIncompleteScaleUp(t *tt.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := createTestIncarnations(ctrl)
	testResourceSyncer := &ResourceSyncer{
		incarnations: m,
		log:          test.MustNewLogger(),
	}

	scale := &testScale{Desired: 12, Current: 4}

	releasableIncarnations := []*Incarnation{
		createTestIncarnation("new", releasing, 20, scale),
		createTestIncarnation("existing", released, 80, scale),
	}
	m.
		EXPECT().
		releasable().
		Return(releasableIncarnations).
		AnyTimes()

	revisions := testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{20, 80})

	releasableIncarnations[0].status.Scale.Current = 24
	releasableIncarnations[0].status.Scale.Desired = 24

	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{40, 60})

	releasableIncarnations[0].status.Scale.Current = 36
	releasableIncarnations[0].status.Scale.Desired = 36

	revisions = testResourceSyncer.prepareRevisions()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{60, 40})
}
