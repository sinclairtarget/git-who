package format_test

import (
	"testing"
	"time"

	"github.com/sinclairtarget/git-who/internal/format"
)

func TestRelativeTime(t *testing.T) {
	now, err := time.Parse(time.DateTime, "2024-12-30 10:13:00")
	if err != nil {
		t.Fatal("could not parse timestamp")
	}

	then, err := time.Parse(time.DateTime, "2023-10-16 17:16:05")
	if err != nil {
		t.Fatal("could not parse timestamp")
	}

	description := format.RelativeTime(now, then)
	if description != "1 year ago" {
		t.Fatalf("expected \"%s\", but got: \"%s\"", "1 year ago", description)
	}
}
