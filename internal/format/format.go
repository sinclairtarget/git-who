/*
* Utility functions for formatting output.
 */
package format

import (
	"fmt"
	"time"
	"unicode/utf8"

	runewidth "github.com/mattn/go-runewidth"
)

// Print string with max length, truncating with ellipsis.
func Abbrev(s string, max int) string {
	tail := "…"

	if len(s) > utf8.RuneCountInString(s) {
		return runewidth.Truncate(s, max, tail)
	}

	if len(s) <= max {
		return s
	}

	return s[:max-1] + tail
}

func GitEmail(email string) string {
	return fmt.Sprintf("<%s>", email)
}

func RelativeTime(now time.Time, t time.Time) string {
	duration := now.Sub(t)

	day := time.Hour * 24
	week := day * 7
	month := day * 30 // eh
	year := day * 365

	if duration < time.Hour {
		minutes := int(duration / time.Minute)
		return fmt.Sprintf("%d min. ago", minutes)
	} else if duration < day {
		hours := int(duration / time.Hour)
		if hours > 1 {
			return fmt.Sprintf("%d hr. ago", hours)
		} else {
			return fmt.Sprintf("%d hour ago", hours)
		}
	} else if duration < week {
		days := int(duration / day)
		if days > 1 {
			return fmt.Sprintf("%d days ago", days)
		} else {
			return fmt.Sprintf("%d day ago", days)
		}
	} else if duration < month {
		weeks := int(duration / week)
		if weeks > 1 {
			return fmt.Sprintf("%d weeks ago", weeks)
		} else {
			return fmt.Sprintf("%d week ago", weeks)
		}
	} else if duration < year {
		months := int(duration / month)
		if months > 1 {
			return fmt.Sprintf("%d mon. ago", months)
		} else {
			return fmt.Sprintf("%d month ago", months)
		}
	} else {
		years := int(duration / year)
		if years > 99 {
			return ">99 yr. ago"
		} else if years > 1 {
			return fmt.Sprintf("%d yr. ago", years)
		} else {
			return fmt.Sprintf("%d year ago", years)
		}
	}
}

// Adds thousands comma and abbreviates numbers > 1m
func Number(num int) string {
	if num < 0 {
		panic("cannot format negative number")
	}

	if num > 100_000_000 {
		return ">99m"
	}

	if num > 1_000_000 {
		mils := float32(num) / 1_000_000
		return fmt.Sprintf("%.1fm", mils)
	}

	if num > 1_000 {
		ones := num % 1_000
		thousands := num / 1_000
		return fmt.Sprintf("%d,%03d", thousands, ones)
	}

	return fmt.Sprintf("%d", num)
}
