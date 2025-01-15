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

func TestNumber(t *testing.T) {
	tests := []struct {
		name string
		n    int
		exp  string
	}{
		{
			name: "zero",
			n:    0,
			exp:  "0",
		},
		{
			name: "hundereds",
			n:    123,
			exp:  "123",
		},
		{
			name: "thousand_and_one",
			n:    1001,
			exp:  "1,001",
		},
		{
			name: "low_thousands",
			n:    1234,
			exp:  "1,234",
		},
		{
			name: "high_thousands",
			n:    957123,
			exp:  "957,123",
		},
		{
			name: "millions",
			n:    1_234_567,
			exp:  "1.2m",
		},
		{
			name: "ten_millions",
			n:    12_345_678,
			exp:  "12.3m",
		},
		{
			name: "hundred_millions",
			n:    123_456_789,
			exp:  ">99m",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ans := format.Number(test.n)
			if ans != test.exp {
				t.Errorf("expected %s but got %s", test.exp, ans)
			}
		})
	}
}

func TestNumberNegativeError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Number() did not panic with negative input")
		}
	}()

	format.Number(-1)
}
