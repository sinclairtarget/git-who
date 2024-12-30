/*
* Utility functions for formatting output.
 */
package format

import "fmt"

// Print string with max length, truncating with ellipsis.
func Abbrev(s string, max int) string {
	if len(s) <= max {
		return s
	}

	return s[:max-1] + "…"
}

func GitEmail(email string) string {
	return fmt.Sprintf("<%s>", email)
}
