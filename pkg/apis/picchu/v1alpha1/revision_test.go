package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFailRevision(t *testing.T) {
	r := Revision{}
	r.Fail()
	failedAt, ok := r.Annotations[AnnotationFailedAt]
	assert.True(t, ok, "FailedAt not set")
	d := r.SinceFailed()
	assert.NotEqual(t, d, time.Duration(0), "SinceFailed reports 0")
	r.Fail()
	failedAtNow := r.Annotations[AnnotationFailedAt]
	assert.Equal(t, failedAt, failedAtNow, "FailedAt shouldn't update on subsequent calls")
}
