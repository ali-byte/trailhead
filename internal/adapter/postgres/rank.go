package postgres

import "strings"

// rankDigits is the base-26 alphabet used for Position ranks, ascending
// ASCII (a < b < ... < z) so byte-order comparison matches the position
// column's COLLATE "C" — see DECISIONS.md "Position Collision Handling
// (Decision B, resolved)".
const rankDigits = "abcdefghijklmnopqrstuvwxyz"

// midpoint returns a rank string strictly between lo and hi in byte order:
// lo < midpoint(lo, hi) < hi. lo == "" means no lower bound (insert at the
// very front of the column); hi == "" means no upper bound (insert at the
// very back). midpoint("", "") — an empty column — returns a starting rank
// with room to grow on both sides.
//
// Ported from the Greenspan/Figma fractional-indexing algorithm
// (https://www.figma.com/blog/realtime-editing-of-ordered-sequences/),
// restricted to the fixed 26-letter alphabet above instead of that
// algorithm's general base-62 digit set. When lo and hi have adjacent
// first digits (no room for a middle character at that position), the
// algorithm extends length rather than failing — e.g. midpoint("a", "b")
// == "an" — so repeated inserts into a narrowing gap never exhaust
// precision. A generated rank never ends in rankDigits[0] ('a'): every
// return path either picks a strictly-greater digit or appends a
// trailing "n", so a self-consistent invariant holds across every rank
// this function ever produces (real DB fixtures in this codebase never
// use a bare "...a" rank either, so this invariant is never violated by
// mixed real/synthetic input).
func midpoint(lo, hi string) string {
	if hi == "" {
		// Unbounded above: any proper extension of lo sorts after it,
		// unboundedly, so there is always room — no digit arithmetic
		// needed.
		return lo + "n"
	}
	if lo != "" && lo >= hi {
		panic("postgres: midpoint(lo, hi): lo must be < hi")
	}
	return midpointBounded(lo, hi)
}

// midpointBounded implements midpoint's core case: hi is a real, non-empty
// upper bound. It walks lo and hi digit by digit (padding lo's missing
// digits with rankDigits[0], the algorithm's "no lower bound" convention)
// until they diverge, then either drops a middle digit into the gap or,
// if the gap is only one digit wide, fixes that digit and recurses
// unbounded (via midpoint's hi == "" case) on lo's remaining suffix.
func midpointBounded(lo, hi string) string {
	n := 0
	for n < len(hi) {
		loDigit := byte(rankDigits[0])
		if n < len(lo) {
			loDigit = lo[n]
		}
		if loDigit != hi[n] {
			break
		}
		n++
	}
	if n >= len(hi) {
		// No rank exists strictly between lo and hi in this alphabet: hi is
		// matched byte-for-byte by lo's zero-padded digits, so every
		// extension of lo sorts at or after hi (e.g. midpoint("", "a") or
		// midpoint("b", "ba") — nothing lies between "" and "a", nor
		// between "b" and "ba"). This is unreachable for ranks this package
		// generates — they never end in rankDigits[0] ('a') — so reaching
		// it means an 'a'-terminated rank entered from outside the
		// algorithm (a raw insert, seed, or manual edit). Fail loudly
		// rather than return a silently out-of-bounds rank; the caller's
		// retry/rebalance path, not a bogus rank, is the correct response.
		panic("postgres: midpoint(lo, hi): no rank exists between lo and hi (hi ends in the minimum digit 'a'); a non-'a'-terminated rank or a rebalance is required")
	}

	prefix := hi[:n]
	digitA := 0
	if n < len(lo) {
		digitA = strings.IndexByte(rankDigits, lo[n])
	}
	digitB := strings.IndexByte(rankDigits, hi[n])

	// A rank byte outside a–z means the input is not a canonical base-26
	// rank (rankDigits[x] would index with -1). This package never produces
	// such a rank; reaching here means a corrupt/non-canonical rank entered
	// from outside the algorithm (raw insert, seed, manual edit). Same
	// scoped-guard posture as the no-gap panic above — a DB CHECK enforcing
	// position ~ '^[a-z]+$' is the production-grade enforcement (see H-23).
	if digitA < 0 || digitB < 0 {
		panic("postgres: midpoint(lo, hi): rank contains a byte outside a–z; ranks must be canonical base-26")
	}

	if digitB-digitA >= 2 {
		mid := digitA + (digitB-digitA)/2
		return prefix + string(rankDigits[mid])
	}

	var loRemainder string
	if n+1 < len(lo) {
		loRemainder = lo[n+1:]
	}
	return prefix + string(rankDigits[digitA]) + loRemainder + "n"
}
