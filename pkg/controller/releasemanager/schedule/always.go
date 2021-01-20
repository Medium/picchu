package schedule

import "time"

func alwaysPermitsRelease(_ time.Time) bool {
	return true
}
