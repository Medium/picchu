package garbagecollector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testRevision struct {
	ttl       time.Duration
	state     string
	createdOn time.Time
}

func (t *testRevision) TTL() time.Duration {
	return t.ttl
}

func (t *testRevision) State() string {
	return t.state
}

func (t *testRevision) CreatedOn() time.Time {
	return t.createdOn
}

func TestSort(t *testing.T) {
	in := []Revision{
		&testRevision{createdOn: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
		&testRevision{createdOn: time.Date(2010, time.January, 1, 0, 0, 0, 0, time.UTC)},
		&testRevision{createdOn: time.Date(2008, time.January, 1, 0, 0, 0, 0, time.UTC)},
		&testRevision{createdOn: time.Date(2007, time.January, 1, 0, 0, 0, 0, time.UTC)},
		&testRevision{createdOn: time.Date(2012, time.January, 1, 0, 0, 0, 0, time.UTC)},
		&testRevision{createdOn: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)},
	}

	out := sortRevisions(in)
	assert.Equal(t, 6, len(out), "all revisions are included")
	assert.Equal(t, 2020, out[0].CreatedOn().Year(), "Revision order is correct")
	assert.Equal(t, 2012, out[1].CreatedOn().Year(), "Revision order is correct")
	assert.Equal(t, 2010, out[2].CreatedOn().Year(), "Revision order is correct")
	assert.Equal(t, 2008, out[3].CreatedOn().Year(), "Revision order is correct")
	assert.Equal(t, 2007, out[4].CreatedOn().Year(), "Revision order is correct")
	assert.Equal(t, 2000, out[5].CreatedOn().Year(), "Revision order is correct")
}
