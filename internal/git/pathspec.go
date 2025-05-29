package git

import (
	"path"
	"regexp"

	"github.com/bmatcuk/doublestar/v4"
)

var excludePathspecRegexp *regexp.Regexp
var excludeStripRegexp *regexp.Regexp

func init() {
	excludePathspecRegexp = regexp.MustCompile(
		`^(:[!\^]:|:[!\^][^!\^/]|:\(exclude\))`,
	)
	excludeStripRegexp = regexp.MustCompile(
		`^(:[!\^]:?|:\(exclude\))`,
	)
}

/*
* We only support the "exclude" pathspec magic.
 */
func IsSupportedPathspec(pathspec string) bool {
	if len(pathspec) > 0 && pathspec[0] == ':' {
		return excludePathspecRegexp.MatchString(pathspec)
	}

	return true
}

/*
* Splits the include pathspecs from the exclude pathspecs.
*
* For the exclude pathspecs, we also strip off the leading "magic".
 */
func SplitPathspecs(pathspecs []string) (includes []string, excludes []string) {
	for _, p := range pathspecs {
		if len(p) == 0 {
			continue // skip this degenerate case, Git disallows it
		}

		if p[0] == ':' {
			// Strip magic
			stripped := excludeStripRegexp.ReplaceAllString(p, "")
			excludes = append(excludes, stripped)
		} else {
			includes = append(includes, p)
		}
	}

	return includes, excludes
}

func PathspecMatch(pathspec string, p string) bool {
	if len(pathspec) == 0 {
		panic("empty string is not valid pathspec")
	}

	// Note: Git uses fnmatch(). This match may differ. Hopefully only rarely.
	didMatch, err := doublestar.PathMatch(pathspec, p)
	if err != nil {
		panic("bad pattern passed to doublestar.Match()")
	}

	if didMatch {
		return true
	}

	// Ensure we mimic Git behavior with trailing slash. See "pathspec" in
	// gitglossary(3).
	subdirPathspec := path.Join(pathspec, "**")
	didMatch, err = doublestar.PathMatch(subdirPathspec, p)
	if err != nil {
		panic("bad pattern passed to doublestar.Match()")
	}

	if didMatch {
		return true
	}

	if pathspec[0] == '*' {
		toplevelPathspec := path.Join("**/", pathspec)
		didMatch, err = doublestar.PathMatch(toplevelPathspec, p)
		if err != nil {
			panic("bad pattern passed to doublestar.Match()")
		}

		if didMatch {
			return true
		}
	}

	return false
}
