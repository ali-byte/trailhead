package postgres

import "testing"

func TestMidpoint_EmptyColumn_ReturnsNonEmptyStartingRank(t *testing.T) {
	got := midpoint("", "")
	if got == "" {
		t.Fatalf("midpoint(\"\", \"\") = %q, want non-empty", got)
	}
}

func TestMidpoint_AdjacentSingleChars_ExtendsLength(t *testing.T) {
	got := midpoint("a", "b")
	want := "an"
	if got != want {
		t.Fatalf("midpoint(%q, %q) = %q, want %q", "a", "b", got, want)
	}
	if got <= "a" || got >= "b" {
		t.Fatalf("midpoint(\"a\", \"b\") = %q, not strictly between", got)
	}
}

func TestMidpoint_RoomForMiddleDigit_SingleChar(t *testing.T) {
	got := midpoint("a", "d")
	if got <= "a" || got >= "d" {
		t.Fatalf("midpoint(\"a\", \"d\") = %q, not strictly between", got)
	}
	if len(got) != 1 {
		t.Fatalf("midpoint(\"a\", \"d\") = %q, want a single-character result when there's room", got)
	}
}

func TestMidpoint_NoLowerBound(t *testing.T) {
	got := midpoint("", "b")
	if got >= "b" {
		t.Fatalf("midpoint(\"\", \"b\") = %q, want < \"b\"", got)
	}
	if got == "" {
		t.Fatalf("midpoint(\"\", \"b\") = %q, want non-empty", got)
	}
}

func TestMidpoint_NoUpperBound(t *testing.T) {
	got := midpoint("m", "")
	if got <= "m" {
		t.Fatalf("midpoint(\"m\", \"\") = %q, want > \"m\"", got)
	}
}

func TestMidpoint_NoUpperBound_FromEmptyLo(t *testing.T) {
	got := midpoint("", "")
	if got == "" {
		t.Fatalf("midpoint(\"\", \"\") = %q, want non-empty", got)
	}
}

// TestMidpoint_RepeatedNarrowingGap_NeverExhaustsPrecision mirrors
// move_test.go's TestMove_RepeatedInsertsIntoSameGap_MaintainsStrictOrder
// at the pure-function level: repeatedly inserting into the same
// narrowing gap must always produce a strictly increasing chain, however
// long the ranks grow.
func TestMidpoint_RepeatedNarrowingGap_NeverExhaustsPrecision(t *testing.T) {
	lo, hi := "b", "z"
	prev := lo
	for i := range 50 {
		got := midpoint(prev, hi)
		if got <= prev || got >= hi {
			t.Fatalf("iteration %d: midpoint(%q, %q) = %q, not strictly between", i, prev, hi, got)
		}
		prev = got
	}
}

// TestMidpoint_RepeatedFrontInserts mirrors Create's revised front-insert
// scheme: repeatedly inserting at the front (lo == "") against a
// shrinking upper bound must also never exhaust precision.
func TestMidpoint_RepeatedFrontInserts(t *testing.T) {
	hi := "z"
	for i := range 50 {
		got := midpoint("", hi)
		if got >= hi {
			t.Fatalf("iteration %d: midpoint(\"\", %q) = %q, not < hi", i, hi, got)
		}
		if got == "" {
			t.Fatalf("iteration %d: midpoint(\"\", %q) returned empty", i, hi)
		}
		hi = got
	}
}

// TestMidpoint_RepeatedBackInserts mirrors Move's end-of-column fallback:
// repeatedly appending at the back (hi == "") must keep producing strictly
// increasing ranks with no upper bound to exhaust.
func TestMidpoint_RepeatedBackInserts(t *testing.T) {
	lo := "b"
	for i := range 50 {
		got := midpoint(lo, "")
		if got <= lo {
			t.Fatalf("iteration %d: midpoint(%q, \"\") = %q, not > lo", i, lo, got)
		}
		lo = got
	}
}

func TestMidpoint_NeverEndsInMinimumDigit(t *testing.T) {
	cases := []struct{ lo, hi string }{
		{"", ""},
		{"", "b"},
		{"a", ""},
		{"a", "b"},
		{"a", "ab"},
		{"m", "z"},
	}
	for _, c := range cases {
		got := midpoint(c.lo, c.hi)
		if got[len(got)-1] == rankDigits[0] {
			t.Fatalf("midpoint(%q, %q) = %q, must not end in the minimum digit %q", c.lo, c.hi, got, string(rankDigits[0]))
		}
	}
}

// TestMidpoint_ManyNeighborsDistinctAndOrdered is a broader stress check:
// generate a sequence of ranks by repeated between-insertion and confirm
// the whole chain sorts strictly and every value is distinct.
func TestMidpoint_ManyNeighborsDistinctAndOrdered(t *testing.T) {
	chain := []string{"a", "z"}
	for i := range 30 {
		insertAt := (i % (len(chain) - 1)) + 1
		lo, hi := chain[insertAt-1], chain[insertAt]
		mid := midpoint(lo, hi)
		chain = append(chain[:insertAt], append([]string{mid}, chain[insertAt:]...)...)
	}
	seen := make(map[string]bool, len(chain))
	for i, r := range chain {
		if seen[r] {
			t.Fatalf("duplicate rank %q in generated chain", r)
		}
		seen[r] = true
		if i > 0 && chain[i-1] >= chain[i] {
			t.Fatalf("chain not strictly increasing at index %d: %q >= %q", i, chain[i-1], chain[i])
		}
	}
}

func TestMidpoint_DocumentedExample(t *testing.T) {
	if got := midpoint("a", "b"); got != "an" {
		t.Fatalf("midpoint(\"a\", \"b\") = %q, want %q", got, "an")
	}
}
