package git_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/sinclairtarget/git-who/internal/repotest"
)

// Run these tests in the test submodule.
func TestMain(m *testing.M) {
	err := repotest.UseTestRepo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	m.Run()
}
