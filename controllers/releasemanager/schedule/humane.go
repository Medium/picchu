package schedule

import "time"

func humanePermitsRelease(t time.Time) bool {
	day := date(t.Year(), t.Month(), t.Day())
	hour := t.In(scheduleLocation).Hour()
	tomorrow := day.Add(time.Hour * 24)
	for _, holiday := range holidays {
		if holiday == day {
			return false
		}
		if holiday == tomorrow && t.Weekday() != time.Saturday && t.Weekday() != time.Sunday {
			return hour >= defaultStartHour && hour < earlyEndHour
		}
	}

	switch t.Weekday() {
	case time.Saturday, time.Sunday:
		return false
	case time.Friday:
		return hour >= defaultStartHour && hour < earlyEndHour
	default:
		return hour >= defaultStartHour && hour < defaultEndHour
	}
}
