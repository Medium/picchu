package garbagecollector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// We use years simply to make the code easier to follow. Use time.Now().Year() + n for "non-expired" revisions
func rev(ttlHours int, year int, state string) Revision {
	return &testRevision{
		ttl:       time.Duration(ttlHours) * time.Hour,
		createdOn: time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC),
		state:     state,
	}
}
func TestDefaultStrategyAllExpired(t *testing.T) {
	// Test set of expired and retired revisions
	in := []Revision{
		rev(1, 2000, "retired"),
		rev(1, 2001, "retired"),
		rev(1, 2002, "retired"),
		rev(1, 2003, "retired"),
		rev(1, 2004, "retired"),
	}

	out, err := FindGarbage(nil, DefaultStrategy, in)
	assert.NoError(t, err, "error during find garbage")
	assert.Equal(t, 2, len(out), "correct number of revisions found")
	sortedOut := sortRevisions(out)
	assert.Equal(t, 2001, sortedOut[0].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2000, sortedOut[1].CreatedOn().Year(), "correct revisions found")
}

func TestDefaultStrategyVariousStates(t *testing.T) {
	// Test with lots of states
	in := []Revision{
		rev(1, 2000, "retired"),
		rev(1, 2001, "retired"),
		rev(1, 2002, "retired"),
		rev(1, 2008, "releasing"),
		rev(1, 2003, "retired"),
		rev(1, 2010, "released"),
		rev(1, 2009, "pendingrelease"),
		rev(1, 2011, "failed"),
		rev(1, 2012, "deployed"),
		rev(1, 2013, "canaried"),
		rev(1, 2014, "tested"),
		rev(1, 2015, "canarying"),
		rev(1, 2004, "retired"),
	}

	out, err := FindGarbage(nil, DefaultStrategy, in)
	assert.NoError(t, err, "error during find garbage")
	assert.Equal(t, 7, len(out), "correct number of revisions found")
	sortedOut := sortRevisions(out)
	assert.Equal(t, 2015, sortedOut[0].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2014, sortedOut[1].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2013, sortedOut[2].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2012, sortedOut[3].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2011, sortedOut[4].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2001, sortedOut[5].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2000, sortedOut[6].CreatedOn().Year(), "correct revisions found")
}

func TestDefaultStrategyNonExpiredRevisions(t *testing.T) {
	// test with some non-expired revisions to ensure they don't get deleted
	in := []Revision{
		rev(1, 2000, "retired"),
		rev(1, 2001, "retired"),
		rev(1, 2002, "retired"),
		rev(1, 2008, "releasing"),
		rev(1, 2003, "retired"),
		rev(1, 2010, "released"),
		rev(1, 2009, "pendingrelease"),
		rev(1, 2011, "failed"),
		rev(1, 2012, "deployed"),
		rev(1, 2013, "canaried"),
		rev(1, 2014, "tested"),
		rev(1, 2015, "canarying"),
		rev(1, 2004, "retired"),
		// undeletable because of ttl
		rev(1, time.Now().Year()+1, "retired"),
		rev(1, time.Now().Year()+2, "retired"),
		rev(1, time.Now().Year()+3, "retired"),
		rev(1, time.Now().Year()+4, "retired"),
		rev(1, time.Now().Year()+5, "deployed"),
	}
	out, err := FindGarbage(nil, DefaultStrategy, in)
	assert.NoError(t, err, "error during find garbage")
	assert.Equal(t, 10, len(out), "correct number of revisions found")
	sortedOut := sortRevisions(out)
	assert.Equal(t, 2015, sortedOut[0].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2014, sortedOut[1].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2013, sortedOut[2].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2012, sortedOut[3].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2011, sortedOut[4].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2004, sortedOut[5].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2003, sortedOut[6].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2002, sortedOut[7].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2001, sortedOut[8].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2000, sortedOut[9].CreatedOn().Year(), "correct revisions found")
}

func TestDefaultStrategyNotEnoughExpired(t *testing.T) {
	// test with not enough retired revisions
	in := []Revision{
		rev(1, 2000, "retired"),
		rev(1, 2008, "releasing"),
		rev(1, 2010, "released"),
		rev(1, 2009, "pendingrelease"),
		rev(1, 2011, "failed"),
		rev(1, 2012, "deployed"),
		rev(1, 2013, "canaried"),
		rev(1, 2014, "tested"),
		rev(1, 2015, "canarying"),
	}
	out, err := FindGarbage(nil, DefaultStrategy, in)
	assert.NoError(t, err, "error during find garbage")
	assert.Equal(t, 5, len(out), "correct number of revisions found")
	sortedOut := sortRevisions(out)
	assert.Equal(t, 2015, sortedOut[0].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2014, sortedOut[1].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2013, sortedOut[2].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2012, sortedOut[3].CreatedOn().Year(), "correct revisions found")
	assert.Equal(t, 2011, sortedOut[4].CreatedOn().Year(), "correct revisions found")
}

func TestDefaultStrategyFailedTTL(t *testing.T) {
	in := []Revision{
		&testRevision{
			ttl:       1 * time.Hour,
			createdOn: time.Now().Add(time.Hour * -1),
			state:     "failed",
		},
		&testRevision{
			ttl:       1 * time.Hour,
			createdOn: time.Now(),
			state:     "failed",
		},
		&testRevision{
			ttl:       1 * time.Hour,
			createdOn: time.Now(),
			state:     "retired",
		},
		&testRevision{
			ttl:       1 * time.Hour,
			createdOn: time.Now(),
			state:     "deployed",
		},
	}
	out, err := FindGarbage(nil, DefaultStrategy, in)
	assert.NoError(t, err, "error during find garbage")
	assert.Equal(t, 1, len(out), "correct number of revisions found")
	sortedOut := sortRevisions(out)
	assert.Equal(t, "failed", sortedOut[0].State(), "correct revisions found")
}

func TestDefaultStrategyDontDeleteAll(t *testing.T) {
	for _, test := range []struct {
		Name             string
		Revisions        []Revision
		ExpectedToDelete int
	}{
		{
			Name: "single-created",
			Revisions: []Revision{
				rev(1, 2011, "created"),
			},
			ExpectedToDelete: 0,
		},
		{
			Name: "double-created",
			Revisions: []Revision{
				rev(1, 2011, "created"),
				rev(1, 2012, "created"),
			},
			ExpectedToDelete: 1,
		},
		{
			Name: "single-deploying",
			Revisions: []Revision{
				rev(1, 2011, "deploying"),
			},
			ExpectedToDelete: 0,
		},
		{
			Name: "single-deployed",
			Revisions: []Revision{
				rev(1, 2011, "deployed"),
			},
			ExpectedToDelete: 0,
		},
		{
			Name: "single-pendingrelease",
			Revisions: []Revision{
				rev(1, 2011, "pendingrelease"),
			},
			ExpectedToDelete: 0,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			assert := assert.New(t)
			out, err := FindGarbage(nil, DefaultStrategy, test.Revisions)
			assert.NoError(err)
			assert.Len(out, test.ExpectedToDelete)
		})
	}
}
