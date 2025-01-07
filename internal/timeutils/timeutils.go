package timeutils

import "time"

func Max(a, b time.Time) time.Time {
	if b.Before(a) {
		return a
	} else {
		return b
	}
}
