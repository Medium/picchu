package releasemanager

import (
	"context"
	tt "testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPreTestingState(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := getMockDeployment(ctrl, false, false, false, false, false, true)
	state, err := handlers["pendingtest"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["pendingtest"].reached(m))

	m = getMockDeployment(ctrl, false, false, false, false, true, true)
	state, err = handlers["pendingtest"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["pendingtest"].reached(m))

	m = getMockDeployment(ctrl, false, true, false, false, false, true)
	state, err = handlers["pendingtest"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["pendingtest"].reached(m))

	m = getMockDeployment(ctrl, false, true, false, false, true, true)
	state, err = handlers["pendingtest"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["pendingtest"].reached(m))

	m = getMockDeployment(ctrl, true, false, false, false, false, true)
	state, err = handlers["pendingtest"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, pendingtest)
	assert.True(t, handlers["pendingtest"].reached(m))

	m = getMockDeployment(ctrl, true, false, false, false, true, true)
	state, err = handlers["pendingtest"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, testing)
	assert.True(t, handlers["pendingtest"].reached(m))

	m = getMockDeployment(ctrl, true, true, false, false, false, true)
	state, err = handlers["pendingtest"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["pendingtest"].reached(m))

	m = getMockDeployment(ctrl, true, true, false, false, true, true)
	state, err = handlers["pendingtest"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["pendingtest"].reached(m))
}

func TestTestingState(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := getMockDeployment(ctrl, false, false, false, false, true, true)
	state, err := handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, false, false, false, false, true, false)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, false, true, false, false, true, true)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, false, true, false, false, true, false)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, true, false, false, false, true, true)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, testing)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, true, false, false, false, true, false)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, tested)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, true, true, false, false, true, true)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["testing"].reached(m))

	m = getMockDeployment(ctrl, true, true, false, false, true, false)
	state, err = handlers["testing"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["testing"].reached(m))
}

func TestTestedStates(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := getMockDeployment(ctrl, false, false, false, false, true, false)
	state, err := handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, false, false, true, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, false, true, false, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, false, true, true, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, true, false, false, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, true, false, true, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, true, true, false, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, false, true, true, true, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, deleted)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, false, false, false, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, tested)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, false, false, true, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, tested)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, false, true, false, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, tested)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, false, true, true, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, released)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, true, false, false, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, true, false, true, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, true, true, false, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["tested"].reached(m))

	m = getMockDeployment(ctrl, true, true, true, true, true, false)
	state, err = handlers["tested"].tick(ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, state, failed)
	assert.True(t, handlers["tested"].reached(m))
}

func getMockDeployment(ctrl *gomock.Controller, hasRevision, isAlarmTriggered, isReleaseEligible, schedulePermitsRelease, isTestStarted, isTestPending bool) *MockDeployment {
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
		isTestPending().
		Return(isTestPending).
		AnyTimes()
	m.
		EXPECT().
		isTestStarted().
		Return(isTestStarted).
		AnyTimes()

	return m
}
