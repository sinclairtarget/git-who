package git

import "slices"

// Handles splitting the Git revisions from the path in a list of args
func ParseArgs(args []string) (revs []string, path string) {
    // TODO: For now this follows a simple rule:
    //
    //   All args are treated as revisions EXCEPT for the arg following "--" if
    //   one exists.
    //
    // In the future, we want to allow the path to be specified without "--",
    // but that will require handling ambiguous cases with tests to see if an
    // arg is a revision.

    path = "."

    // --- Find index of separator ---
    sepIndex := slices.Index(args, "--")
    if sepIndex < 0 {
        sepIndex = len(args)
    }
    
    // --- If following arg exists and could be a path, use that ---
    if sepIndex + 1 < len(args) && args[sepIndex + 1] != "--" {
        path = args[sepIndex + 1]
    }

    return args[:sepIndex], path
}
