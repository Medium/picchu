package prometheus

import (
	"bytes"
	"context"
	"testing"
	"text/template"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"go.medium.engineering/picchu/prometheus/mocks"
	"go.uber.org/mock/gomock"
)

func TestPrometheusCache(t *testing.T) {
	canaryTemplate := CanaryFiringTemplate
	testAlertCache(t, *canaryTemplate, true)

	sloTemplate := SLOFiringTemplate
	testAlertCache(t, *sloTemplate, false)
}

func TestPrometheusAlerts(t *testing.T) {
	canaryTemplate := CanaryFiringTemplate
	testAlert(t, *canaryTemplate, true)

	sloTemplate := SLOFiringTemplate
	testAlert(t, *sloTemplate, false)
}

func testAlertCache(t *testing.T, template template.Template, canariesOnly bool) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	m := mocks.NewMockPromAPI(ctrl)
	api := InjectAPI(m, time.Duration(25)*time.Millisecond)

	aq := NewAlertQuery("tutu", "v1")

	q := bytes.NewBufferString("")
	assert.Nil(t, template.Execute(q, aq), "Template execute shouldn't fail")

	m.
		EXPECT().
		Query(gomock.Any(), gomock.Eq(q.String()), gomock.Any()).
		Return(model.Vector{}, nil, nil).
		Times(2)

	for i := 0; i < 5; i++ {
		r, err := api.TaggedAlerts(context.TODO(), aq, time.Now(), canariesOnly)
		assert.Nil(t, err, "Should succeed in querying alerts")
		assert.Equal(t, map[string][]string{}, r, "Should get no firing alerts")
	}
	time.Sleep(time.Duration(25) * time.Millisecond)
	r, err := api.TaggedAlerts(context.TODO(), aq, time.Now(), canariesOnly)
	assert.Nil(t, err, "Should succeed in querying alerts")
	assert.Equal(t, map[string][]string{}, r, "Should get no firing alerts")
}

func TestIsRevisionTriggered(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	makeAPI := func(samples model.Vector) *API {
		m := mocks.NewMockPromAPI(ctrl)
		m.EXPECT().
			Query(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(samples, nil, nil).
			AnyTimes()
		return InjectAPI(m, time.Duration(0))
	}

	aq := NewAlertQuery("tutu", "v1")

	// canariesOnly=false, shared PSL fires with no tag label → rollback triggered via tags[""] fallback
	t.Run("SLO_untagged_alert_triggers_rollback", func(t *testing.T) {
		api := makeAPI(model.Vector{
			&model.Sample{Metric: model.Metric{"app": "tutu", "alertname": "SLOBurnRateHigh"}},
		})
		triggered, alerts, err := api.IsRevisionTriggered(context.TODO(), aq.App, aq.Tag, false)
		assert.NoError(t, err)
		assert.True(t, triggered)
		assert.Equal(t, []string{"SLOBurnRateHigh"}, alerts)
	})

	// canariesOnly=true, untagged alert must not trigger rollback
	t.Run("canary_mode_untagged_alert_ignored", func(t *testing.T) {
		api := makeAPI(model.Vector{
			&model.Sample{Metric: model.Metric{"app": "tutu", "alertname": "SLOBurnRateHigh"}},
		})
		triggered, alerts, err := api.IsRevisionTriggered(context.TODO(), aq.App, aq.Tag, true)
		assert.NoError(t, err)
		assert.False(t, triggered)
		assert.Nil(t, alerts)
	})

	// canariesOnly=false, tagged canary alert takes priority over untagged SLO alert
	t.Run("tagged_alert_takes_priority_over_untagged", func(t *testing.T) {
		api := makeAPI(model.Vector{
			&model.Sample{Metric: model.Metric{"app": "tutu", "tag": "v1", "alertname": "CanaryError", "canary": "true"}},
			&model.Sample{Metric: model.Metric{"app": "tutu", "alertname": "SLOBurnRateHigh"}},
		})
		triggered, alerts, err := api.IsRevisionTriggered(context.TODO(), aq.App, aq.Tag, false)
		assert.NoError(t, err)
		assert.True(t, triggered)
		assert.Equal(t, []string{"CanaryError"}, alerts)
	})

	// canariesOnly=false, no alerts firing → no rollback
	t.Run("no_alerts_no_rollback", func(t *testing.T) {
		api := makeAPI(model.Vector{})
		triggered, alerts, err := api.IsRevisionTriggered(context.TODO(), aq.App, aq.Tag, false)
		assert.NoError(t, err)
		assert.False(t, triggered)
		assert.Nil(t, alerts)
	})
}

func testAlert(t *testing.T, template template.Template, canariesOnly bool) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	m := mocks.NewMockPromAPI(ctrl)
	api := InjectAPI(m, time.Duration(25)*time.Millisecond)

	aq := NewAlertQuery("tutu", "v1")

	q := bytes.NewBufferString("")
	assert.Nil(t, template.Execute(q, aq), "Template execute shouldn't fail")

	response := model.Vector{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"app":       "tutu",
				"tag":       "v1",
				"alertname": "test",
			},
		},
		// No tag label: excluded in canary mode; included in SLO mode (shared tag-agnostic PSL).
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"app":       "tutu",
				"xtag":      "v2",
				"alertname": "test",
			},
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"xapp":      "tutu",
				"tag":       "v3",
				"alertname": "test",
			},
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"app":       "tutux",
				"tag":       "v4",
				"alertname": "test",
			},
		},
	}

	m.
		EXPECT().
		Query(gomock.Any(), gomock.Eq(q.String()), gomock.Any()).
		Return(response, nil, nil).
		Times(1)

	r, err := api.TaggedAlerts(context.TODO(), aq, time.Now(), canariesOnly)
	assert.Nil(t, err, "Should succeed in querying alerts")

	if canariesOnly {
		// Canary mode requires a tag label; empty-tag samples are excluded.
		assert.Equal(t, map[string][]string{"v1": {"test"}}, r, "Should get 1 firing alert (v1)")
	} else {
		// SLO mode includes empty-tag samples (shared tag-agnostic PSL has no tag label).
		assert.Equal(t, map[string][]string{"v1": {"test"}, "": {"test"}}, r, "Should get alerts for v1 and empty-tag SLO")
	}
}
