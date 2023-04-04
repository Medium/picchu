package schedule

import (
	"testing"
	tt "testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func parse(isoDatetime string) time.Time {
	tm, err := time.Parse(time.RFC3339, isoDatetime)
	if err != nil {
		panic(err)
	}
	return tm
}

func TestSchedulePermitsRelease(t *tt.T) {
	cases := []struct {
		Time     time.Time
		Expected bool
	}{
		{parse("2019-04-17T13:59:00Z"), false},
		{parse("2019-04-17T14:00:00Z"), true},
		{parse("2019-04-17T17:00:00Z"), true},
		{parse("2019-04-17T22:59:00Z"), true},
		{parse("2019-04-17T23:00:00Z"), false},
		{parse("2019-04-18T00:00:00Z"), false},
		{parse("2019-04-19T20:59:00Z"), true},
		{parse("2019-04-19T21:00:00Z"), false},
		{parse("2019-04-20T18:00:00Z"), false},
		{parse("2019-04-21T18:00:00Z"), false},
		{parse("2019-04-22T13:59:00Z"), false},
		{parse("2019-04-22T14:00:00Z"), true},
		{parse("2020-07-05T15:00:00Z"), false},
		{parse("2020-09-06T15:00:00Z"), false},
		{parse("2020-11-25T21:59:00Z"), true},
		{parse("2020-11-25T22:00:00Z"), false},
		{parse("2020-11-26T00:00:00Z"), false},
		{parse("2020-11-26T14:59:00Z"), false},
		{parse("2020-11-26T15:00:00Z"), false},
		{parse("2020-11-26T23:59:00Z"), false},
		{parse("2020-11-27T00:00:00Z"), false},
		{parse("2020-11-27T15:00:00Z"), false},
		{parse("2020-11-27T23:59:00Z"), false},
		{parse("2020-11-30T14:59:00Z"), false},
		{parse("2020-11-30T15:00:00Z"), true},
	}
	for _, c := range cases {
		permitted := PermitsRelease(c.Time, "humane")
		if permitted != c.Expected {
			t.Errorf("schedulePermitsRelease(%q) was %v; expected %v", c.Time, permitted, c.Expected)
		}
	}

	if !PermitsRelease(time.Now(), "indifferent") {
		t.Error("schedulePermitsRelease() was false for non-humane schedule")
	}
}

// Use this one for future tests
func TestPermitsRelease(t *tt.T) {
	for _, c := range []struct {
		Name      string
		Timestamp time.Time
		Schedule  string
		Expected  bool
	}{
		{
			Name:      "friday night inhumane",
			Timestamp: parse("2019-04-19T23:59:59Z"),
			Schedule:  "inhumane",
			Expected:  true,
		},
		{
			Name:      "saturday morning inhumane",
			Timestamp: parse("2019-04-20T00:00:00Z"),
			Schedule:  "inhumane",
			Expected:  false,
		},
		{
			Name:      "sunday night inhumane",
			Timestamp: parse("2019-04-21T23:59:59Z"),
			Schedule:  "inhumane",
			Expected:  false,
		},
		{
			Name:      "monday morning inhumane",
			Timestamp: parse("2019-04-22T00:00:00Z"),
			Schedule:  "inhumane",
			Expected:  true,
		},
		{
			Name:      "pre-thanksgiving day inhumane",
			Timestamp: parse("2021-11-24T23:59:59Z"),
			Schedule:  "inhumane",
			Expected:  true,
		},
		{
			Name:      "labor day morning inhumane",
			Timestamp: parse("2021-09-06T00:00:00Z"),
			Schedule:  "inhumane",
			Expected:  false,
		},
		{
			Name:      "labor day night inhumane",
			Timestamp: parse("2021-09-06T23:59:59Z"),
			Schedule:  "inhumane",
			Expected:  false,
		},
		{
			Name:      "post-labor day inhumane",
			Timestamp: parse("2021-09-07T00:00:00Z"),
			Schedule:  "inhumane",
			Expected:  true,
		},
	} {
		t.Run(c.Name, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(c.Expected, PermitsRelease(c.Timestamp, c.Schedule))
		})
	}
}
