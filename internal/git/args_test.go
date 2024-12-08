package git_test

import (
    "slices"
    "testing"

    "github.com/sinclairtarget/git-who/internal/git"
)

func TestParseArgs(t *testing.T) {
    tests := []struct{
        name    string
        args    []string
        expRevs []string
        expPath string
    } {
        {
            "empty_args",
            []string {},
            []string {},
            ".",
        },
        {
            "nil_args",
            nil,
            []string {},
            ".",
        },
        {
            "no_separator",
            []string { "foo", "bar" },
            []string { "foo", "bar" },
            ".",
        },
        {
            "separator",
            []string { "foo", "--", "bar" },
            []string { "foo" },
            "bar",
        },
        {
            "trailing_separator",
            []string { "foo", "--" },
            []string { "foo" },
            ".",
        },
        {
            "duplicate_separator", // Should ignore extra args
            []string { "foo", "--", "bar", "--" },
            []string { "foo" },
            "bar",
        },
        {
            "duplicate_trailing_separator",
            []string { "foo", "--", "--" },
            []string { "foo" },
            ".",
        },
    }

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            revs, path := git.ParseArgs(test.args)
            if !slices.Equal(revs, test.expRevs) {
                t.Errorf(
                    "expected %v as revs but got %v",
                    test.expRevs,
                    revs,
                )
            }

            if path != test.expPath {
                t.Errorf(
                    "expected \"%s\" as path but got \"%s\"",
                    test.expPath,
                    path,
                )
            }
        })
    }
}
