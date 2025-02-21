package controllers

// TODO(bob): errors on the deployment interface aren't tested here, and should be

import (
	"context"
	tt "testing"
	"time"

	"go.medium.engineering/picchu/test"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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
		testHandler(ctx, t, "created", expected, mock, nil)
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

	testcase := func(expected State, mock *MockDeployment, lu *time.Time) {
		testHandler(ctx, t, "deploying", expected, mock, lu)
	}

	now := time.Now()
	old := time.Now().Add(-time.Duration(4) * time.Hour)

	testcase(deleting, m(false, false, false), &now)
	testcase(deleting, m(false, false, true), &now)
	testcase(deleting, m(false, true, false), &now)
	testcase(deleting, m(false, true, true), &now)
	testcase(failing, m(true, true, false), &now)
	testcase(failing, m(true, true, true), &now)

	testcase(deleting, m(false, false, false), nil)
	testcase(deleting, m(false, false, true), nil)
	testcase(deleting, m(false, true, false), nil)
	testcase(deleting, m(false, true, true), nil)
	testcase(failing, m(true, true, false), nil)
	testcase(failing, m(true, true, true), nil)

	testcase(deploying, expectSync(m(true, false, false)), nil)
	testcase(timingout, expectSync(m(true, false, false)), &old)

	testcase(deployed, expectSync(m(true, false, true)), nil)
}

func TestDeployed(t *tt.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	m := func(hasRevision, markedAsFailed, isDeployed, isReleaseEligible, isCanaryPending, isExpired bool, externalTestStatus ExternalTestStatus) *MockDeployment {
		return createMockDeployment(ctrl, responses{
			hasRevision:        hasRevision,
			markedAsFailed:     markedAsFailed,
			isDeployed:         isDeployed,
			isReleaseEligible:  isReleaseEligible,
			isCanaryPending:    isCanaryPending,
			externalTestStatus: externalTestStatus,
			isExpired:          isExpired,
		})
	}

	testcase := func(expected State, mock *MockDeployment) {
		t.Run("testcase", func(t *tt.T) {
			testHandler(ctx, t, "deployed", expected, mock, nil)
		})
	}

	// TODO(lyra): clean up; when !hasRevision, only ExternalTestUnknown is possible
	testcase(deleting, m(false, false, false, false, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, false, true, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, false, false, false, false, ExternalTestPending))
	testcase(deleting, m(false, false, false, true, false, false, ExternalTestPending))
	testcase(deleting, m(false, false, true, false, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, true, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, false, false, false, ExternalTestPending))
	testcase(deleting, m(false, false, true, true, false, false, ExternalTestPending))
	testcase(deleting, m(false, false, true, false, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, true, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, false, false, false, ExternalTestPending))
	testcase(deleting, m(false, false, true, true, false, false, ExternalTestPending))
	testcase(deleting, m(false, true, false, false, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, true, false, true, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, true, false, false, false, false, ExternalTestPending))
	testcase(deleting, m(false, true, false, true, false, false, ExternalTestPending))
	testcase(deleting, m(false, true, true, false, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, true, true, true, false, false, ExternalTestDisabled))
	testcase(deleting, m(false, true, true, false, false, false, ExternalTestPending))
	testcase(deleting, m(false, true, true, true, false, false, ExternalTestPending))

	testcase(failing, m(true, true, false, false, false, false, ExternalTestDisabled))
	testcase(failing, m(true, true, false, true, false, false, ExternalTestDisabled))
	testcase(failing, m(true, true, false, false, false, false, ExternalTestPending))
	testcase(failing, m(true, true, false, true, false, false, ExternalTestPending))
	testcase(failing, m(true, true, true, false, false, false, ExternalTestDisabled))
	testcase(failing, m(true, true, true, true, false, false, ExternalTestDisabled))
	testcase(failing, m(true, true, true, false, false, false, ExternalTestPending))
	testcase(failing, m(true, true, true, true, false, false, ExternalTestPending))

	testcase(deploying, expectSync(m(true, false, false, false, false, false, ExternalTestDisabled)))
	testcase(deploying, expectSync(m(true, false, false, true, false, false, ExternalTestDisabled)))
	testcase(deploying, expectSync(m(true, false, false, false, false, false, ExternalTestPending)))
	testcase(deploying, expectSync(m(true, false, false, true, false, false, ExternalTestPending)))

	testcase(deployed, expectSync(m(true, false, true, false, false, false, ExternalTestDisabled)))

	testcase(pendingtest, expectSync(m(true, false, true, true, false, false, ExternalTestPending)))
	testcase(pendingtest, expectSync(m(true, false, true, false, false, false, ExternalTestPending)))
	testcase(testing, expectSync(m(true, false, true, true, false, false, ExternalTestStarted)))
	testcase(testing, expectSync(m(true, false, true, false, false, false, ExternalTestStarted)))
	testcase(tested, expectSync(m(true, false, true, true, false, false, ExternalTestSucceeded)))
	testcase(tested, expectSync(m(true, false, true, false, false, false, ExternalTestSucceeded)))

	testcase(pendingrelease, expectSync(m(true, false, true, true, false, false, ExternalTestDisabled)))

	// now with canaries
	testcase(deleting, m(false, false, false, false, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, false, true, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, false, false, true, false, ExternalTestPending))
	testcase(deleting, m(false, false, false, true, true, false, ExternalTestPending))
	testcase(deleting, m(false, false, true, false, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, true, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, false, true, false, ExternalTestPending))
	testcase(deleting, m(false, false, true, true, true, false, ExternalTestPending))
	testcase(deleting, m(false, false, true, false, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, true, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, false, true, false, ExternalTestPending))
	testcase(deleting, m(false, false, true, true, true, false, ExternalTestPending))
	testcase(deleting, m(false, true, false, false, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, true, false, true, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, true, false, false, true, false, ExternalTestPending))
	testcase(deleting, m(false, true, false, true, true, false, ExternalTestPending))
	testcase(deleting, m(false, true, true, false, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, true, true, true, true, false, ExternalTestDisabled))
	testcase(deleting, m(false, true, true, false, true, false, ExternalTestPending))
	testcase(deleting, m(false, true, true, true, true, false, ExternalTestPending))

	testcase(failing, m(true, true, false, false, true, false, ExternalTestDisabled))
	testcase(failing, m(true, true, false, true, true, false, ExternalTestDisabled))
	testcase(failing, m(true, true, false, false, true, false, ExternalTestPending))
	testcase(failing, m(true, true, false, true, true, false, ExternalTestPending))
	testcase(failing, m(true, true, true, false, true, false, ExternalTestDisabled))
	testcase(failing, m(true, true, true, true, true, false, ExternalTestDisabled))
	testcase(failing, m(true, true, true, false, true, false, ExternalTestPending))
	testcase(failing, m(true, true, true, true, true, false, ExternalTestPending))
	testcase(failing, m(true, false, true, true, true, false, ExternalTestFailed))
	testcase(failing, m(true, false, true, true, false, false, ExternalTestFailed))

	testcase(deploying, expectSync(m(true, false, false, false, true, false, ExternalTestDisabled)))
	testcase(deploying, expectSync(m(true, false, false, true, true, false, ExternalTestDisabled)))
	testcase(deploying, expectSync(m(true, false, false, false, true, false, ExternalTestPending)))
	testcase(deploying, expectSync(m(true, false, false, true, true, false, ExternalTestPending)))

	testcase(deployed, expectSync(m(true, false, true, false, true, false, ExternalTestDisabled)))

	testcase(pendingtest, expectSync(m(true, false, true, true, true, false, ExternalTestPending)))
	testcase(pendingtest, expectSync(m(true, false, true, false, true, false, ExternalTestPending)))

	testcase(canarying, expectSync(m(true, false, true, true, true, false, ExternalTestDisabled)))

	testcase(deleting, m(false, false, false, false, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, false, true, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, false, false, false, true, ExternalTestPending))
	testcase(deleting, m(false, false, false, true, false, true, ExternalTestPending))
	testcase(deleting, m(false, false, true, false, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, true, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, false, false, true, ExternalTestPending))
	testcase(deleting, m(false, false, true, true, false, true, ExternalTestPending))
	testcase(deleting, m(false, false, true, false, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, true, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, false, false, true, ExternalTestPending))
	testcase(deleting, m(false, false, true, true, false, true, ExternalTestPending))
	testcase(deleting, m(false, true, false, false, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, true, false, true, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, true, false, false, false, true, ExternalTestPending))
	testcase(deleting, m(false, true, false, true, false, true, ExternalTestPending))
	testcase(deleting, m(false, true, true, false, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, true, true, true, false, true, ExternalTestDisabled))
	testcase(deleting, m(false, true, true, false, false, true, ExternalTestPending))
	testcase(deleting, m(false, true, true, true, false, true, ExternalTestPending))

	testcase(failing, m(true, true, false, false, false, true, ExternalTestDisabled))
	testcase(failing, m(true, true, false, true, false, true, ExternalTestDisabled))
	testcase(failing, m(true, true, false, false, false, true, ExternalTestPending))
	testcase(failing, m(true, true, false, true, false, true, ExternalTestPending))
	testcase(failing, m(true, true, true, false, false, true, ExternalTestDisabled))
	testcase(failing, m(true, true, true, true, false, true, ExternalTestDisabled))
	testcase(failing, m(true, true, true, false, false, true, ExternalTestPending))
	testcase(failing, m(true, true, true, true, false, true, ExternalTestPending))

	testcase(deploying, expectSync(m(true, false, false, false, false, true, ExternalTestDisabled)))
	testcase(deploying, expectSync(m(true, false, false, true, false, true, ExternalTestDisabled)))
	testcase(deploying, expectSync(m(true, false, false, false, false, true, ExternalTestPending)))
	testcase(deploying, expectSync(m(true, false, false, true, false, true, ExternalTestPending)))

	testcase(retiring, expectSync(m(true, false, true, false, false, true, ExternalTestDisabled)))

	testcase(pendingtest, expectSync(m(true, false, true, true, false, true, ExternalTestPending)))
	testcase(pendingtest, expectSync(m(true, false, true, false, false, true, ExternalTestPending)))
	testcase(testing, expectSync(m(true, false, true, true, false, true, ExternalTestStarted)))
	testcase(testing, expectSync(m(true, false, true, false, false, true, ExternalTestStarted)))
	testcase(tested, expectSync(m(true, false, true, true, false, true, ExternalTestSucceeded)))
	testcase(tested, expectSync(m(true, false, true, false, false, true, ExternalTestSucceeded)))

	testcase(pendingrelease, expectSync(m(true, false, true, true, false, true, ExternalTestDisabled)))

	// now with canaries
	testcase(deleting, m(false, false, false, false, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, false, true, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, false, false, true, true, ExternalTestPending))
	testcase(deleting, m(false, false, false, true, true, true, ExternalTestPending))
	testcase(deleting, m(false, false, true, false, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, true, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, false, true, true, ExternalTestPending))
	testcase(deleting, m(false, false, true, true, true, true, ExternalTestPending))
	testcase(deleting, m(false, false, true, false, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, true, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, false, true, false, true, true, ExternalTestPending))
	testcase(deleting, m(false, false, true, true, true, true, ExternalTestPending))
	testcase(deleting, m(false, true, false, false, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, true, false, true, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, true, false, false, true, true, ExternalTestPending))
	testcase(deleting, m(false, true, false, true, true, true, ExternalTestPending))
	testcase(deleting, m(false, true, true, false, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, true, true, true, true, true, ExternalTestDisabled))
	testcase(deleting, m(false, true, true, false, true, true, ExternalTestPending))
	testcase(deleting, m(false, true, true, true, true, true, ExternalTestPending))

	testcase(failing, m(true, true, false, false, true, true, ExternalTestDisabled))
	testcase(failing, m(true, true, false, true, true, true, ExternalTestDisabled))
	testcase(failing, m(true, true, false, false, true, true, ExternalTestPending))
	testcase(failing, m(true, true, false, true, true, true, ExternalTestPending))
	testcase(failing, m(true, true, true, false, true, true, ExternalTestDisabled))
	testcase(failing, m(true, true, true, true, true, true, ExternalTestDisabled))
	testcase(failing, m(true, true, true, false, true, true, ExternalTestPending))
	testcase(failing, m(true, true, true, true, true, true, ExternalTestPending))
	testcase(failing, m(true, false, true, true, true, true, ExternalTestFailed))
	testcase(failing, m(true, false, true, true, false, true, ExternalTestFailed))

	testcase(deploying, expectSync(m(true, false, false, false, true, true, ExternalTestDisabled)))
	testcase(deploying, expectSync(m(true, false, false, true, true, true, ExternalTestDisabled)))
	testcase(deploying, expectSync(m(true, false, false, false, true, true, ExternalTestPending)))
	testcase(deploying, expectSync(m(true, false, false, true, true, true, ExternalTestPending)))

	testcase(retiring, expectSync(m(true, false, true, false, true, true, ExternalTestDisabled)))

	testcase(pendingtest, expectSync(m(true, false, true, true, true, true, ExternalTestPending)))
	testcase(pendingtest, expectSync(m(true, false, true, false, true, true, ExternalTestPending)))

	testcase(canarying, expectSync(m(true, false, true, true, true, true, ExternalTestDisabled)))
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
		testHandler(ctx, t, "pendingtest", expected, mock, nil)
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
		testHandler(ctx, t, "testing", expected, mock, nil)
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

	testcase(pendingtest, m(true, true, ExternalTestPending))
	testcase(testing, expectSync(m(true, true, ExternalTestStarted)))
	testcase(tested, m(true, true, ExternalTestSucceeded))
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
		testHandler(ctx, t, "tested", expected, mock, nil)
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
	testcase(tested, m(true, false, false, true, ExternalTestSucceeded))
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
		testHandler(ctx, t, "canarying", expected, mock, nil)
	}

	testcase(deleting, m(false, false, false))
	testcase(deleting, m(false, false, true))
	testcase(deleting, m(false, true, false))
	testcase(deleting, m(false, true, true))
	testcase(canaried, expectSync(expectSyncDatadogSLOs(expectSyncTaggedServiceLevels(expectSyncDatadogCanarySLOs(expectSyncCanaryRules(m(true, false, false)))))))
	testcase(canarying, expectSync(expectSyncDatadogSLOs(expectSyncTaggedServiceLevels(expectSyncDatadogCanarySLOs(expectSyncCanaryRules(m(true, false, true)))))))
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
		testHandler(ctx, t, "canaried", expected, mock, nil)
	}

	testcase(deleting, m(false, false, false))
	testcase(deleting, m(false, false, true))
	testcase(deleting, m(false, true, false))
	testcase(deleting, m(false, true, true))
	testcase(canaried, expectSync(expectDeleteDatadogCanarySLOs(expectDeleteCanaryRules(m(true, false, false)))))
	testcase(pendingrelease, expectSync(expectDeleteDatadogCanarySLOs(expectDeleteCanaryRules(m(true, false, true)))))
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
		testHandler(ctx, t, "pendingrelease", expected, mock, nil)
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
		testHandler(ctx, t, "releasing", expected, mock, nil)
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
		testHandler(ctx, t, "released", expected, mock, nil)
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
		testHandler(ctx, t, "retiring", expected, mock, nil)
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

	testcase(retired, expectRetire(expectDeleteTaggedServiceLevels(expectDeleteDatadogSLOs(m(true, false, false, 0)))))
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
		testHandler(ctx, t, "retired", expected, mock, nil)
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
		testHandler(ctx, t, "deleting", expected, mock, nil)
	}

	testcase(deleting, m(false, 100))
	testcase(deleting, m(false, 1))
	testcase(deleted, expectDeleteCanaryRules(expectDeleteDatadogCanarySLOs(expectDeleteTaggedServiceLevels(expectDeleteDatadogSLOs(expectDelete(m(false, 0)))))))
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
		testHandler(ctx, t, "deleted", expected, mock, nil)
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
		testHandler(ctx, t, "failing", expected, mock, nil)
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

	// ddog eventually??
	testcase(failed, expectDeleteCanaryRules(expectDeleteDatadogCanarySLOs(expectDeleteTaggedServiceLevels(expectDeleteDatadogSLOs(expectRetire(m(true, true, ExternalTestDisabled, 0)))))))
	testcase(failing, m(true, true, ExternalTestDisabled, 1))
	testcase(failing, m(true, true, ExternalTestDisabled, 100))
	testcase(failed, expectDeleteCanaryRules(expectDeleteDatadogCanarySLOs(expectDeleteTaggedServiceLevels(expectDeleteDatadogSLOs(expectRetire(m(true, false, ExternalTestFailed, 0)))))))
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
		testHandler(ctx, t, "failed", expected, mock, nil)
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

func TestTimingout(t *tt.T) {
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
		testHandler(ctx, t, "timingout", expected, mock, nil)
	}

	testcase(failing, m(true, true, ExternalTestSucceeded))
	testcase(failing, m(true, false, ExternalTestFailed))
	testcase(timingout, m(true, false, ExternalTestPending))
}

func testHandler(ctx context.Context, t *tt.T, handler string, expected State, m *MockDeployment, lu *time.Time) {
	state, err := handlers[handler](ctx, m, lu)
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
	syncDatadogSLOs           error
	deleteTaggedServiceLevels error
	deleteDatadogSLOs         error
	isTimingOut               bool
	isExpired                 bool
}

func createMockDeployment(ctrl *gomock.Controller, r responses) *MockDeployment {
	m := NewMockDeployment(ctrl)
	log := test.MustNewLogger()

	m.
		EXPECT().
		getLog().
		Return(log).
		AnyTimes()
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
	m.
		EXPECT().
		isExpired().
		Return(r.isExpired).
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

func expectSyncDatadogCanarySLOs(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		syncDatadogCanarySLOs(gomock.Any()).
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

func expectDeleteDatadogCanarySLOs(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		deleteDatadogCanarySLOs(gomock.Any()).
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

func expectSyncDatadogSLOs(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		syncDatadogSLOs(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}

func expectDeleteDatadogSLOs(mock *MockDeployment) *MockDeployment {
	mock.
		EXPECT().
		deleteDatadogSLOs(gomock.Any()).
		Return(nil).
		Times(1)
	return mock
}
