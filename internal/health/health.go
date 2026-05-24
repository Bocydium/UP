package health

import (
	"time"

	"github.com/aapollo/up/internal/aur"
)

// Score represents a package's health rating (0-100).
type Score struct {
	Value     int
	Breakdown map[string]int
	Verdict   string
}

// Calculate computes a health score for an AUR package.
func Calculate(pkg *aur.Package) Score {
	breakdown := make(map[string]int)

	// Votes (max 30 points)
	voteScore := pkg.Votes / 4
	if voteScore > 30 {
		voteScore = 30
	}
	breakdown["votes"] = voteScore

	// Maintainer activity (max 25 points)
	maintainerScore := 25
	if pkg.Maintainer == "" {
		maintainerScore = 0
	}
	breakdown["maintainer"] = maintainerScore

	// Freshness (max 25 points)
	freshScore := 25
	if pkg.OutOfDate > 0 {
		age := int(time.Since(time.Unix(pkg.OutOfDate, 0)).Hours() / 24)
		if age > 365 {
			freshScore = 0
		} else if age > 180 {
			freshScore = 10
		} else if age > 90 {
			freshScore = 15
		} else if age > 30 {
			freshScore = 20
		} else {
			freshScore = 22
		}
	}
	breakdown["freshness"] = freshScore

	// Popularity indicator (max 20 points)
	popScore := 0
	if pkg.Votes > 500 {
		popScore = 20
	} else if pkg.Votes > 100 {
		popScore = 15
	} else if pkg.Votes > 20 {
		popScore = 10
	} else if pkg.Votes > 5 {
		popScore = 5
	}
	breakdown["popularity"] = popScore

	total := voteScore + maintainerScore + freshScore + popScore

	verdict := "excellent"
	if total < 70 {
		verdict = "good"
	}
	if total < 50 {
		verdict = "fair"
	}
	if total < 30 {
		verdict = "poor"
	}
	if total < 15 {
		verdict = "dangerous"
	}

	return Score{
		Value:     total,
		Breakdown: breakdown,
		Verdict:   verdict,
	}
}

// Bar returns a visual bar for the score.
func (s Score) Bar() string {
	filled := s.Value / 5
	empty := 20 - filled
	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}
	return bar
}

// Color returns the ANSI color for the score.
func (s Score) Color() string {
	if s.Value >= 70 {
		return "\033[32m"
	}
	if s.Value >= 50 {
		return "\033[33m"
	}
	if s.Value >= 30 {
		return "\033[38;5;208m"
	}
	return "\033[31m"
}
