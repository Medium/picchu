package scaling

import (
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"k8s.io/utils/pointer"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/scaling/mocks"
)

func prepareLinearMock(ctrl *gomock.Controller, isReconciled bool, currentPercent, peakPercent, increment, max, delay int, lastUpdated time.Time) ScalableTarget {
	m := mocks.NewMockScalableTarget(ctrl)
	releaseInfo := picchu.ReleaseInfo{
		Eligible:         false,
		Max:              uint32(max),
		ScalingStrategy:  picchu.ScalingStrategyLinear,
		GeometricScaling: picchu.GeometricScaling{},
		LinearScaling:    picchu.LinearScaling{},
		Rate: picchu.RateInfo{
			Increment:    uint32(increment),
			DelaySeconds: pointer.Int64Ptr(int64(delay)),
		},
		Schedule: "",
		TTL:      0,
	}

	m.
		EXPECT().
		CurrentPercent().
		Return(uint32(currentPercent)).
		AnyTimes()
	m.
		EXPECT().
		IsReconciled(gomock.Any()).
		Return(isReconciled).
		AnyTimes()
	m.
		EXPECT().
		PeakPercent().
		Return(uint32(peakPercent)).
		AnyTimes()
	m.
		EXPECT().
		ReleaseInfo().
		Return(releaseInfo).
		AnyTimes()
	m.
		EXPECT().
		LastUpdated().
		Return(lastUpdated).
		AnyTimes()

	return m
}

func TestLinearScaling(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	m := prepareLinearMock(ctrl, true, 0, 0, 5, 100, 5, time.Time{})

	assert.Equal(t, 5, int(LinearScale(m, 100, time.Now())), "Scale should increment by 5")

	m = prepareLinearMock(ctrl, true, 100, 100, 5, 100, 5, time.Time{})

	assert.Equal(t, 100, int(LinearScale(m, 100, time.Now())), "Scale shouldn't increment by 5 when at max")

	m = prepareLinearMock(ctrl, true, 50, 100, 5, 100, 5, time.Time{})

	assert.Equal(t, 50, int(LinearScale(m, 50, time.Now())), "Scale shouldn't increment by 5 when at max remaining")

	m = prepareLinearMock(ctrl, true, 50, 100, 5, 50, 5, time.Time{})

	assert.Equal(t, 50, int(LinearScale(m, 100, time.Now())), "Scale shouldn't increment by 5 when at max")

	m = prepareLinearMock(ctrl, true, 50, 50, 5, 100, 5, time.Time{})

	assert.Equal(t, 55, int(LinearScale(m, 100, time.Now())), "Scale should increment by 5")

	m = prepareLinearMock(ctrl, false, 50, 50, 5, 100, 5, time.Time{})

	assert.Equal(t, 50, int(LinearScale(m, 100, time.Now())), "Scale shouldn't be incremented")

	m = prepareLinearMock(ctrl, true, 0, 99, 5, 100, 5, time.Time{})

	assert.Equal(t, 5, int(LinearScale(m, 100, time.Now())), "Scale should not skip to peak")

	m = prepareLinearMock(ctrl, true, 0, 99, 5, 100, 5, time.Time{})

	assert.Equal(t, 5, int(LinearScale(m, 80, time.Now())), "Scale should not skip to max remaining")
}
