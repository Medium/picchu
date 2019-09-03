package prometheus

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/prometheus/mocks"
)

func TestPrometheusCache(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	m := mocks.NewMockPromAPI(ctrl)
	api := InjectAPI(m, time.Duration(25)*time.Millisecond)

	aq := NewAlertQuery("tutu", string(picchuv1alpha1.SLO))

	q := bytes.NewBufferString("")
	assert.Nil(t, alertTemplate.Execute(q, aq), "Template execute shouldn't fail")

	m.
		EXPECT().
		Query(gomock.Any(), gomock.Eq(q.String()), gomock.Any()).
		Return(model.Vector{}, nil).
		Times(2)

	for i := 0; i < 5; i++ {
		r, err := api.TaggedAlerts(context.TODO(), aq, time.Now())
		assert.Nil(t, err, "Should succeed in querying alerts")
		assert.Equal(t, []string{}, r, "Should get no firing alerts")
	}
	time.Sleep(time.Duration(25) * time.Millisecond)
	r, err := api.TaggedAlerts(context.TODO(), aq, time.Now())
	assert.Nil(t, err, "Should succeed in querying alerts")
	assert.Equal(t, []string{}, r, "Should get no firing alerts")
}

func TestPrometheusAlerts(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	m := mocks.NewMockPromAPI(ctrl)
	api := InjectAPI(m, time.Duration(25)*time.Millisecond)

	aq := NewAlertQuery("tutu", string(picchuv1alpha1.SLO))

	q := bytes.NewBufferString("")
	assert.Nil(t, alertTemplate.Execute(q, aq), "Template execute shouldn't fail")

	response := model.Vector{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"app": "tutu",
				"tag": "v1",
			},
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"app":  "tutu",
				"xtag": "v2",
			},
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"xapp": "tutu",
				"tag":  "v3",
			},
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"app": "tutux",
				"tag": "v4",
			},
		},
	}

	m.
		EXPECT().
		Query(gomock.Any(), gomock.Eq(q.String()), gomock.Any()).
		Return(response, nil).
		Times(1)

	r, err := api.TaggedAlerts(context.TODO(), aq, time.Now())
	assert.Nil(t, err, "Should succeed in querying alerts")
	assert.Equal(t, []string{"v1"}, r, "Should get 1 firing alerts (v1)")
}
