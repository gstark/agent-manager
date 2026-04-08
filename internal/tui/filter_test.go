package tui

import "testing"

func TestWeightedNameFirstFilterPrefersNameMatches(t *testing.T) {
	targets := []string{
		"gh-search" + filterFieldSeparator + "search helper",
		"alpha" + filterFieldSeparator + "gh helper",
		"ghost" + filterFieldSeparator + "misc",
	}

	ranks := weightedNameFirstFilter("gh", targets)
	if len(ranks) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(ranks))
	}

	if ranks[0].Index == 1 {
		t.Fatalf("expected a name match to rank first, got description-only match at index %d", ranks[0].Index)
	}

	if ranks[1].Index == 1 {
		t.Fatalf("expected both name matches to rank ahead of description-only match, got index %d in second position", ranks[1].Index)
	}

	if ranks[2].Index != 1 {
		t.Fatalf("expected description-only match to rank last, got index %d", ranks[2].Index)
	}
}

func TestWeightedNameFirstFilterUsesDescriptionWhenNeeded(t *testing.T) {
	targets := []string{
		"alpha" + filterFieldSeparator + "beta gamma",
		"delta" + filterFieldSeparator + "epsilon",
	}

	ranks := weightedNameFirstFilter("bg", targets)
	if len(ranks) != 1 {
		t.Fatalf("expected 1 match, got %d", len(ranks))
	}

	if ranks[0].Index != 0 {
		t.Fatalf("expected description match at index 0, got %d", ranks[0].Index)
	}
}
