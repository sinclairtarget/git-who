// Utils for running tests in the test submodule/repo.
package repotest

import (
	"fmt"
	"os"
)

const msg = `error changing working directory to submodule: %w
Did you remember to initialize the submodule? See README.md`

func UseTestRepo() error {
	err := os.Chdir("../../test-repo")
	if err != nil {
		return fmt.Errorf(msg, err)
	}

	return nil
}
