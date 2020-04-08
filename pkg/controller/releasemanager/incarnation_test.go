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

func TestIncarnation_targetScale(t *ttesting.T) {
	testIncarnation := createTestIncarnation("test", testing, 10)
	testIncarnation.status.State.Current = "releasing"
	testIncarnation.revision.Spec.Targets[0].Release.Rate.Increment = 10
	testIncarnation.status.CurrentPercent = 0
	assert.Equal(t, 1.0, testIncarnation.targetScale())
	testIncarnation.status.CurrentPercent = 10
	assert.Equal(t, 0.5, testIncarnation.targetScale())
	testIncarnation.status.CurrentPercent = 90
	assert.Equal(t, 0.9, testIncarnation.targetScale())
	testIncarnation.status.CurrentPercent = 100
	assert.Equal(t, 1.0, testIncarnation.targetScale())
}

func TestIncarnation_PortInfoMerging(t *ttesting.T) {
	i := createTestIncarnation("test", testing, 10)
	i.revision.Spec.Ports = []v1alpha1.PortInfo{
		{
			Name: "http",
			Port: 8080,
		},
		{
			Name: "status",
			Port: 4444,
		},
	}
	assert.Equal(t, 1, len(i.revision.Spec.Targets))
	i.revision.Spec.Targets[0].Ports = []v1alpha1.PortInfo{
		{
			Name:  "http",
			Hosts: []string{"www.medm.io"},
		},
		{
			Name:  "grpc",
			Hosts: []string{"grpc.medm.io"},
		},
	}

	for j := range i.revision.Spec.Ports {
		v1alpha1.SetPortDefaults(&i.revision.Spec.Ports[j])
	}
	for j := range i.revision.Spec.Targets[0].Ports {
		v1alpha1.SetPortDefaults(&i.revision.Spec.Targets[0].Ports[j])
	}

	ports := i.ports()
	assert.Equal(t, 3, len(ports))
	for _, port := range ports {
		switch port.Name {
		case "http":
			assert.EqualValues(t, 8080, port.Port)
			assert.Equal(t, []string{"www.medm.io"}, port.Hosts)
		case "grpc":
			assert.EqualValues(t, 80, port.Port)
			assert.Equal(t, []string{"grpc.medm.io"}, port.Hosts)
		case "status":
			assert.EqualValues(t, 4444, port.Port)
			assert.Nil(t, port.Hosts)
		}
	}
}
