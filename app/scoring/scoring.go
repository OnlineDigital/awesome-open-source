// Package scoring turns raw GitHub activity signals into one maintenance
// score between 1.0 (abandoned) and 5.0 (thriving).
//
// The score is deterministic and explainable: every component is a number in
// [0,1], the weights are fixed, and the full breakdown is stored with the
// project so the UI can show why a project got its rating.
//
//	recency   (40%)  how long ago ANY branch received a commit; half-life 90 days
//	commits   (25%)  commit volume over the last 90 and 365 days (log-saturating)
//	releases  (25%)  age of the last release + number of releases in 12 months
//	fixes     (10%)  share of recent commits that are bug fixes + patch releases
//
// Components without data (no releases, no commit messages) are dropped and
// their weight is redistributed; a project without any release gets a 7%
// discount so it can't outrank equally busy projects that ship releases. Hard rules then cap the result: archived
// repositories are always 1.0 and projects silent for >1y / >2y are capped
// at 2.4 / 1.4.
package scoring

import (
	"math"
	"regexp"
	"sort"
	"time"
)

type Release struct {
	Tag         string    `json:"tag"`
	PublishedAt time.Time `json:"publishedAt"`
	Prerelease  bool      `json:"prerelease"`
}

// Signals are the raw inputs gathered from GitHub.
type Signals struct {
	Now      time.Time `json:"-"`
	Archived bool      `json:"archived"`
	Disabled bool      `json:"disabled"`

	CreatedAt    time.Time `json:"createdAt"`
	PushedAt     time.Time `json:"pushedAt"`
	LastCommitAt time.Time `json:"lastCommitAt"` // newest commit across all branches

	Commits30  int `json:"commits30"` // default branch
	Commits90  int `json:"commits90"`
	Commits365 int `json:"commits365"`

	ActiveBranches90 int `json:"activeBranches90"` // branches with a commit in the last 90 days
	TotalBranches    int `json:"totalBranches"`

	// RecentCommitMessages is a sample (newest first) used to estimate bug-fix activity.
	RecentCommitMessages []string `json:"-"`
	SampledCommits       int      `json:"sampledCommits"`
	FixCommits           int      `json:"fixCommits"`

	Releases []Release `json:"releases"`
}

type Component struct {
	Name     string  `json:"name"`
	Value    float64 `json:"value"`  // 0..1, meaningful only when Used
	Weight   float64 `json:"weight"` // effective weight after redistribution
	Used     bool    `json:"used"`
	Evidence string  `json:"evidence,omitempty"`
}

type Result struct {
	Score      float64     `json:"score"` // 1.0 .. 5.0, one decimal
	Tier       int         `json:"tier"`  // 1..5
	Label      string      `json:"label"` // thriving|active|maintained|slow|abandoned
	Raw        float64     `json:"raw"`   // 0..1 before caps
	Cap        string      `json:"cap,omitempty"`
	Components []Component `json:"components"`
	Version    int         `json:"version"`
}

const Version = 1

var Labels = map[int]string{5: "thriving", 4: "active", 3: "maintained", 2: "slow", 1: "abandoned"}

var fixPattern = regexp.MustCompile(`(?i)\b(fix(es|ed)?|bug(fix)?|hotfix|patch|regression|crash|security|cve-\d+)\b`)

// IsFixMessage reports whether a commit headline looks like a bug fix.
func IsFixMessage(msg string) bool { return fixPattern.MatchString(msg) }

func days(from, to time.Time) float64 {
	if from.IsZero() {
		return math.Inf(1)
	}
	d := to.Sub(from).Hours() / 24
	if d < 0 {
		return 0
	}
	return d
}

func halfLife(d, half float64) float64 {
	if math.IsInf(d, 1) {
		return 0
	}
	return math.Pow(2, -d/half)
}

// sat maps a count onto 0..1 logarithmically; sat(k,k)=1.
func sat(x, k float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Min(1, math.Log1p(x)/math.Log1p(k))
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }

// Compute calculates the activity score. Signals.Now defaults to time.Now().
func Compute(s Signals) Result {
	now := s.Now
	if now.IsZero() {
		now = time.Now()
	}
	last := s.LastCommitAt
	if s.PushedAt.After(last) {
		last = s.PushedAt
	}
	sinceCommit := days(last, now)

	var comps []Component

	// 1. Recency
	comps = append(comps, Component{
		Name: "recency", Value: halfLife(sinceCommit, 90), Weight: 0.40, Used: true,
		Evidence: evidenceDays("last commit", sinceCommit),
	})

	// 2. Commit volume (default branch) plus a small bonus for parallel branch work.
	freq := 0.6*sat(float64(s.Commits90), 30) + 0.4*sat(float64(s.Commits365), 120)
	if s.ActiveBranches90 > 1 {
		freq = math.Min(1, freq+0.05*math.Min(4, float64(s.ActiveBranches90-1)))
	}
	comps = append(comps, Component{Name: "commits", Value: freq, Weight: 0.25, Used: true})

	// 3. Releases
	var published []Release
	for _, r := range s.Releases {
		if !r.PublishedAt.IsZero() {
			published = append(published, r)
		}
	}
	sort.Slice(published, func(i, j int) bool { return published[i].PublishedAt.After(published[j].PublishedAt) })
	if len(published) > 0 {
		sinceRelease := days(published[0].PublishedAt, now)
		in12m := 0
		for _, r := range published {
			if days(r.PublishedAt, now) <= 365 {
				in12m++
			}
		}
		v := 0.5*halfLife(sinceRelease, 180) + 0.5*sat(float64(in12m), 6)
		comps = append(comps, Component{Name: "releases", Value: v, Weight: 0.25, Used: true, Evidence: evidenceDays("last release", sinceRelease)})
	} else {
		comps = append(comps, Component{Name: "releases", Weight: 0.25, Evidence: "no GitHub releases"})
	}

	// 4. Bug-fix signal: fix-looking commits + patch-level releases in the last year.
	fixValue, fixUsed := 0.0, false
	if s.SampledCommits >= 10 {
		share := float64(s.FixCommits) / float64(s.SampledCommits)
		// A healthy project has roughly 10-40% fix commits; none at all is neutral-low.
		fixValue = math.Min(1, share/0.25)
		fixUsed = true
	}
	if patch := patchReleases(published, now); patch > 0 {
		fixValue = math.Min(1, fixValue+0.5*sat(float64(patch), 4))
		fixUsed = true
	}
	comps = append(comps, Component{Name: "fixes", Value: fixValue, Weight: 0.10, Used: fixUsed})

	// Redistribute the weight of unused components.
	var total float64
	for _, c := range comps {
		if c.Used {
			total += c.Weight
		}
	}
	raw := 0.0
	for i := range comps {
		if !comps[i].Used {
			comps[i].Weight = 0
			continue
		}
		comps[i].Weight = round3(comps[i].Weight / total)
		raw += comps[i].Value * comps[i].Weight
	}
	// Without release data the picture is incomplete: a small discount keeps
	// commit-only projects from ranking above equally busy projects that ship.
	if len(published) == 0 {
		raw *= 0.93
	}

	score := 1 + 4*raw
	res := Result{Raw: round3(raw), Components: comps, Version: Version}

	switch {
	case s.Archived || s.Disabled:
		score, res.Cap = 1.0, "archived"
	case sinceCommit > 730:
		score, res.Cap = math.Min(score, 1.4), "no commits for more than 2 years"
	case sinceCommit > 365:
		score, res.Cap = math.Min(score, 2.4), "no commits for more than 1 year"
	}
	res.Score = round1(math.Max(1, math.Min(5, score)))
	res.Tier = Tier(res.Score)
	res.Label = Labels[res.Tier]
	return res
}

// Tier maps a score onto the five named levels.
func Tier(score float64) int {
	switch {
	case score >= 4.5:
		return 5
	case score >= 3.5:
		return 4
	case score >= 2.5:
		return 3
	case score >= 1.7:
		return 2
	default:
		return 1
	}
}

var semver = regexp.MustCompile(`^[vV]?(\d+)\.(\d+)\.(\d+)`)

// patchReleases counts releases in the last 12 months that only bump the patch number.
func patchReleases(rs []Release, now time.Time) int {
	count := 0
	for _, r := range rs {
		if days(r.PublishedAt, now) > 365 || r.Prerelease {
			continue
		}
		m := semver.FindStringSubmatch(r.Tag)
		if m == nil || m[3] == "0" {
			continue
		}
		count++
	}
	return count
}

func round3(f float64) float64 { return math.Round(f*1000) / 1000 }

func evidenceDays(what string, d float64) string {
	if math.IsInf(d, 1) {
		return what + ": never"
	}
	return what + ": " + itoa(int(d)) + " days ago"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
