package tally

import (
	"slices"
	"testing"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/utils/iterutils"
)

func TestTimeSeriesCombine(t *testing.T) {
	a := TimeSeries{
		TimeBucket{
			Name: "2024-04-01",
			Time: time.Date(2024, 4, 1, 0, 0, 0, 0, time.Local),
			tallies: map[string]Tally{
				"alice": {added: 3},
				"bob":   {added: 2},
			},
		},
		TimeBucket{
			Name: "2024-04-02",
			Time: time.Date(2024, 4, 2, 0, 0, 0, 0, time.Local),
			tallies: map[string]Tally{
				"bob": {added: 1},
			},
		},
		TimeBucket{
			Name: "2024-04-03",
			Time: time.Date(2024, 4, 3, 0, 0, 0, 0, time.Local),
			tallies: map[string]Tally{
				"bob":  {added: 4},
				"john": {added: 7},
			},
		},
	}

	b := TimeSeries{
		TimeBucket{
			Name: "2024-04-02",
			Time: time.Date(2024, 4, 2, 0, 0, 0, 0, time.Local),
			tallies: map[string]Tally{
				"alice": {added: 1},
			},
		},
		TimeBucket{
			Name: "2024-04-03",
			Time: time.Date(2024, 4, 3, 0, 0, 0, 0, time.Local),
			tallies: map[string]Tally{
				"bob": {added: 2},
			},
		},
		TimeBucket{
			Name: "2024-04-04",
			Time: time.Date(2024, 4, 4, 0, 0, 0, 0, time.Local),
			tallies: map[string]Tally{
				"alice": {added: 9},
			},
		},
	}

	c := a.Combine(b)
	expected := TimeSeries{
		TimeBucket{
			Name: "2024-04-01",
			Time: time.Date(2024, 4, 1, 0, 0, 0, 0, time.Local),
			tallies: map[string]Tally{
				"alice": {added: 3},
				"bob":   {added: 2},
			},
		},
		TimeBucket{
			Name: "2024-04-02",
			Time: time.Date(2024, 4, 2, 0, 0, 0, 0, time.Local),
			tallies: map[string]Tally{
				"alice": {added: 1},
				"bob":   {added: 1},
			},
		},
		TimeBucket{
			Name: "2024-04-03",
			Time: time.Date(2024, 4, 3, 0, 0, 0, 0, time.Local),
			tallies: map[string]Tally{
				"bob":  {added: 6},
				"john": {added: 7},
			},
		},
		TimeBucket{
			Name: "2024-04-04",
			Time: time.Date(2024, 4, 4, 0, 0, 0, 0, time.Local),
			tallies: map[string]Tally{
				"alice": {added: 9},
			},
		},
	}

	if c[0].Name != expected[0].Name {
		t.Errorf("first bucket date is wrong")
	}
	if c[0].tallies["alice"].added != expected[0].tallies["alice"].added {
		t.Errorf("alice tally for first bucket is wrong")
	}
	if c[0].tallies["bob"].added != expected[0].tallies["bob"].added {
		t.Errorf("bob tally for first bucket is wrong")
	}

	if c[1].Name != expected[1].Name {
		t.Errorf("second bucket date is wrong")
	}
	if c[1].tallies["alice"].added != expected[1].tallies["alice"].added {
		t.Errorf("alice tally for second bucket is wrong")
	}
	if c[1].tallies["bob"].added != expected[1].tallies["bob"].added {
		t.Errorf("bob tally for second bucket is wrong")
	}

	if c[2].Name != expected[2].Name {
		t.Errorf("third bucket date is wrong")
	}
	if c[2].tallies["alice"].added != expected[2].tallies["alice"].added {
		t.Errorf("alice tally for third bucket is wrong")
	}
	if c[2].tallies["bob"].added != expected[2].tallies["bob"].added {
		t.Errorf("bob tally for third bucket is wrong")
	}
	if c[2].tallies["john"].added != expected[2].tallies["john"].added {
		t.Errorf("john tally for third bucket is wrong")
	}

	if c[3].Name != expected[3].Name {
		t.Errorf("fourth bucket date is wrong")
	}
	if c[3].tallies["alice"].added != expected[3].tallies["alice"].added {
		t.Errorf("alice tally for fourth bucket is wrong")
	}
}

func TestTallyCommitsTimelineEmpty(t *testing.T) {
	seq := iterutils.WithoutErrors(slices.Values([]git.Commit{}))
	opts := TallyOpts{
		Mode: CommitMode,
		Key:  func(c git.Commit) string { return c.AuthorEmail },
	}
	end := time.Now()

	buckets, err := TallyCommitsTimeline(seq, opts, end)
	if err != nil {
		t.Errorf("TallyCommitsTimeline() returned error: %v", err)
	}

	if len(buckets) > 0 {
		t.Errorf(
			"TallyCommitsTimeline() should have returned empty slice but returned %v",
			buckets,
		)
	}
}
