package garbagecollector

import "sort"

// sortRevisions sorts the revisions in reverse chronological order
func sortRevisions(revisions []Revision) []Revision {
	r := make([]Revision, len(revisions))
	r = append(r[:0:0], revisions...)
	sort.Slice(r, func(x, y int) bool {
		a := r[x].CreatedOn()
		b := r[y].CreatedOn()
		return a.After(b)
	})
	return r
}
