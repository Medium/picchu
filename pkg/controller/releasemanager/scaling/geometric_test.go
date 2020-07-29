package scaling

import (
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	testify "github.com/stretchr/testify/assert"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/scaling/mocks"
)

func prepareGeometricMock(ctrl *gomock.Controller, isReconciled bool, currentPercent, peakPercent, factor, max, start int, delay time.Duration, lastUpdated time.Time) ScalableTarget {
	m := mocks.NewMockScalableTarget(ctrl)
	releaseInfo := picchu.ReleaseInfo{
		Eligible:        false,
		Max:             uint32(max),
		ScalingStrategy: picchu.ScalingStrategyGeometric,
		GeometricScaling: picchu.GeometricScaling{
			Start:  uint32(start),
			Factor: uint32(factor),
			Delay:  &metav1.Duration{Duration: delay},
		},
		LinearScaling: picchu.LinearScaling{},
		Rate:          picchu.RateInfo{},
		Schedule:      "",
		TTL:           0,
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

func TestGeometricScaling(t *testing.T) {
	assert := testify.New(t)
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	m := prepareGeometricMock(ctrl, true, 0, 0, 2, 100, 5, time.Duration(5)*time.Second, time.Time{})

	assert.Equal(5, int(GeometricScale(m, 100, time.Now())), "Scale start at 5")

	m = prepareGeometricMock(ctrl, true, 100, 100, 2, 100, 5, time.Duration(5)*time.Second, time.Time{})

	assert.Equal(100, int(GeometricScale(m, 100, time.Now())), "Scale shouldn't increment when at max")

	m = prepareGeometricMock(ctrl, true, 50, 100, 2, 100, 5, time.Duration(5)*time.Second, time.Time{})

	assert.Equal(50, int(GeometricScale(m, 50, time.Now())), "Scale shouldn't increment when at max remaining")

	m = prepareGeometricMock(ctrl, true, 50, 100, 2, 50, 5, time.Duration(5)*time.Second, time.Time{})

	assert.Equal(50, int(GeometricScale(m, 100, time.Now())), "Scale shouldn't increment when at max")

	m = prepareGeometricMock(ctrl, true, 25, 50, 2, 100, 5, time.Duration(5)*time.Second, time.Time{})

	assert.Equal(50, int(GeometricScale(m, 100, time.Now())), "Scale should double")

	m = prepareGeometricMock(ctrl, true, 50, 50, 2, 100, 5, time.Duration(5)*time.Second, time.Time{})

	assert.Equal(100, int(GeometricScale(m, 100, time.Now())), "Scale should double")

	m = prepareGeometricMock(ctrl, false, 50, 50, 2, 100, 5, time.Duration(5)*time.Second, time.Time{})

	assert.Equal(50, int(GeometricScale(m, 100, time.Now())), "Scale shouldn't be incremented when unreconciled")

	m = prepareGeometricMock(ctrl, true, 0, 99, 2, 100, 5, time.Duration(5)*time.Second, time.Time{})

	assert.Equal(5, int(GeometricScale(m, 100, time.Now())), "Scale should not skip to peak")

	m = prepareGeometricMock(ctrl, true, 0, 99, 2, 100, 5, time.Duration(5)*time.Second, time.Time{})

	assert.Equal(5, int(GeometricScale(m, 80, time.Now())), "Scale should not skip to max remaining")
}
