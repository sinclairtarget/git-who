package cmd

import (
	"fmt"
	"strings"
)

type LogFilters struct {
	Since    string
	Until    string
	Authors  []string
	Nauthors []string
}

// Turn into CLI args we can pass to `git log`
func (f LogFilters) ToArgs() []string {
	args := []string{}

	if f.Since != "" {
		args = append(args, "--since", f.Since)
	}

	if f.Until != "" {
		args = append(args, "--until", f.Until)
	}

	for _, author := range f.Authors {
		args = append(args, "--author", author)
	}

	if len(f.Nauthors) > 0 {
		args = append(args, "--perl-regexp")

		// Build regex pattern OR-ing together all the nauthors
		var b strings.Builder
		for i, nauthor := range f.Nauthors {
			b.WriteString(nauthor)
			if i < len(f.Nauthors)-1 {
				b.WriteString("|")
			}
		}

		regex := fmt.Sprintf(`^((?!%s).*)$`, b.String())
		args = append(args, "--author", regex)
	}

	return args
}
