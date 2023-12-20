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
		date(2020, time.June, 19),     // Juneteenth
		date(2020, time.July, 3),      // Day before Independence day
		date(2020, time.July, 4),      // Saturday, Independence day
		date(2020, time.July, 5),      // Sunday, Day after Independence day
		date(2020, time.September, 7), // Labor day
		date(2020, time.November, 26), // Thanksgiving
		date(2020, time.November, 27), // Day after thanksgiving
		date(2020, time.December, 17), // Holiday Freeze
		date(2020, time.December, 18), // Holiday Freeze
		date(2020, time.December, 21), // Holiday break
		date(2020, time.December, 22), // Holiday break
		date(2020, time.December, 23), // Holiday break
		date(2020, time.December, 24), // Holiday break
		date(2020, time.December, 25), // Holiday break
		date(2020, time.December, 26), // Sat, Holiday break
		date(2020, time.December, 27), // Sunday, Holiday break
		date(2020, time.December, 28), // Holiday break
		date(2020, time.December, 29), // Holiday break
		date(2020, time.December, 30), // Holiday break
		date(2020, time.December, 31), // Holiday break

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

		date(2023, time.January, 1),
		date(2023, time.December, 20),  // 2023 Holiday break "code freeze"
		date(2023, time.December, 21),
		date(2023, time.December, 22),
		date(2023, time.December, 23),
		date(2023, time.December, 24),
		date(2023, time.December, 25),
		date(2023, time.December, 26),
		date(2023, time.December, 27),
		date(2023, time.December, 28),
		date(2023, time.December, 29),
		date(2023, time.December, 30),
		date(2023, time.December, 31),

		date(2024, time.January, 1),
	}
}

func PermitsRelease(t time.Time, schedule string) bool {
	switch schedule {
	case "humane", "Humane":
		return humanePermitsRelease(t)
	case "inhumane", "Inhumane":
		return inhumanePermitsRelease(t)
	default:
		return humanePermitsRelease(t)
	}
}
