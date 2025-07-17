// Helpers for running tests in the test submodule/repo.
package repotest

import (
	"os"
	"testing"
)

const msg = `error changing working directory to submodule: %v
Did you remember to initialize the submodule? See README.md`

func UseTestRepo(t *testing.T) {
	err := os.Chdir("../../repos/test-repo")
	if err != nil {
		t.Fatalf(msg, err)
	}
}
