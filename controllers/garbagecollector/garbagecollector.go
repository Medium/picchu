package garbagecollector

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
)

// Strategy is an enum for different strategies
type Strategy int

// Revision represents a picchu revision's state and ttl
type Revision interface {
	State() string
	TTL() time.Duration
	CreatedOn() time.Time
}

const (
	// MinimumRetired is the minimum amount of retired revisions we should keep for rollbacks
	MinimumRetired = 3
	// DefaultStrategy deletes any expired revisions, except if they are needed to fullfill the MimimumRetired quota
	DefaultStrategy Strategy = iota
)

// FindGarbage returns safe-to-delete revisions according to strategy
func FindGarbage(log logr.Logger, strategy Strategy, revisions []Revision) ([]Revision, error) {
	switch strategy {
	// Future expansion
	case DefaultStrategy:
		return defaultStrategy(log, revisions)
	}
	return nil, fmt.Errorf("Strategy not implemented")
}
