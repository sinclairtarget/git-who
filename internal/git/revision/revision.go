package revision

import (
	"regexp"
)

var commitHashRegexp *regexp.Regexp

func init() {
	commitHashRegexp = regexp.MustCompile(`^\^?[a-f0-9]+$`)
}

// Returns true if this is a (full-length) Git revision hash, false otherwise.
//
// We also need to handle a hash with "^" in front.
func IsFullHash(s string) bool {
	matched := commitHashRegexp.MatchString(s)
	return matched && (len(s) == 40 || len(s) == 41)
}
