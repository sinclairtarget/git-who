package pretty

import (
	"os"

	"golang.org/x/term"
)

// Allow backspacing and replacing output for e.g. progress indicator?
func AllowDynamic(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
