package cache_test

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"testing"

	"github.com/sinclairtarget/git-who/internal/cache"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/repotest"
)

func TestMailmapHash(t *testing.T) {
	tests := map[string]struct {
		repoMailmapFilePath      string
		repoMailmapFileContent   string
		configMailmapFilePath    string
		configMailmapFileContent string
		configMailmapBlobPath    string
		configMailmapBlobContent string
		expectedHash             uint32
		expectedError            error
	}{
		"none mailmap": {
			expectedHash: fnv.New32().Sum32(),
		},
		"miss mailmap": {
			repoMailmapFilePath:   ".mailmap",
			configMailmapFilePath: ".contacts",
			configMailmapBlobPath: ".rolodex",
			expectedHash:          fnv.New32().Sum32(),
		},
		"repo mailmap": {
			repoMailmapFilePath:    ".mailmap",
			repoMailmapFileContent: "Alice <alice@mail.com>",
			expectedHash:           1682318899,
		},
		"file mailmap": {
			configMailmapFilePath:    ".contacts",
			configMailmapFileContent: "Bob <bob@mail.com>",
			expectedHash:             3288519805,
		},
		"blob mailmap": {
			configMailmapBlobPath:    ".rolodex",
			configMailmapBlobContent: "Chris <chris@mail.com>",
			expectedHash:             2428936825,
		},
		"join mailmap": {
			repoMailmapFilePath:      ".mailmap",
			repoMailmapFileContent:   "Alice <alice@mail.com>",
			configMailmapFilePath:    ".contacts",
			configMailmapFileContent: "Bob <bob@mail.com>",
			configMailmapBlobPath:    ".rolodex",
			configMailmapBlobContent: "Chris <chris@mail.com>",
			expectedHash:             419655903,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			config, err := repotest.GitConfigPath()
			if err != nil {
				t.Fatalf("failed to get config path: %v", err)
			}
			original, err := os.ReadFile(config)
			if err != nil {
				t.Fatalf("failed to read config file: %v", err)
			}
			defer func() {
				err := os.WriteFile(config, original, 0o644)
				if err != nil {
					t.Fatalf("failed to restore .git/config: %v", err)
				}
			}()
			if test.repoMailmapFileContent != "" && test.repoMailmapFilePath != "" {
				err := os.WriteFile(
					test.repoMailmapFilePath,
					[]byte(test.repoMailmapFileContent),
					0o644,
				)
				if err != nil {
					t.Fatalf("failed to write %s: %v", test.repoMailmapFilePath, err)
				}
				defer func() {
					err := os.Remove(test.repoMailmapFilePath)
					if err != nil {
						t.Fatalf("failed to remove %s: %v", test.repoMailmapFilePath, err)
					}
				}()
			}
			if test.configMailmapFileContent != "" && test.configMailmapFilePath != "" {
				err := os.WriteFile(
					test.configMailmapFilePath,
					[]byte(test.configMailmapFileContent),
					0o644,
				)
				if err != nil {
					t.Fatalf("failed to write %s: %v", test.configMailmapFilePath, err)
				}
				defer func() {
					err := os.Remove(test.configMailmapFilePath)
					if err != nil {
						t.Fatalf("failed to remove %s: %v", test.configMailmapFilePath, err)
					}
				}()
			}
			if test.configMailmapFilePath != "" {
				args := []string{
					"config",
					"--local",
					"mailmap.file",
					test.configMailmapFilePath,
				}
				cmd := exec.Command("git", args...)
				err := cmd.Run()
				if err != nil {
					t.Fatalf("failed to set git %v: %v", args, err)
				}
			}
			if test.configMailmapBlobContent != "" && test.configMailmapBlobPath != "" {
				err = os.WriteFile(
					test.configMailmapBlobPath,
					[]byte(test.configMailmapBlobContent),
					0o644,
				)
				if err != nil {
					t.Fatalf("failed to write %s: %v", test.configMailmapBlobPath, err)
				}
				defer func() {
					err := os.Remove(test.configMailmapBlobPath)
					if err != nil {
						t.Fatalf("failed to remove %s: %v", test.configMailmapBlobPath, err)
					}
				}()
			}
			if test.configMailmapBlobPath != "" {
				args := []string{
					"config",
					"--local",
					"mailmap.blob",
					fmt.Sprintf(":0:%s", test.configMailmapBlobPath),
				}
				cmd := exec.Command("git", args...)
				err := cmd.Run()
				if err != nil {
					t.Fatalf("failed to set git %v: %v", args, err)
				}
			}
			if test.configMailmapBlobContent != "" && test.configMailmapBlobPath != "" {
				args := []string{
					"add",
					test.configMailmapBlobPath,
				}
				cmd := exec.Command("git", args...)
				err := cmd.Run()
				if err != nil {
					t.Fatalf("failed to stage %v: %v", test.configMailmapBlobPath, err)
				}
				defer func() {
					args := []string{
						"restore",
						"--staged",
						test.configMailmapBlobPath,
					}
					cmd := exec.Command("git", args...)
					err := cmd.Run()
					if err != nil {
						t.Fatalf("failed to unstage %v: %v", test.configMailmapBlobPath, err)
					}
				}()
			}
			ctx := context.Background()
			h := fnv.New32()
			rf := git.RepoConfigFiles{
				MailmapPath: test.repoMailmapFilePath,
			}
			err = cache.MailmapHash(ctx, h, rf)
			if err != nil {
				t.Errorf("got error: %v", err)
			}
			if h.Sum32() != test.expectedHash {
				t.Errorf(
					"expected %v as hash but got %v",
					test.expectedHash,
					h.Sum32(),
				)
			}
		})
	}
}
