package releasemanager

import (
	ttesting "testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIncarnation_getExternalTestStatus(t *ttesting.T) {
	testIncarnation := createTestIncarnation("test", testing, 10)
	recentLastUpdated := v1.NewTime(time.Now().Add(-5 * time.Minute))
	oldLastUpdated := v1.NewTime(time.Now().Add(-15 * time.Minute))
	timeout := v1.Duration{10 * time.Minute}

	// ExternalTestDisabled
	assertExternalTestStatus(t, testIncarnation, v1alpha1.ExternalTest{
		Enabled:     false,
		Timeout:     &timeout,
		Started:     false,
		Completed:   false,
		Succeeded:   false,
		LastUpdated: nil,
	}, ExternalTestDisabled, false)

	// ExternalTestPending
	assertExternalTestStatus(t, testIncarnation, v1alpha1.ExternalTest{
		Enabled:     true,
		Timeout:     &timeout,
		Started:     false,
		Completed:   false,
		Succeeded:   false,
		LastUpdated: &recentLastUpdated,
	}, ExternalTestPending, false)

	// ExternalTestStarted
	assertExternalTestStatus(t, testIncarnation, v1alpha1.ExternalTest{
		Enabled:     true,
		Timeout:     &timeout,
		Started:     true,
		Completed:   false,
		Succeeded:   false,
		LastUpdated: &recentLastUpdated,
	}, ExternalTestStarted, false)

	// Timing out
	assertExternalTestStatus(t, testIncarnation, v1alpha1.ExternalTest{
		Enabled:     true,
		Timeout:     &timeout,
		Started:     true,
		Completed:   false,
		Succeeded:   false,
		LastUpdated: &oldLastUpdated,
	}, ExternalTestStarted, true)

	// ExternalTestFailed
	assertExternalTestStatus(t, testIncarnation, v1alpha1.ExternalTest{
		Enabled:     true,
		Timeout:     &timeout,
		Started:     true,
		Completed:   true,
		Succeeded:   false,
		LastUpdated: &recentLastUpdated,
	}, ExternalTestFailed, false)

	// ExternalTestSucceeded
	assertExternalTestStatus(t, testIncarnation, v1alpha1.ExternalTest{
		Enabled:     true,
		Timeout:     &timeout,
		Started:     true,
		Completed:   true,
		Succeeded:   true,
		LastUpdated: &oldLastUpdated,
	}, ExternalTestSucceeded, false)
}

func assertExternalTestStatus(
	t *ttesting.T,
	testIncarnation *Incarnation,
	externalTest v1alpha1.ExternalTest,
	expectedStatus ExternalTestStatus,
	expectedTimeout bool,
) {
	testIncarnation.revision.Spec.Targets[0].ExternalTest = externalTest
	assert.Equal(t, expectedStatus, testIncarnation.getExternalTestStatus())
	assert.Equal(t, expectedTimeout, testIncarnation.isTimingOut())
}
