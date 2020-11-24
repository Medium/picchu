package releasemanager

import (
	tt "testing"
	"time"
)

func TestSchedulePermitsRelease(t *tt.T) {
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
		permitted := schedulePermitsRelease(c.Time, "humane")
		if permitted != c.Expected {
			t.Errorf("schedulePermitsRelease(%q) was %v; expected %v", c.Time, permitted, c.Expected)
		}
	}

	if !schedulePermitsRelease(time.Now(), "indifferent") {
		t.Error("schedulePermitsRelease() was false for non-humane schedule")
	}
}
