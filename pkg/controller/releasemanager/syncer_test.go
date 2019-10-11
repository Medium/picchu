package releasemanager

import (
	tt "testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func createTestIncarnation(tag string, currentPercent uint32) *Incarnation {
	return &Incarnation{
		tag: tag,
		status: &picchuv1alpha1.ReleaseManagerRevisionStatus{
			// GitTimestamp:   &metav1.Time{gitTimestamp},
			CurrentPercent: currentPercent,
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
		createTestIncarnation("deployed", 70),
	}
	unreleasableIncarnations := []*Incarnation{}
	releasableIncarnations := []*Incarnation{
		createTestIncarnation("test0", 0),
		createTestIncarnation("test1", 10),
		createTestIncarnation("test2", 20),
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

	revisions, _ := testResourceSyncer.prepareRevisionsAndRules()
	t.Logf("revisions - %v", revisions)
	logIncarnations(t, "deployed", deployedIncarnations)
	logIncarnations(t, "releasableIncarnations", releasableIncarnations)

	assert.Equal(t, uint32(100), releasableIncarnations[0].status.CurrentPercent)
	assert.Equal(t, uint32(0), releasableIncarnations[1].status.CurrentPercent)
	assert.Equal(t, uint32(0), releasableIncarnations[2].status.CurrentPercent)

	assert.Equal(t, uint32(0), revisions[0].Weight)   // deployed
	assert.Equal(t, uint32(100), revisions[1].Weight) // latest releasable
	assert.Equal(t, uint32(0), revisions[2].Weight)   // old releasable
	assert.Equal(t, uint32(0), revisions[3].Weight)   // old releasable

	// assert.Equal(t, []monitoringv1.Rule{}, rules)
}
