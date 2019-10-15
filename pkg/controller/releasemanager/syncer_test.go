package releasemanager

import (
	tt "testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	rmplan "go.medium.engineering/picchu/pkg/controller/releasemanager/plan"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func createTestIncarnation(tag string, currentPercent uint32) *Incarnation {
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
								Increment:    20,
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
			CurrentPercent: currentPercent,
			Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{
				Current: int32(currentPercent),
				Desired: int32(currentPercent),
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

	// must be sorted by git timestamp, newest first
	// currentPercent should add up to 100
	deployedIncarnations := []*Incarnation{
		createTestIncarnation("deployed", 40),
	}
	unreleasableIncarnations := []*Incarnation{}
	releasableIncarnations := []*Incarnation{
		createTestIncarnation("test0", 10),
		createTestIncarnation("test1", 20),
		createTestIncarnation("test2", 30),
	}
	alertableIncarnations := []*Incarnation{}

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

	var revisions []rmplan.Revision
	for i := 0; i < 4; i++ {
		revisions, _ = testResourceSyncer.prepareRevisionsAndRules()
	}
	t.Logf("revisions - %v", revisions)
	logIncarnations(t, "deployed", deployedIncarnations)
	logIncarnations(t, "releasableIncarnations", releasableIncarnations)

	assert.Equal(t, uint32(100), releasableIncarnations[0].status.CurrentPercent)
	assert.Equal(t, uint32(0), releasableIncarnations[1].status.CurrentPercent)
	assert.Equal(t, uint32(0), releasableIncarnations[2].status.CurrentPercent)

	for _, rev := range revisions {
		switch rev.Tag {
		case "test0": // latest releasable
			assert.Equal(t, uint32(100), rev.Weight)
		default:
			assert.Equal(t, uint32(0), rev.Weight)
		}
	}

	// assert.Equal(t, []monitoringv1.Rule{}, rules)
}
