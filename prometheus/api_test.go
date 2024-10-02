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
	assert.Equal(t, map[string][]string{"v1": {"test"}}, r, "Should get 1 firing alerts (v1)")
}
