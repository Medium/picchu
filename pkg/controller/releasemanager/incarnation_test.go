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

	// ExternalTestDisabled
	assertExternalTestStatus(t, testIncarnation, v1alpha1.ExternalTest{
		Enabled:     false,
		Started:     false,
		Completed:   false,
		Succeeded:   false,
		LastUpdated: nil,
	}, ExternalTestDisabled)

	// ExternalTestPending
	recentLastUpdated := v1.NewTime(time.Now().Add(-5 * time.Minute))
	assertExternalTestStatus(t, testIncarnation, v1alpha1.ExternalTest{
		Enabled:     true,
		Started:     false,
		Completed:   false,
		Succeeded:   false,
		LastUpdated: &recentLastUpdated,
	}, ExternalTestPending)

	// ExternalTestStarted
	assertExternalTestStatus(t, testIncarnation, v1alpha1.ExternalTest{
		Enabled:     true,
		Started:     true,
		Completed:   false,
		Succeeded:   false,
		LastUpdated: &recentLastUpdated,
	}, ExternalTestStarted)

	// ExternalTestFailed
	assertExternalTestStatus(t, testIncarnation, v1alpha1.ExternalTest{
		Enabled:     true,
		Started:     true,
		Completed:   true,
		Succeeded:   false,
		LastUpdated: &recentLastUpdated,
	}, ExternalTestFailed)

	// oldLastUpdated := v1.NewTime(time.Now().Add(-15 * time.Minute))
	// assertExternalTestStatus(t, testIncarnation, v1alpha1.ExternalTest{
	// 	Enabled:     true,
	// 	Started:     true,
	// 	Completed:   false,
	// 	Succeeded:   false,
	// 	LastUpdated: &oldLastUpdated,
	// }, ExternalTestFailed)

	// ExternalTestSucceeded
	assertExternalTestStatus(t, testIncarnation, v1alpha1.ExternalTest{
		Enabled:     true,
		Started:     true,
		Completed:   true,
		Succeeded:   true,
		LastUpdated: nil,
	}, ExternalTestSucceeded)
}

func assertExternalTestStatus(
	t *ttesting.T,
	testIncarnation *Incarnation,
	externalTest v1alpha1.ExternalTest,
	expectedStatus ExternalTestStatus,
) {
	testIncarnation.revision.Spec.Targets[0].ExternalTest = externalTest
	assert.Equal(t, expectedStatus, testIncarnation.getExternalTestStatus())
}
