package main

import (
	"fmt"
)

// The "table" subcommand summarizes the authorship history of the given
// commits and path in a table printed to stdout.
func table(revs []string, path string, useCsv bool) error {
	fmt.Printf("table() revs: %v, path: %s, useCsv: %t\n", revs, path, useCsv)
	return nil
}
