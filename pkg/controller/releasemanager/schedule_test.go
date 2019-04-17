package releasemanager

import (
	"testing"
	"time"
)

func TestSchedulePermitsRelease(t *testing.T) {
	parse := func(isoDatetime string) time.Time {
		tm, err := time.Parse(time.RFC3339, isoDatetime)
		if err != nil {
			panic(err)
		}
		return tm
	}

	cases := []struct {
		Time     time.Time
		Expected bool
	}{
		{parse("2019-04-17T16:59:00Z"), false},
		{parse("2019-04-17T17:00:00Z"), true},
		{parse("2019-04-17T23:59:00Z"), true},
		{parse("2019-04-18T00:00:00Z"), false},
		{parse("2019-04-19T21:59:00Z"), true},
		{parse("2019-04-19T22:00:00Z"), false},
		{parse("2019-04-20T18:00:00Z"), false},
		{parse("2019-04-21T18:00:00Z"), false},
		{parse("2019-04-22T16:59:00Z"), false},
		{parse("2019-04-22T17:00:00Z"), true},
	}
	for _, c := range cases {
		permitted := schedulePermitsRelease(c.Time, "humane")
		if permitted != c.Expected {
			t.Errorf("schedulePermitsRelease(%q) was %v; expected %v", c.Time, permitted, c.Expected)
		}
	}

	if !schedulePermitsRelease(time.Now(), "indifferent") {
		t.Error("schedulePermitsRelease() was false for non-humane schedule")
	}
}
