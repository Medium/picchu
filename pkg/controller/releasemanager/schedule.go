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
	if schedule != "humane" && schedule != "Humane" {
		return true
	}

	t = t.In(scheduleLocation).Truncate(time.Hour)
	hour := t.Hour()

	switch t.Weekday() {
	case time.Saturday, time.Sunday:
		return false
	case time.Friday:
		return hour >= 7 && hour < 15
	default:
		return hour >= 7 && hour < 16
	}
}
