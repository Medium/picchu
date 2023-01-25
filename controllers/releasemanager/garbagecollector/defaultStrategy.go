package garbagecollector

import (
	"time"

	"github.com/go-logr/logr"
)

func defaultStrategy(log logr.Logger, revisions []Revision) ([]Revision, error) {
	budget := MinimumRetired
	toDelete := []Revision{}
	sorted := sortRevisions(revisions)
	for i := range sorted {
		rev := sorted[i]
		if rev.State() == "retired" && budget > 0 {
			budget--
			continue
		}
		if time.Now().After(rev.CreatedOn().Add(rev.TTL())) {
			switch rev.State() {
			case "retired", "deleting", "deleted", "failed":
				toDelete = append(toDelete, rev)
			default:
				continue
			}
		}

	}

	// Don't delete all the revisions
	if len(revisions) == len(toDelete) {
		return toDelete[:len(toDelete)-1], nil
	}

	return toDelete, nil
}
