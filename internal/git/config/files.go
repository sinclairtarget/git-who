package config

import (
	"bufio"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"strings"

	rev "github.com/sinclairtarget/git-who/internal/git/revision"
)

// Not .gitconfig files, but still configure Git behavior
type SupplementalFiles struct {
	MailmapPath    string
	IgnoreRevsPath string
}

func (sf SupplementalFiles) HasMailmap() bool {
	return len(sf.MailmapPath) > 0
}

func (sf SupplementalFiles) HasIgnoreRevs() bool {
	return len(sf.IgnoreRevsPath) > 0
}

func (sf SupplementalFiles) MailmapHash(h hash.Hash32) error {
	if sf.HasMailmap() {
		f, err := os.Open(sf.MailmapPath)
		if !errors.Is(err, fs.ErrNotExist) {
			if err != nil {
				return fmt.Errorf("could not read mailmap file: %v", err)
			}
			defer f.Close()

			_, err = io.Copy(h, f)
			if err != nil {
				return fmt.Errorf("error hashing mailmap file: %v", err)
			}
		}
	}

	return nil
}

// Get git blame ignored revisions
func (sf SupplementalFiles) IgnoreRevs() (_ []string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error reading git blame ignore revs: %w", err)
		}
	}()

	var revs []string

	if !sf.HasIgnoreRevs() {
		return revs, nil
	}

	f, err := os.Open(sf.IgnoreRevsPath)
	if err != nil {
		return revs, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if rev.IsFullHash(line) {
			revs = append(revs, line)
		}
	}

	err = scanner.Err()
	if err != nil {
		return revs, err
	}

	return revs, nil
}
