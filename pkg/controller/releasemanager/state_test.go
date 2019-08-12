package releasemanager

import (
	"context"
	tt "testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestTestingState(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := getMockDeployment(ctrl, false, false, false, false, false)
	state, err := handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, false, false, false, false, true)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, false, true, false, false, false)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, false, true, false, false, true)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, true, false, false, false, false)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, testing)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, true, false, false, false, true)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, tested)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, true, true, false, false, false)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, true, true, false, false, true)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["testing"].reached(m))
}

func TestTestedStates(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := getMockDeployment(ctrl, false, false, false, false, false)
	state, err := handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, false, false, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, false, true, false, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, false, true, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, true, false, false, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, true, false, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, true, true, false, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, true, true, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, false, false, false, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, tested)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, false, false, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, tested)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, false, true, false, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, tested)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, false, true, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, released)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, true, false, false, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, true, false, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, true, true, false, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, true, true, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["tested"].reached(m))
}

func getMockDeployment(ctrl *gomock.Controller, hasRevision, isAlarmTriggered, isReleaseEligible, schedulePermitsRelease, isTestingComplete bool) *MockDeployment {
	m := NewMockDeployment(ctrl)

	m.
		EXPECT().
		hasRevision().
		Return(hasRevision).
		AnyTimes()
	m.
		EXPECT().
		isReleaseEligible().
		Return(isReleaseEligible).
		AnyTimes()
	m.
		EXPECT().
		schedulePermitsRelease().
		Return(schedulePermitsRelease).
		AnyTimes()
	m.
		EXPECT().
		isAlarmTriggered().
		Return(isAlarmTriggered).
		AnyTimes()
	m.
		EXPECT().
		isTestingComplete().
		Return(isTestingComplete).
		AnyTimes()

	return m
}
