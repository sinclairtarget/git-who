// Helpers for running tests in the test submodule/repo.
package repotest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// GitConfigPath returns the git config file path for testing.
func GitConfigPath() (string, error) {
	info, err := os.Stat(".git")
	if err != nil {
		return "", fmt.Errorf("failed to stat .git: %w", err)
	}
	if info.IsDir() {
		return filepath.Abs(filepath.Join(".git", "config"))
	}
	submodule, err := os.ReadFile(".git")
	if err != nil {
		return "", fmt.Errorf("cannot read .git file: %w", err)
	}
	text := strings.TrimSpace(string(submodule))
	prefix := "gitdir: "
	if !strings.HasPrefix(text, prefix) {
		return "", fmt.Errorf("failed to find submodule config")
	}
	gitdir := strings.TrimSpace(strings.TrimPrefix(text, prefix))
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(filepath.Dir(".git"), gitdir)
	}
	return filepath.Join(gitdir, "config"), nil
}
