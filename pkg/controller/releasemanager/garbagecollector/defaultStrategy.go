package garbagecollector

import (
	"time"

	"github.com/go-logr/logr"
)

const FailedTTL = time.Duration(8) * time.Hour

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
		if rev.State() == "failed" {
			if time.Now().After(rev.CreatedOn().Add(FailedTTL)) {
				toDelete = append(toDelete, rev)
				continue
			}
		}
		if time.Now().After(rev.CreatedOn().Add(rev.TTL())) {
			switch rev.State() {
			case "pendingrelease", "releasing", "released":
				continue
			default:
				toDelete = append(toDelete, rev)
			}
		}

	}

	// Don't delete all the revisions
	if len(revisions) == len(toDelete) {
		return toDelete[:len(toDelete)-1], nil
	}

	return toDelete, nil
}
