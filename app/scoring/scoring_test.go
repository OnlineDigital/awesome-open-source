package scoring

import (
	"fmt"
	"testing"
	"time"
)

var now = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)

func ago(d int) time.Time { return now.AddDate(0, 0, -d) }

func releases(n, everyDays, startAgo int, patch bool) []Release {
	var rs []Release
	for i := 0; i < n; i++ {
		tag := fmt.Sprintf("v1.%d.0", n-i)
		if patch {
			tag = fmt.Sprintf("v1.0.%d", n-i)
		}
		rs = append(rs, Release{Tag: tag, PublishedAt: ago(startAgo + i*everyDays)})
	}
	return rs
}

func TestScenarios(t *testing.T) {
	cases := []struct {
		name     string
		s        Signals
		min, max float64
		tier     int
	}{
		{"thriving: daily commits, monthly releases", Signals{Now: now, LastCommitAt: ago(1), PushedAt: ago(1), Commits30: 80, Commits90: 240, Commits365: 900,
			SampledCommits: 100, FixCommits: 25, Releases: releases(14, 25, 3, false)}, 4.5, 5, 5},
		{"active: regular commits, a few releases", Signals{Now: now, LastCommitAt: ago(20), Commits90: 8, Commits365: 40,
			SampledCommits: 40, FixCommits: 10, Releases: releases(3, 90, 100, false)}, 3.5, 4.4, 4},
		{"maintained: slow but alive", Signals{Now: now, LastCommitAt: ago(70), Commits90: 3, Commits365: 22,
			SampledCommits: 22, FixCommits: 4, Releases: releases(2, 150, 80, false)}, 2.5, 3.4, 3},
		{"slow: nothing for five months", Signals{Now: now, LastCommitAt: ago(150), Commits365: 15,
			SampledCommits: 15, FixCommits: 1, Releases: releases(1, 0, 200, false)}, 1.7, 2.4, 2},
		{"stalled: silent for 14 months", Signals{Now: now, LastCommitAt: ago(430), Commits365: 0, Releases: releases(1, 0, 600, false)}, 1, 1.6, 1},
		{"abandoned: silent for 3 years", Signals{Now: now, LastCommitAt: ago(1100)}, 1, 1.0, 1},
		{"archived overrides everything", Signals{Now: now, Archived: true, LastCommitAt: ago(1), Commits90: 100, Commits365: 400, Releases: releases(10, 20, 1, false)}, 1, 1, 1},
		{"no releases but very busy", Signals{Now: now, LastCommitAt: ago(0), Commits30: 60, Commits90: 180, Commits365: 700, SampledCommits: 100, FixCommits: 30}, 4.5, 4.9, 5},
		{"any-branch push counts as recency", Signals{Now: now, LastCommitAt: ago(300), PushedAt: ago(2), Commits90: 12, Commits365: 60}, 2.5, 5, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Compute(c.s)
			t.Logf("score=%.1f tier=%d raw=%.3f cap=%q", r.Score, r.Tier, r.Raw, r.Cap)
			if r.Score < c.min || r.Score > c.max {
				t.Fatalf("score %.1f outside [%.1f, %.1f]", r.Score, c.min, c.max)
			}
			if c.tier != 0 && r.Tier != c.tier {
				t.Fatalf("tier %d, want %d", r.Tier, c.tier)
			}
		})
	}
}

func TestMonotonicInRecency(t *testing.T) {
	prev := 6.0
	for _, d := range []int{0, 10, 30, 90, 180, 360, 500, 900} {
		r := Compute(Signals{Now: now, LastCommitAt: ago(d), Commits90: 10, Commits365: 50, Releases: releases(4, 60, d, false)})
		if r.Score > prev {
			t.Fatalf("score increased as the project got older: %d days -> %.1f (prev %.1f)", d, r.Score, prev)
		}
		prev = r.Score
	}
}

func TestBoundsAndWeights(t *testing.T) {
	for _, s := range []Signals{{Now: now}, {Now: now, LastCommitAt: now, Commits30: 1e6, Commits90: 1e6, Commits365: 1e6, ActiveBranches90: 50, Releases: releases(100, 1, 0, true)}} {
		r := Compute(s)
		if r.Score < 1 || r.Score > 5 {
			t.Fatalf("score out of range: %v", r.Score)
		}
		sum := 0.0
		for _, c := range r.Components {
			sum += c.Weight
		}
		if sum < 0.998 || sum > 1.002 {
			t.Fatalf("weights sum to %v", sum)
		}
	}
}

func TestFixMessage(t *testing.T) {
	for msg, want := range map[string]bool{"fix: crash on start": true, "Fixes #12": true, "bump deps": false, "feat: add search": false, "Hotfix release": true} {
		if IsFixMessage(msg) != want {
			t.Errorf("IsFixMessage(%q) != %v", msg, want)
		}
	}
}
