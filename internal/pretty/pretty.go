package pretty

import (
	"os"

	"golang.org/x/term"
)

func AllowDynamic(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
