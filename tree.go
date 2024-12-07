package main

import (
    "fmt"
)

func tree(revs []string, path string, mode tallyMode, depth int) error {
    fmt.Printf("tree() revs: %v, path: %s, mode: %v, depth: %d\n",
               revs,
               path,
               mode,
               depth);
    return nil
}
