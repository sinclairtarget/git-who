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
	RepoMailmapPath   string
	GlobalMailmapPath string
	IgnoreRevsPath    string
}

func (sf SupplementalFiles) HasMailmap() bool {
	return len(sf.RepoMailmapPath) > 0 || len(sf.GlobalMailmapPath) > 0
}

func (sf SupplementalFiles) HasIgnoreRevs() bool {
	return len(sf.IgnoreRevsPath) > 0
}

func (sf SupplementalFiles) MailmapHash(h hash.Hash32) error {
	if len(sf.RepoMailmapPath) > 0 {
		f, err := os.Open(sf.RepoMailmapPath)
		if !errors.Is(err, fs.ErrNotExist) {
			if err != nil {
				return fmt.Errorf("could not read repo mailmap file: %v", err)
			}
			defer f.Close()

			_, err = io.Copy(h, f)
			if err != nil {
				return fmt.Errorf("error hashing repo mailmap file: %v", err)
			}
		}
	}

	if len(sf.GlobalMailmapPath) > 0 {
		f, err := os.Open(sf.GlobalMailmapPath)
		if !errors.Is(err, fs.ErrNotExist) {
			if err != nil {
				return fmt.Errorf("could not read global mailmap file: %v", err)
			}
			defer f.Close()

			_, err = io.Copy(h, f)
			if err != nil {
				return fmt.Errorf("error hashing global mailmap file: %v", err)
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

		// Comments starting with "#" are allowed in the ignore revs file
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
