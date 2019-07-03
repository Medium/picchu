package releasemanager

import (
	"fmt"
	"time"
)

var (
	scheduleLocation *time.Location
)

func init() {
	// TODO(lyra): make configurable
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(fmt.Sprintf("failed to load schedule time zone: %v", err))
	}

	scheduleLocation = loc
}

// TODO(lyra): make configurable
// TODO(lyra): give schedule its own type
func schedulePermitsRelease(t time.Time, schedule string) bool {
	// FIXME(lyra): 2019-07-04 holiday hack
	// release apps with the "always" schedule as normal
	// don't release "humane" apps (i.e., lite, rito, tutu) automatically (OOB only)
	if schedule == "humane" || schedule == "Humane" {
		return false
	}

	t = t.In(scheduleLocation).Truncate(time.Hour)
	hour := t.Hour()

	switch t.Weekday() {
	case time.Saturday, time.Sunday:
		return false
	case time.Friday:
		return hour >= 10 && hour < 15
	default:
		return hour >= 10 && hour < 16
	}
}
