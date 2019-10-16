package releasemanager

import (
	tt "testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	rmplan "go.medium.engineering/picchu/pkg/controller/releasemanager/plan"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func createTestIncarnation(tag string, currentState State, currentPercent, increment int) *Incarnation {
	delaySeconds := int64(0)
	scaleMin := int32(1)
	return &Incarnation{
		tag: tag,
		revision: &picchuv1alpha1.Revision{
			Spec: picchuv1alpha1.RevisionSpec{
				Targets: []picchuv1alpha1.RevisionTarget{
					{
						Name: tag,
						Release: picchuv1alpha1.ReleaseInfo{
							Max: 100,
							Rate: picchuv1alpha1.RateInfo{
								Increment:    uint32(increment),
								DelaySeconds: &delaySeconds,
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
			log: logf.Log.WithName("controller_releasemanager_syncer_test"),
			releaseManager: &picchuv1alpha1.ReleaseManager{
				Spec: picchuv1alpha1.ReleaseManagerSpec{
					Target: tag,
				},
			},
		},
	}
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

func TestPrepareRevisionsAndRules(t *tt.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	releaseRateIncrement := 20

	deployedIncarnations := []*Incarnation{
		createTestIncarnation("deployed", released, 100, releaseRateIncrement),
	}
	unreleasableIncarnations := []*Incarnation{}
	alertableIncarnations := []*Incarnation{}
	releasableIncarnations := []*Incarnation{
		// sorted by GitTimestamp, newest first
		// note: does not add up to 100
		createTestIncarnation("test0", canarying, 10, releaseRateIncrement),
		createTestIncarnation("test1", canarying, 10, releaseRateIncrement),
		createTestIncarnation("test2", pendingrelease, 10, releaseRateIncrement),
		createTestIncarnation("test3", pendingrelease, 10, releaseRateIncrement),
		createTestIncarnation("test4", released, 40, releaseRateIncrement),
	}

	m := NewMockIncarnations(ctrl)
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
		releasable().
		Return(releasableIncarnations).
		AnyTimes()
	m.
		EXPECT().
		alertable().
		Return(alertableIncarnations).
		AnyTimes()

	testResourceSyncer := &ResourceSyncer{
		incarnations: m,
		log:          logf.Log.WithName("controller_releasemanager_syncer_test"),
	}

	// testing when revision percents don't add up to 100
	// revisions should add up after running prepareRevisionsAndRules() once
	revisions, _ := testResourceSyncer.prepareRevisionsAndRules()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{0, 0, 50, 10, 40})

	// testing "normal" test case
	revisions, _ = testResourceSyncer.prepareRevisionsAndRules()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{0, 0, 70, 10, 20})

	revisions, _ = testResourceSyncer.prepareRevisionsAndRules()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{0, 0, 90, 10, 0})

	revisions, _ = testResourceSyncer.prepareRevisionsAndRules()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{0, 0, 100, 0, 0})

	revisions, _ = testResourceSyncer.prepareRevisionsAndRules()
	assertIncarnationPercent(t, releasableIncarnations, revisions, []int{0, 0, 100, 0, 0})
}

func assertIncarnationPercent(
	t *tt.T,
	incarnations []*Incarnation,
	revisions []rmplan.Revision,
	assertPercents []int) {

	t.Logf("revisions - %v", revisions)
	logIncarnations(t, "incarnations", incarnations)

	incarnationTagMap := map[string]int{}

	for i, assertPercent := range assertPercents {
		assert.Equal(t, uint32(assertPercent), incarnations[i].status.CurrentPercent)
		incarnationTagMap[incarnations[i].tag] = assertPercent
	}

	for _, rev := range revisions {
		assertPercent := incarnationTagMap[rev.Tag]
		assert.Equal(t, uint32(assertPercent), rev.Weight)
	}
}
