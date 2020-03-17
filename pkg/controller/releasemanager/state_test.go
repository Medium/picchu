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

	m := func(hasRevision, markedAsFailed, isDeployed bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:    hasRevision,
			markedAsFailed: markedAsFailed,
			isDeployed:     isDeployed,
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

	m := func(hasRevision, markedAsFailed, isDeployed, isReleaseEligible, isCanaryPending bool, externalTestStatus ExternalTestStatus) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:        hasRevision,
			markedAsFailed:     markedAsFailed,
			isDeployed:         isDeployed,
			isReleaseEligible:  isReleaseEligible,
			isCanaryPending:    isCanaryPending,
			externalTestStatus: externalTestStatus,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "deployed", expected, mock)
	}

	// TODO(lyra): clean up; when !hasRevision, only ExternalTestUnknown is possible
	testcase(deleting, m(false, false, false, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, false, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, false, false, false, ExternalTestPending))
	testcase(deleting, m(false, false, false, true, false, ExternalTestPending))
	testcase(deleting, m(false, false, true, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, false, false, ExternalTestPending))
	testcase(deleting, m(false, false, true, true, false, ExternalTestPending))
	testcase(deleting, m(false, false, true, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, false, false, ExternalTestPending))
	testcase(deleting, m(false, false, true, true, false, ExternalTestPending))
	testcase(deleting, m(false, true, false, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, true, false, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, true, false, false, false, ExternalTestPending))
	testcase(deleting, m(false, true, false, true, false, ExternalTestPending))
	testcase(deleting, m(false, true, true, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, true, true, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, true, true, false, false, ExternalTestPending))
	testcase(deleting, m(false, true, true, true, false, ExternalTestPending))

	testcase(failing, m(true, true, false, false, false, ExternalTestDisabled))
	testcase(failing, m(true, true, false, true, false, ExternalTestDisabled))
	testcase(failing, m(true, true, false, false, false, ExternalTestPending))
	testcase(failing, m(true, true, false, true, false, ExternalTestPending))
	testcase(failing, m(true, true, true, false, false, ExternalTestDisabled))
	testcase(failing, m(true, true, true, true, false, ExternalTestDisabled))
	testcase(failing, m(true, true, true, false, false, ExternalTestPending))
	testcase(failing, m(true, true, true, true, false, ExternalTestPending))

	testcase(deploying, expectSync(m(true, false, false, false, false, ExternalTestDisabled)))
	testcase(deploying, expectSync(m(true, false, false, true, false, ExternalTestDisabled)))
	testcase(deploying, expectSync(m(true, false, false, false, false, ExternalTestPending)))
	testcase(deploying, expectSync(m(true, false, false, true, false, ExternalTestPending)))

	testcase(deployed, expectSync(m(true, false, true, false, false, ExternalTestDisabled)))

	testcase(pendingtest, expectSync(m(true, false, true, true, false, ExternalTestPending)))
	testcase(pendingtest, expectSync(m(true, false, true, false, false, ExternalTestPending)))
	testcase(testing, expectSync(m(true, false, true, true, false, ExternalTestStarted)))
	testcase(testing, expectSync(m(true, false, true, false, false, ExternalTestStarted)))
	testcase(tested, expectSync(m(true, false, true, true, false, ExternalTestSucceeded)))
	testcase(tested, expectSync(m(true, false, true, false, false, ExternalTestSucceeded)))

	testcase(pendingrelease, expectSync(m(true, false, true, true, false, ExternalTestDisabled)))

	// now with canaries
	testcase(deleting, m(false, false, false, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, false, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, false, false, true, ExternalTestPending))
	testcase(deleting, m(false, false, false, true, true, ExternalTestPending))
	testcase(deleting, m(false, false, true, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, false, true, ExternalTestPending))
	testcase(deleting, m(false, false, true, true, true, ExternalTestPending))
	testcase(deleting, m(false, false, true, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, false, true, ExternalTestPending))
	testcase(deleting, m(false, false, true, true, true, ExternalTestPending))
	testcase(deleting, m(false, true, false, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, true, false, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, true, false, false, true, ExternalTestPending))
	testcase(deleting, m(false, true, false, true, true, ExternalTestPending))
	testcase(deleting, m(false, true, true, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, true, true, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, true, true, false, true, ExternalTestPending))
	testcase(deleting, m(false, true, true, true, true, ExternalTestPending))

	testcase(failing, m(true, true, false, false, true, ExternalTestDisabled))
	testcase(failing, m(true, true, false, true, true, ExternalTestDisabled))
	testcase(failing, m(true, true, false, false, true, ExternalTestPending))
	testcase(failing, m(true, true, false, true, true, ExternalTestPending))
	testcase(failing, m(true, true, true, false, true, ExternalTestDisabled))
	testcase(failing, m(true, true, true, true, true, ExternalTestDisabled))
	testcase(failing, m(true, true, true, false, true, ExternalTestPending))
	testcase(failing, m(true, true, true, true, true, ExternalTestPending))
	testcase(failing, m(true, false, true, true, true, ExternalTestFailed))
	testcase(failing, m(true, false, true, true, false, ExternalTestFailed))

	testcase(deploying, expectSync(m(true, false, false, false, true, ExternalTestDisabled)))
	testcase(deploying, expectSync(m(true, false, false, true, true, ExternalTestDisabled)))
	testcase(deploying, expectSync(m(true, false, false, false, true, ExternalTestPending)))
	testcase(deploying, expectSync(m(true, false, false, true, true, ExternalTestPending)))

	testcase(canarying, expectSync(m(true, false, true, false, true, ExternalTestDisabled)))

	testcase(pendingtest, expectSync(m(true, false, true, true, true, ExternalTestPending)))
	testcase(pendingtest, expectSync(m(true, false, true, false, true, ExternalTestPending)))

	testcase(canarying, expectSync(m(true, false, true, true, true, ExternalTestDisabled)))
}

func TestPendingTest(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, markedAsFailed bool, externalTestStatus ExternalTestStatus) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:        hasRevision,
			markedAsFailed:     markedAsFailed,
			externalTestStatus: externalTestStatus,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "pendingtest", expected, mock)
	}

	testcase(deleting, m(false, false, ExternalTestPending))
	testcase(deleting, m(false, false, ExternalTestStarted))
	testcase(deleting, m(false, true, ExternalTestPending))
	testcase(deleting, m(false, true, ExternalTestStarted))
	testcase(pendingtest, expectSync(m(true, false, ExternalTestPending)))
	testcase(testing, m(true, false, ExternalTestStarted))
	testcase(failing, m(true, false, ExternalTestFailed))
	testcase(failing, m(true, true, ExternalTestPending))
	testcase(failing, m(true, true, ExternalTestStarted))
}

func TestTesting(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, markedAsFailed bool, externalTestStatus ExternalTestStatus) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:        hasRevision,
			markedAsFailed:     markedAsFailed,
			externalTestStatus: externalTestStatus,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "testing", expected, mock)
	}

	testcase(deleting, m(false, false, ExternalTestPending))
	testcase(deleting, m(false, false, ExternalTestStarted))
	testcase(deleting, m(false, false, ExternalTestSucceeded))
	testcase(deleting, m(false, true, ExternalTestPending))
	testcase(deleting, m(false, true, ExternalTestStarted))
	testcase(deleting, m(false, true, ExternalTestSucceeded))

	testcase(pendingtest, m(true, false, ExternalTestPending))
	testcase(testing, expectSync(m(true, false, ExternalTestStarted)))
	testcase(tested, m(true, false, ExternalTestSucceeded))
	testcase(failing, m(true, false, ExternalTestFailed))

	testcase(failing, m(true, true, ExternalTestPending))
	testcase(failing, m(true, true, ExternalTestStarted))
	testcase(failing, m(true, true, ExternalTestSucceeded))
	testcase(failing, m(true, true, ExternalTestFailed))

	m = func(hasRevision, markedAsFailed bool, externalTestStatus ExternalTestStatus) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:        hasRevision,
			markedAsFailed:     markedAsFailed,
			externalTestStatus: externalTestStatus,
			isTimingOut:        true,
		})
	}
	testcase(timingout, m(true, false, ExternalTestPending))
	testcase(timingout, m(true, false, ExternalTestStarted))
	testcase(tested, m(true, false, ExternalTestSucceeded))
	testcase(failing, m(true, false, ExternalTestFailed))
}

func TestTested(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, markedAsFailed, isReleaseEligible, isCanaryPending bool, externalTestStatus ExternalTestStatus) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:        hasRevision,
			markedAsFailed:     markedAsFailed,
			isReleaseEligible:  isReleaseEligible,
			isCanaryPending:    isCanaryPending,
			externalTestStatus: externalTestStatus,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "tested", expected, mock)
	}

	testcase(deleting, m(false, false, false, false, ExternalTestSucceeded))
	testcase(deleting, m(false, false, true, false, ExternalTestSucceeded))
	testcase(deleting, m(false, true, false, false, ExternalTestSucceeded))
	testcase(deleting, m(false, true, true, false, ExternalTestSucceeded))
	testcase(tested, m(true, false, false, false, ExternalTestSucceeded))
	testcase(pendingtest, m(true, false, false, false, ExternalTestPending))
	testcase(testing, m(true, false, false, false, ExternalTestStarted))
	testcase(pendingrelease, m(true, false, true, false, ExternalTestSucceeded))
	testcase(failing, m(true, true, false, false, ExternalTestSucceeded))
	testcase(failing, m(true, true, true, false, ExternalTestSucceeded))
	testcase(failing, m(true, false, false, false, ExternalTestFailed))
	testcase(failing, m(true, false, true, false, ExternalTestFailed))

	// now with canary
	testcase(deleting, m(false, false, false, true, ExternalTestSucceeded))
	testcase(deleting, m(false, false, true, true, ExternalTestSucceeded))
	testcase(deleting, m(false, true, false, true, ExternalTestSucceeded))
	testcase(deleting, m(false, true, true, true, ExternalTestSucceeded))
	testcase(canarying, m(true, false, false, true, ExternalTestSucceeded))
	testcase(canarying, m(true, false, true, true, ExternalTestSucceeded))
	testcase(failing, m(true, true, false, true, ExternalTestSucceeded))
	testcase(failing, m(true, true, true, true, ExternalTestSucceeded))
}

func TestCanarying(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, markedAsFailed, isCanaryPending bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:     hasRevision,
			markedAsFailed:  markedAsFailed,
			isCanaryPending: isCanaryPending,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "canarying", expected, mock)
	}

	testcase(deleting, m(false, false, false))
	testcase(deleting, m(false, false, true))
	testcase(deleting, m(false, true, false))
	testcase(deleting, m(false, true, true))
	testcase(canaried, expectSync(expectSyncTaggedServiceLevels(expectSyncCanaryRules(m(true, false, false)))))
	testcase(canarying, expectSync(expectSyncTaggedServiceLevels(expectSyncCanaryRules(m(true, false, true)))))
	testcase(failing, m(true, true, false))
	testcase(failing, m(true, true, true))
}

func TestCanaried(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, markedAsFailed, isReleaseEligible bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:       hasRevision,
			markedAsFailed:    markedAsFailed,
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
	testcase(canaried, expectSync(expectDeleteCanaryRules(m(true, false, false))))
	testcase(pendingrelease, expectSync(expectDeleteCanaryRules(m(true, false, true))))
	testcase(failing, m(true, true, false))
	testcase(failing, m(true, true, true))
}

func TestPendingRelease(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, markedAsFailed, isReleaseEligible, schedulePermitsRelease bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:            hasRevision,
			markedAsFailed:         markedAsFailed,
			isReleaseEligible:      isReleaseEligible,
			schedulePermitsRelease: schedulePermitsRelease,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "pendingrelease", expected, mock)
	}

	testcase(deleting, m(false, false, false, false))
	testcase(deleting, m(false, false, false, true))
	testcase(deleting, m(false, false, true, false))
	testcase(deleting, m(false, false, true, true))
	testcase(deleting, m(false, true, false, false))
	testcase(deleting, m(false, true, false, true))
	testcase(deleting, m(false, true, true, false))
	testcase(deleting, m(false, true, true, true))

	testcase(failing, m(true, true, false, false))
	testcase(failing, m(true, true, false, true))
	testcase(failing, m(true, true, true, false))
	testcase(failing, m(true, true, true, true))

	testcase(retiring, m(true, false, false, true))
	testcase(retiring, m(true, false, false, false))

	testcase(releasing, m(true, false, true, true))

	testcase(pendingrelease, expectSync(m(true, false, true, false)))
}

func TestReleasing(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, markedAsFailed, isReleaseEligible bool, peakPercent uint32) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:       hasRevision,
			markedAsFailed:    markedAsFailed,
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
}

func TestReleased(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, markedAsFailed, isReleaseEligible bool, peakPercent uint32) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:       hasRevision,
			markedAsFailed:    markedAsFailed,
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
}

func TestRetiring(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, markedAsFailed, isReleaseEligible bool, currentPercent uint32) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:       hasRevision,
			markedAsFailed:    markedAsFailed,
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
	testcase(retiring, m(false, false, false, 100))
	testcase(retiring, m(false, false, true, 100))
	testcase(retiring, m(false, true, false, 100))
	testcase(retiring, m(false, true, true, 100))
	testcase(retiring, m(false, false, false, 1))
	testcase(retiring, m(false, false, true, 1))
	testcase(retiring, m(false, true, false, 1))
	testcase(retiring, m(false, true, true, 1))

	testcase(failing, m(true, true, false, 0))
	testcase(failing, m(true, true, true, 0))
	testcase(retiring, m(true, true, false, 100))
	testcase(retiring, m(true, true, true, 100))
	testcase(retiring, m(true, true, false, 1))
	testcase(retiring, m(true, true, true, 1))

	testcase(retired, expectRetire(expectDeleteTaggedServiceLevels(m(true, false, false, 0))))
	testcase(retiring, m(true, false, false, 1))
	testcase(retiring, m(true, false, false, 100))

	testcase(deploying, m(true, false, true, 0))
	testcase(retiring, m(true, false, true, 100))
	testcase(retiring, m(true, false, true, 1))
}

func TestRetired(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, markedAsFailed, isReleaseEligible bool) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:       hasRevision,
			markedAsFailed:    markedAsFailed,
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

	testcase(deleting, m(false, 100))
	testcase(deleting, m(false, 1))
	testcase(deleted, expectDeleteCanaryRules(expectDeleteTaggedServiceLevels(expectDelete(m(false, 0)))))
	testcase(deploying, m(true, 0))
	testcase(deleting, m(true, 100))
	testcase(deleting, m(true, 1))
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

	m := func(hasRevision, markedAsFailed bool, externalTestStatus ExternalTestStatus, currentPercent uint32) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:        hasRevision,
			markedAsFailed:     markedAsFailed,
			externalTestStatus: externalTestStatus,
			currentPercent:     currentPercent,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "failing", expected, mock)
	}

	testcase(deleting, m(false, true, ExternalTestUnknown, 0))
	testcase(failing, m(false, true, ExternalTestUnknown, 100))
	testcase(deleting, m(false, false, ExternalTestUnknown, 0))
	testcase(failing, m(false, false, ExternalTestUnknown, 100))

	testcase(deploying, m(true, false, ExternalTestDisabled, 0))
	testcase(failing, m(true, false, ExternalTestDisabled, 100))
	testcase(failing, m(true, false, ExternalTestDisabled, 1))
	testcase(deploying, m(true, false, ExternalTestSucceeded, 0))
	testcase(failing, m(true, false, ExternalTestSucceeded, 100))

	testcase(deploying, m(true, false, ExternalTestSucceeded, 0))
	testcase(failing, m(true, false, ExternalTestSucceeded, 100))

	testcase(failed, expectDeleteCanaryRules(expectDeleteTaggedServiceLevels(expectRetire(m(true, true, ExternalTestDisabled, 0)))))
	testcase(failing, m(true, true, ExternalTestDisabled, 1))
	testcase(failing, m(true, true, ExternalTestDisabled, 100))
	testcase(failed, expectDeleteCanaryRules(expectDeleteTaggedServiceLevels(expectRetire(m(true, false, ExternalTestFailed, 0)))))
	testcase(failing, m(true, false, ExternalTestFailed, 1))
	testcase(failing, m(true, false, ExternalTestFailed, 100))
}

func TestFailed(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, markedAsFailed bool, externalTestStatus ExternalTestStatus) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:        hasRevision,
			markedAsFailed:     markedAsFailed,
			externalTestStatus: externalTestStatus,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		testHandler(ctx, t, "failed", expected, mock)
	}

	testcase(deleting, m(false, true, ExternalTestUnknown))
	testcase(deleting, m(false, false, ExternalTestUnknown))

	testcase(deploying, m(true, false, ExternalTestDisabled))
	testcase(deploying, m(true, false, ExternalTestPending))
	testcase(deploying, m(true, false, ExternalTestStarted))
	testcase(deploying, m(true, false, ExternalTestSucceeded))

	testcase(failed, expectRetire(m(true, true, ExternalTestSucceeded)))
	testcase(failed, expectRetire(m(true, false, ExternalTestFailed)))
}

func testHandler(ctx context.Context, t *tt.T, handler string, expected State, m *MockDeployment) {
	state, err := handlers[handler](ctx, m)
	assert.NoError(t, err)
	assert.Equal(t, expected, state)
}

type responses struct {
	hasRevision               bool
	markedAsFailed            bool
	isReleaseEligible         bool
	externalTestStatus        ExternalTestStatus
	isCanaryPending           bool
	isDeployed                bool
	schedulePermitsRelease    bool
	currentPercent            uint32
	peakPercent               uint32
	syncCanaryRules           error
	deleteCanaryRules         error
	syncTaggedServiceLevels   error
	deleteTaggedServiceLevels error
	isTimingOut               bool
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
		markedAsFailed().
		Return(r.markedAsFailed).
		AnyTimes()
	m.
		EXPECT().
		getExternalTestStatus().
		Return(r.externalTestStatus).
		AnyTimes()
	m.
		EXPECT().
		isCanaryPending().
		Return(r.isCanaryPending).
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
	m.
		EXPECT().
		isTimingOut().
		Return(r.isTimingOut).
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

func expectSyncServiceLevels(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		syncServiceLevels(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}

func expectDeleteServiceLevels(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		deleteServiceLevels(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}

func expectSyncTaggedServiceLevels(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		syncTaggedServiceLevels(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}

func expectDeleteTaggedServiceLevels(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		deleteTaggedServiceLevels(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}
