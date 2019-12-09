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
			case "pendingrelease", "releasing", "released":
				continue
			default:
				toDelete = append(toDelete, rev)
			}
		}

	}
	return toDelete, nil
}
