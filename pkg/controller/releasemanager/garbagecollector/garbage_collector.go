package garbagecollector

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

func TestMissingStrategy(t *testing.T) {
	testLogger := logr.Logger{}
	_, err := FindGarbage(testLogger, 2, []Revision{})
	assert.Error(t, err, "invalid strategy fails")
}
