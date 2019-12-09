package garbagecollector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMissingStrategy(t *testing.T) {
	_, err := FindGarbage(nil, 2, []Revision{})
	assert.Error(t, err, "invalid strategy fails")
}
