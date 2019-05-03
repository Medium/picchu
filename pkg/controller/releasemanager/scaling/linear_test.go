package scaling

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/scaling/mocks"
)

func prepareMock(ctrl *gomock.Controller, currentPercent, peakPercent, increment, max, delay int, lastUpdated time.Time) ScalableTarget {
	m := mocks.NewMockScalableTarget(ctrl)

	m.
		EXPECT().
		CurrentPercent().
		Return(uint32(currentPercent)).
		AnyTimes()
	m.
		EXPECT().
		PeakPercent().
		Return(uint32(peakPercent)).
		AnyTimes()
	m.
		EXPECT().
		Increment().
		Return(uint32(increment)).
		AnyTimes()
	m.
		EXPECT().
		Max().
		Return(uint32(max)).
		AnyTimes()
	m.
		EXPECT().
		Delay().
		Return(time.Duration(delay) * time.Second).
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

	m := prepareMock(ctrl, 0, 0, 5, 100, 5, time.Time{})

	assert.Equal(t, 5, int(LinearScale(m, 100, time.Now())), "Scale should increment by 5")

	m = prepareMock(ctrl, 100, 100, 5, 100, 5, time.Time{})

	assert.Equal(t, 100, int(LinearScale(m, 100, time.Now())), "Scale shouldn't increment by 5 when at max")

	m = prepareMock(ctrl, 50, 100, 5, 100, 5, time.Time{})

	assert.Equal(t, 50, int(LinearScale(m, 50, time.Now())), "Scale shouldn't increment by 5 when at max remaining")

	m = prepareMock(ctrl, 50, 100, 5, 50, 5, time.Time{})

	assert.Equal(t, 50, int(LinearScale(m, 100, time.Now())), "Scale shouldn't increment by 5 when at max")

	m = prepareMock(ctrl, 50, 50, 5, 100, 5, time.Time{})

	assert.Equal(t, 55, int(LinearScale(m, 100, time.Now())), "Scale should increment by 5")

	m = prepareMock(ctrl, 0, 100, 5, 100, 5, time.Time{})

	assert.Equal(t, 100, int(LinearScale(m, 100, time.Now())), "Scale should skip to peak")

	m = prepareMock(ctrl, 0, 100, 5, 100, 5, time.Time{})

	assert.Equal(t, 80, int(LinearScale(m, 80, time.Now())), "Scale should skip to max remaining")
}
