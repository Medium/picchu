package schedule

import "time"

func inhumanePermitsRelease(t time.Time) bool {
	day := date(t.Year(), t.Month(), t.Day())
	for _, holiday := range holidays {
		if holiday == day {
			return false
		}
	}

	switch t.Weekday() {
	case time.Saturday, time.Sunday:
		return false
	default:
		return true
	}
}
