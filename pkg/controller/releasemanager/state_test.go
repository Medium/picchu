package releasemanager

// TODO(bob): errors on the deployment interface aren't tested here, and should be

import (
	"context"
	tt "testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCreated(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision: hasRevision,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "created", expected, mock)
	}

	testcase(deleting, m(false))
	testcase(deploying, m(true))
}

func TestDeploying(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered, isDeployed bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:      hasRevision,
			isAlarmTriggered: isAlarmTriggered,
			isDeployed:       isDeployed,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "deploying", expected, mock)
	}

	testcase(deleting, m(false, false, false))
	testcase(deleting, m(false, false, true))
	testcase(deleting, m(false, true, false))
	testcase(deleting, m(false, true, true))
	testcase(failing, m(true, true, false))
	testcase(failing, m(true, true, true))

	testcase(deploying, expectSync(m(true, false, false)))

	testcase(deployed, expectSync(m(true, false, true)))
}

func TestDeployed(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered, isDeployed, isTestPending, isReleaseEligible, isCanaryPending bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:       hasRevision,
			isAlarmTriggered:  isAlarmTriggered,
			isDeployed:        isDeployed,
			isTestPending:     isTestPending,
			isReleaseEligible: isReleaseEligible,
			isCanaryPending:   isCanaryPending,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "deployed", expected, mock)
	}

	testcase(deleting, m(false, false, false, false, false, false))
	testcase(deleting, m(false, false, false, false, true, false))
	testcase(deleting, m(false, false, false, true, false, false))
	testcase(deleting, m(false, false, false, true, true, false))
	testcase(deleting, m(false, false, true, false, false, false))
	testcase(deleting, m(false, false, true, false, true, false))
	testcase(deleting, m(false, false, true, true, false, false))
	testcase(deleting, m(false, false, true, true, true, false))
	testcase(deleting, m(false, false, true, false, false, false))
	testcase(deleting, m(false, false, true, false, true, false))
	testcase(deleting, m(false, false, true, true, false, false))
	testcase(deleting, m(false, false, true, true, true, false))
	testcase(deleting, m(false, true, false, false, false, false))
	testcase(deleting, m(false, true, false, false, true, false))
	testcase(deleting, m(false, true, false, true, false, false))
	testcase(deleting, m(false, true, false, true, true, false))
	testcase(deleting, m(false, true, true, false, false, false))
	testcase(deleting, m(false, true, true, false, true, false))
	testcase(deleting, m(false, true, true, true, false, false))
	testcase(deleting, m(false, true, true, true, true, false))

	testcase(failing, m(true, true, false, false, false, false))
	testcase(failing, m(true, true, false, false, true, false))
	testcase(failing, m(true, true, false, true, false, false))
	testcase(failing, m(true, true, false, true, true, false))
	testcase(failing, m(true, true, true, false, false, false))
	testcase(failing, m(true, true, true, false, true, false))
	testcase(failing, m(true, true, true, true, false, false))
	testcase(failing, m(true, true, true, true, true, false))

	testcase(deploying, expectSync(m(true, false, false, false, false, false)))
	testcase(deploying, expectSync(m(true, false, false, false, true, false)))
	testcase(deploying, expectSync(m(true, false, false, true, false, false)))
	testcase(deploying, expectSync(m(true, false, false, true, true, false)))

	testcase(deployed, expectSync(m(true, false, true, false, false, false)))

	testcase(pendingtest, expectSync(m(true, false, true, true, true, false)))
	testcase(pendingtest, expectSync(m(true, false, true, true, false, false)))

	testcase(pendingrelease, expectSync(m(true, false, true, false, true, false)))

	// now with canaries
	testcase(deleting, m(false, false, false, false, false, true))
	testcase(deleting, m(false, false, false, false, true, true))
	testcase(deleting, m(false, false, false, true, false, true))
	testcase(deleting, m(false, false, false, true, true, true))
	testcase(deleting, m(false, false, true, false, false, true))
	testcase(deleting, m(false, false, true, false, true, true))
	testcase(deleting, m(false, false, true, true, false, true))
	testcase(deleting, m(false, false, true, true, true, true))
	testcase(deleting, m(false, false, true, false, false, true))
	testcase(deleting, m(false, false, true, false, true, true))
	testcase(deleting, m(false, false, true, true, false, true))
	testcase(deleting, m(false, false, true, true, true, true))
	testcase(deleting, m(false, true, false, false, false, true))
	testcase(deleting, m(false, true, false, false, true, true))
	testcase(deleting, m(false, true, false, true, false, true))
	testcase(deleting, m(false, true, false, true, true, true))
	testcase(deleting, m(false, true, true, false, false, true))
	testcase(deleting, m(false, true, true, false, true, true))
	testcase(deleting, m(false, true, true, true, false, true))
	testcase(deleting, m(false, true, true, true, true, true))

	testcase(failing, m(true, true, false, false, false, true))
	testcase(failing, m(true, true, false, false, true, true))
	testcase(failing, m(true, true, false, true, false, true))
	testcase(failing, m(true, true, false, true, true, true))
	testcase(failing, m(true, true, true, false, false, true))
	testcase(failing, m(true, true, true, false, true, true))
	testcase(failing, m(true, true, true, true, false, true))
	testcase(failing, m(true, true, true, true, true, true))

	testcase(deploying, expectSync(m(true, false, false, false, false, true)))
	testcase(deploying, expectSync(m(true, false, false, false, true, true)))
	testcase(deploying, expectSync(m(true, false, false, true, false, true)))
	testcase(deploying, expectSync(m(true, false, false, true, true, true)))

	testcase(canarying, expectSync(m(true, false, true, false, false, true)))

	testcase(pendingtest, expectSync(m(true, false, true, true, true, true)))
	testcase(pendingtest, expectSync(m(true, false, true, true, false, true)))

	testcase(canarying, expectSync(m(true, false, true, false, true, true)))
}

func TestPendingTest(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered, isTestStarted bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:      hasRevision,
			isAlarmTriggered: isAlarmTriggered,
			isTestStarted:    isTestStarted,
			isTestPending:    true,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "pendingtest", expected, mock)
	}

	testcase(deleting, m(false, false, false))
	testcase(deleting, m(false, false, true))
	testcase(deleting, m(false, true, false))
	testcase(deleting, m(false, true, true))
	testcase(pendingtest, m(true, false, false))
	testcase(testing, m(true, false, true))
	testcase(failing, m(true, true, false))
	testcase(failing, m(true, true, true))
}

func TestTesting(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered, isTestPending bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:      hasRevision,
			isAlarmTriggered: isAlarmTriggered,
			isTestStarted:    true,
			isTestPending:    isTestPending,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "testing", expected, mock)
	}

	testcase(deleting, m(false, false, false))
	testcase(deleting, m(false, false, true))
	testcase(deleting, m(false, true, false))
	testcase(deleting, m(false, true, true))
	testcase(tested, m(true, false, false))
	testcase(testing, m(true, false, true))
	testcase(failing, m(true, true, false))
	testcase(failing, m(true, true, true))
}

func TestTested(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered, isReleaseEligible, isCanaryPending bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:       hasRevision,
			isAlarmTriggered:  isAlarmTriggered,
			isReleaseEligible: isReleaseEligible,
			isTestStarted:     true,
			isCanaryPending:   isCanaryPending,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "tested", expected, mock)
	}

	testcase(deleting, m(false, false, false, false))
	testcase(deleting, m(false, false, true, false))
	testcase(deleting, m(false, true, false, false))
	testcase(deleting, m(false, true, true, false))
	testcase(tested, m(true, false, false, false))
	testcase(pendingrelease, m(true, false, true, false))
	testcase(failing, m(true, true, false, false))
	testcase(failing, m(true, true, true, false))

	// now with canary
	testcase(deleting, m(false, false, false, true))
	testcase(deleting, m(false, false, true, true))
	testcase(deleting, m(false, true, false, true))
	testcase(deleting, m(false, true, true, true))
	testcase(canarying, m(true, false, false, true))
	testcase(canarying, m(true, false, true, true))
	testcase(failing, m(true, true, false, true))
	testcase(failing, m(true, true, true, true))
}

func TestCanarying(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered, isCanaryPending bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:      hasRevision,
			isAlarmTriggered: isAlarmTriggered,
			isCanaryPending:  isCanaryPending,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "canarying", expected, mock)
	}

	testcase(deleting, m(false, false, false))
	testcase(deleting, m(false, false, true))
	testcase(deleting, m(false, true, false))
	testcase(deleting, m(false, true, true))
	testcase(canaried, expectSyncCanaryRules(m(true, false, false)))
	testcase(canarying, expectSyncCanaryRules(m(true, false, true)))
	testcase(failing, m(true, true, false))
	testcase(failing, m(true, true, true))
}

func TestCanaried(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered, isReleaseEligible bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:       hasRevision,
			isAlarmTriggered:  isAlarmTriggered,
			isReleaseEligible: isReleaseEligible,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "canaried", expected, mock)
	}

	testcase(deleting, m(false, false, false))
	testcase(deleting, m(false, false, true))
	testcase(deleting, m(false, true, false))
	testcase(deleting, m(false, true, true))
	testcase(canaried, expectDeleteCanaryRules(m(true, false, false)))
	testcase(pendingrelease, expectDeleteCanaryRules(m(true, false, true)))
	testcase(failing, m(true, true, false))
	testcase(failing, m(true, true, true))
}

func TestPendingRelease(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered, isReleaseEligible, schedulePermitsRelease bool, peakPercent uint32) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:            hasRevision,
			isAlarmTriggered:       isAlarmTriggered,
			isReleaseEligible:      isReleaseEligible,
			schedulePermitsRelease: schedulePermitsRelease,
			peakPercent:            peakPercent,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "pendingrelease", expected, mock)
	}

	testcase(deleting, m(false, false, false, false, 0))
	testcase(deleting, m(false, false, false, true, 0))
	testcase(deleting, m(false, false, true, false, 0))
	testcase(deleting, m(false, false, true, true, 0))
	testcase(deleting, m(false, true, false, false, 0))
	testcase(deleting, m(false, true, false, true, 0))
	testcase(deleting, m(false, true, true, false, 0))
	testcase(deleting, m(false, true, true, true, 0))

	testcase(failing, m(true, true, false, false, 0))
	testcase(failing, m(true, true, false, true, 0))
	testcase(failing, m(true, true, true, false, 0))
	testcase(failing, m(true, true, true, true, 0))

	testcase(retiring, m(true, false, false, true, 0))
	testcase(retiring, m(true, false, false, false, 0))

	testcase(releasing, m(true, false, true, true, 0))

	testcase(pendingrelease, m(true, false, true, false, 0))

	testcase(deleting, m(false, false, false, false, 100))
	testcase(deleting, m(false, false, false, true, 100))
	testcase(deleting, m(false, false, true, false, 100))
	testcase(deleting, m(false, false, true, true, 100))
	testcase(deleting, m(false, true, false, false, 100))
	testcase(deleting, m(false, true, false, true, 100))
	testcase(deleting, m(false, true, true, false, 100))
	testcase(deleting, m(false, true, true, true, 100))

	testcase(failing, m(true, true, false, false, 100))
	testcase(failing, m(true, true, false, true, 100))
	testcase(failing, m(true, true, true, false, 100))
	testcase(failing, m(true, true, true, true, 100))

	testcase(retiring, m(true, false, false, true, 100))
	testcase(retiring, m(true, false, false, false, 100))

	testcase(releasing, m(true, false, true, true, 100))

	testcase(releasing, m(true, false, true, false, 100))
}

func TestReleasing(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered, isReleaseEligible bool, peakPercent uint32) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:       hasRevision,
			isAlarmTriggered:  isAlarmTriggered,
			isReleaseEligible: isReleaseEligible,
			peakPercent:       peakPercent,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "releasing", expected, mock)
	}

	testcase(deleting, m(false, false, false, 0))
	testcase(deleting, m(false, false, true, 0))
	testcase(deleting, m(false, true, false, 0))
	testcase(deleting, m(false, true, true, 0))
	testcase(deleting, m(false, false, false, 100))
	testcase(deleting, m(false, false, true, 100))
	testcase(deleting, m(false, true, false, 100))
	testcase(deleting, m(false, true, true, 100))

	testcase(failing, m(true, true, false, 0))
	testcase(failing, m(true, true, true, 0))
	testcase(failing, m(true, true, false, 100))
	testcase(failing, m(true, true, true, 100))

	testcase(retiring, m(true, false, false, 0))
	testcase(retiring, m(true, false, false, 100))

	testcase(releasing, expectSyncSLIRules(expectSync(m(true, false, true, 0))))
	testcase(releasing, expectSyncSLIRules(expectSync(m(true, false, true, 99))))
	testcase(released, expectSyncSLIRules(expectSync(m(true, false, true, 100))))
}

func TestReleased(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered, isReleaseEligible bool, peakPercent uint32) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:       hasRevision,
			isAlarmTriggered:  isAlarmTriggered,
			isReleaseEligible: isReleaseEligible,
			peakPercent:       peakPercent,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "released", expected, mock)
	}

	testcase(deleting, m(false, false, false, 0))
	testcase(deleting, m(false, false, true, 0))
	testcase(deleting, m(false, true, false, 0))
	testcase(deleting, m(false, true, true, 0))
	testcase(deleting, m(false, false, false, 100))
	testcase(deleting, m(false, false, true, 100))
	testcase(deleting, m(false, true, false, 100))
	testcase(deleting, m(false, true, true, 100))

	testcase(failing, m(true, true, false, 0))
	testcase(failing, m(true, true, true, 0))
	testcase(failing, m(true, true, false, 100))
	testcase(failing, m(true, true, true, 100))

	testcase(retiring, m(true, false, false, 0))
	testcase(retiring, m(true, false, false, 100))

	testcase(releasing, expectSyncSLIRules(expectSync(m(true, false, true, 0))))
	testcase(releasing, expectSyncSLIRules(expectSync(m(true, false, true, 99))))
	testcase(released, expectSyncSLIRules(expectSync(m(true, false, true, 100))))
}

func TestRetiring(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered, isReleaseEligible bool, currentPercent uint32) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:       hasRevision,
			isAlarmTriggered:  isAlarmTriggered,
			isReleaseEligible: isReleaseEligible,
			currentPercent:    currentPercent,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "retiring", expected, mock)
	}

	testcase(deleting, m(false, false, false, 0))
	testcase(deleting, m(false, false, true, 0))
	testcase(deleting, m(false, true, false, 0))
	testcase(deleting, m(false, true, true, 0))
	testcase(deleting, m(false, false, false, 100))
	testcase(deleting, m(false, false, true, 100))
	testcase(deleting, m(false, true, false, 100))
	testcase(deleting, m(false, true, true, 100))

	testcase(failing, m(true, true, false, 0))
	testcase(failing, m(true, true, true, 0))
	testcase(failing, m(true, true, false, 100))
	testcase(failing, m(true, true, true, 100))

	testcase(retired, expectDeleteSLIRules(expectRetire(m(true, false, false, 0))))
	testcase(retiring, expectDeleteSLIRules(m(true, false, false, 1)))
	testcase(retiring, expectDeleteSLIRules(m(true, false, false, 100)))

	testcase(deploying, m(true, false, true, 0))
	testcase(deploying, m(true, false, true, 100))
}

func TestRetired(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered, isReleaseEligible bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:       hasRevision,
			isAlarmTriggered:  isAlarmTriggered,
			isReleaseEligible: isReleaseEligible,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "retired", expected, mock)
	}

	testcase(deleting, m(false, false, false))
	testcase(deleting, m(false, false, true))
	testcase(deleting, m(false, true, false))
	testcase(deleting, m(false, true, true))

	testcase(failing, m(true, true, false))
	testcase(failing, m(true, true, true))

	testcase(retired, expectRetire(m(true, false, false)))
	testcase(deploying, m(true, false, true))
}

func TestDeleting(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision bool, currentPercent uint32) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:    hasRevision,
			currentPercent: currentPercent,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "deleting", expected, mock)
	}

	testcase(deleting, expectDeleteCanaryRules(expectDeleteSLIRules(m(false, 100))))
	testcase(deleting, expectDeleteCanaryRules(expectDeleteSLIRules(m(false, 1))))
	testcase(deleted, expectDeleteCanaryRules(expectDeleteSLIRules(expectDelete(m(false, 0)))))
	testcase(deploying, m(true, 0))
	testcase(deploying, m(true, 100))
}

func TestDeleted(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision: hasRevision,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "deleted", expected, mock)
	}

	testcase(deleted, m(false))
	testcase(deploying, m(true))
}

func TestFailing(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered bool, currentPercent uint32) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:      hasRevision,
			isAlarmTriggered: isAlarmTriggered,
			currentPercent:   currentPercent,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "failing", expected, mock)
	}

	testcase(deleting, m(false, true, 0))
	testcase(deleting, m(false, true, 100))
	testcase(deleting, m(false, false, 0))
	testcase(deleting, m(false, false, 100))

	testcase(deploying, m(true, false, 0))
	testcase(deploying, m(true, false, 100))

	testcase(failed, expectDeleteCanaryRules(expectDeleteSLIRules(expectRetire(m(true, true, 0)))))
	testcase(failing, expectDeleteCanaryRules(expectDeleteSLIRules(m(true, true, 1))))
	testcase(failing, expectDeleteCanaryRules(expectDeleteSLIRules(m(true, true, 100))))
}

func TestFailed(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, isAlarmTriggered bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:      hasRevision,
			isAlarmTriggered: isAlarmTriggered,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "failed", expected, mock)
	}

	testcase(deleting, m(false, true))
	testcase(deleting, m(false, false))

	testcase(deploying, m(true, false))

	testcase(failed, expectRetire(m(true, true)))
}

func testHandler(ctx context.Context, t *tt.T, handler string, expected State, m *MockDeployment) {
	state, err := handlers[handler](ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, expected, state)
}

type responses struct {
	hasRevision            bool
	isAlarmTriggered       bool
	isReleaseEligible      bool
	isTestStarted          bool
	isTestPending          bool
	isCanaryPending        bool
	isDeployed             bool
	schedulePermitsRelease bool
	currentPercent         uint32
	peakPercent            uint32
	syncCanaryRules        error
	deleteCanaryRules      error
	syncSLIRules           error
	deleteSLIRules         error
}

func createMockDeployment(ctrl *gomock.Controller, r responses) *MockDeployment {
	m := NewMockDeployment(ctrl)

	m.
		EXPECT().
		hasRevision().
		Return(r.hasRevision).
		AnyTimes()
	m.
		EXPECT().
		isReleaseEligible().
		Return(r.isReleaseEligible).
		AnyTimes()
	m.
		EXPECT().
		isAlarmTriggered().
		Return(r.isAlarmTriggered).
		AnyTimes()
	m.
		EXPECT().
		isTestPending().
		Return(r.isTestPending).
		AnyTimes()
	m.
		EXPECT().
		isCanaryPending().
		Return(r.isCanaryPending).
		AnyTimes()
	m.
		EXPECT().
		isTestStarted().
		Return(r.isTestStarted).
		AnyTimes()
	m.
		EXPECT().
		isDeployed().
		Return(r.isDeployed).
		AnyTimes()
	m.
		EXPECT().
		schedulePermitsRelease().
		Return(r.schedulePermitsRelease).
		AnyTimes()
	m.
		EXPECT().
		currentPercent().
		Return(r.currentPercent).
		AnyTimes()
	m.
		EXPECT().
		peakPercent().
		Return(r.peakPercent).
		AnyTimes()
	return m
}

func expectSync(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		sync(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}

func expectRetire(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		retire(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}

func expectDelete(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		del(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}

func expectSyncCanaryRules(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		syncCanaryRules(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}

func expectDeleteCanaryRules(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		deleteCanaryRules(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}

func expectSyncSLIRules(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		syncSLIRules(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}

func expectDeleteSLIRules(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		deleteSLIRules(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}
