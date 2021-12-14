package schedule

import (
	"fmt"
	"time"
)

const (
	defaultStartHour = 7
	defaultEndHour   = 16
	earlyEndHour     = 14
)

var (
	scheduleLocation *time.Location
	holidays         []time.Time
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, scheduleLocation)
}

func init() {
	// TODO(lyra): make configurable
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(fmt.Sprintf("failed to load schedule time zone: %v", err))
	}

	scheduleLocation = loc

	// TODO(bob): Make configurable
	holidays = []time.Time{

		date(2021, time.January, 1),   // New year's day
		date(2021, time.January, 18),  // MLK day
		date(2021, time.May, 31),      // Memorial day
		date(2021, time.June, 19),     // Juneteenth
		date(2021, time.July, 4),      // Sunday, Independence day
		date(2021, time.July, 5),      // Day after Independence day
		date(2021, time.September, 6), // Labor day
		date(2021, time.November, 25), // Thanksgiving
		date(2021, time.November, 26), // Day after Thanksgiving
		date(2021, time.December, 24), // Holiday break
		date(2021, time.December, 25), // Sat, Holiday break
		date(2021, time.December, 26), // Sun, Holiday break
		date(2021, time.December, 31), // Holiday break
	}
}

func PermitsRelease(t time.Time, schedule string) bool {
	switch schedule {
	case "humane", "Humane":
		return humanePermitsRelease(t)
	case "inhumane", "Inhumane":
		return inhumanePermitsRelease(t)
	default:
		return alwaysPermitsRelease(t)
	}
}
