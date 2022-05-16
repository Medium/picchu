package garbagecollector

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

func TestMissingStrategy(t *testing.T) {
	tLog := logr.Logger{}
	_, err := FindGarbage(tLog, 2, []Revision{})
	assert.Error(t, err, "invalid strategy fails")
}
