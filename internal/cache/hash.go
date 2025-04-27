package cache

import (
	"context"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/sinclairtarget/git-who/internal/git"
)

// MailmapHash adds content from configured mailmaps to a repository hash value.
func MailmapHash(
	ctx context.Context,
	h hash.Hash32,
	rf git.RepoConfigFiles,
) error {
	if rf.HasMailmap() {
		f, err := os.Open(rf.MailmapPath)
		if !errors.Is(err, fs.ErrNotExist) {
			if err != nil {
				return fmt.Errorf("could not read repo mailmap file: %w", err)
			}
			defer f.Close()
			_, err := io.Copy(h, f)
			if err != nil {
				return fmt.Errorf("error hashing repo mailmap file: %w", err)
			}
		}
	}

	configMailmapFile := ""
	configMailmapFileResponse, err := git.RunConfig(ctx, "mailmap.file")
	if err != nil {
		return fmt.Errorf("error running config mailmap file command: %w", err)
	}
	out, err := configMailmapFileResponse.Stdout()
	if err != nil {
		return fmt.Errorf("error parsing config mailmap file command: %w", err)
	} else {
		configMailmapFile = out.Stdout
	}
	if configMailmapFile != "" {
		if strings.HasPrefix(configMailmapFile, "~") {
			usr, err := user.Current()
			if err != nil {
				return fmt.Errorf("could not make mailmap file path: %w", err)
			}
			configMailmapFile = filepath.Join(usr.HomeDir, configMailmapFile[1:])
		}
		f, err := os.Open(configMailmapFile)
		if !errors.Is(err, os.ErrNotExist) {
			if err != nil {
				return fmt.Errorf("could not read config mailmap file: %w", err)
			}
			defer f.Close()
			_, err = io.Copy(h, f)
			if err != nil {
				return fmt.Errorf("error hashing config mailmap file: %w", err)
			}
		}
	}

	configMailmapBlob := ""
	configMailmapBlobResponse, err := git.RunConfig(ctx, "mailmap.blob")
	if err != nil {
		return fmt.Errorf("error running config mailmap blob command: %w", err)
	}
	out, err = configMailmapBlobResponse.Stdout()
	if err != nil {
		return fmt.Errorf("error parsing config mailmap blob command: %w", err)
	} else {
		configMailmapBlob = out.Stdout
	}
	if strings.Contains(configMailmapBlob, ":") {
		blobs := []string{
			configMailmapBlob,
		}
		revision, err := git.RunRevParse(ctx, blobs)
		if err != nil {
			return fmt.Errorf("error running revision parse command: %w", err)
		}
		out, err := revision.Stdout()
		if err != nil {
			return fmt.Errorf("could not resolve config mailmap blob: %w", err)
		}
		if out.ExitCode != 0 {
			configMailmapBlob = ""
		} else {
			configMailmapBlob = out.Stdout
		}
	}
	if configMailmapBlob != "" {
		response, err := git.RunCatFile(ctx, configMailmapBlob)
		if err != nil {
			return fmt.Errorf("could not read config mailmap blob: %w", err)
		}
		out, err := response.Stdout()
		if err != nil {
			return fmt.Errorf("error parsing config mailmap blob: %w", err)
		}
		contents := out.Stdout
		if _, err := h.Write([]byte(contents)); err != nil {
			return fmt.Errorf("error hashing config mailmap blob: %w", err)
		}
	}

	return nil
}
