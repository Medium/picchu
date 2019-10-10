package releasemanager

import (
	tt "testing"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	rmplan "go.medium.engineering/picchu/pkg/controller/releasemanager/plan"
)

func TestPrepareRevisionsAndRules(t *tt.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockIncarnations(ctrl)
	m.
		EXPECT().
		deployed().
		Return([]*Incarnation{}).
		AnyTimes()

	testResourceSyncer := &ResourceSyncer{incarnations: m}
	revisions, rules := testResourceSyncer.prepareRevisionsAndRules()

	assert.Equal(t, []rmplan.Revision{}, revisions)
	assert.Equal(t, []monitoringv1.Rule{}, rules)
}
