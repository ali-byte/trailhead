package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// ErrInvalidURL is wrapped into the error returned by Canonicalize when the
// input does not parse as an absolute http/https URL with a non-empty
// host. See DECISIONS.md "Invalid URL Validation Bar".
var ErrInvalidURL = errors.New("invalid URL: must be an absolute http or https URL with a host")

// trackingParams is the deny-list of known tracking/marketing query
// parameters stripped during canonicalization. Part of the locked
// canonicalization rule set in DECISIONS.md "Canonical URL - Query
// Parameters" - extend deliberately, this is not a free-standing
// convenience list. Keys are compared case-insensitively.
var trackingParams = map[string]struct{}{
	"utm_source":   {},
	"utm_medium":   {},
	"utm_campaign": {},
	"utm_term":     {},
	"utm_content":  {},
	"utm_id":       {},
	"gclid":        {},
	"fbclid":       {},
	"mc_eid":       {},
	"mc_cid":       {},
	"ref":          {},
	"igshid":       {},
}

// Canonicalize derives the canonical form of a URL per the locked rule set
// in DECISIONS.md: scheme forced to https, tracking query parameters
// stripped and remaining keys sorted alphabetically, trailing slash and
// leading "www." stripped, fragment dropped. Canonicalize is a pure
// function of its input - the same rawURL always yields the same
// CanonicalURL, and any implementation following this rule set must agree.
func Canonicalize(rawURL string) (CanonicalURL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("Canonicalize(%q): %w (%v)", rawURL, ErrInvalidURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("Canonicalize(%q): %w: scheme %q", rawURL, ErrInvalidURL, u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("Canonicalize(%q): %w: empty host", rawURL, ErrInvalidURL)
	}

	u.Scheme = "https"
	u.Host = strings.ToLower(u.Host)
	u.Host = strings.TrimPrefix(u.Host, "www.")
	u.Fragment = ""
	u.RawFragment = ""

	switch {
	case u.Path == "":
		u.Path = "/"
	case len(u.Path) > 1:
		u.Path = strings.TrimSuffix(u.Path, "/")
	}

	if u.RawQuery != "" {
		q := u.Query()
		for key := range q {
			if _, tracked := trackingParams[strings.ToLower(key)]; tracked {
				q.Del(key)
			}
		}
		keys := make([]string, 0, len(q))
		for k := range q {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		sorted := url.Values{}
		for _, k := range keys {
			for _, v := range q[k] {
				sorted.Add(k, v)
			}
		}
		u.RawQuery = sorted.Encode()
	}

	return CanonicalURL(u.String()), nil
}

// DeriveIdentityHash computes the SHA-256 hash of a CanonicalURL,
// hex-encoded. A pure function of its input - see DECISIONS.md "Identity
// Hash".
func DeriveIdentityHash(c CanonicalURL) IdentityHash {
	sum := sha256.Sum256([]byte(c))
	return IdentityHash(hex.EncodeToString(sum[:]))
}

// DefaultTitle derives a display title from a URL string alone (no remote
// fetch - see DECISIONS.md "Title Defaulting"): hostname plus a
// de-slugified last path segment, e.g.
// "https://example.com/blog/my-great-post" -> "example.com - My Great Post".
// Used by the repository implementation when NewBookmark.Title is nil.
func DefaultTitle(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}

	host := strings.ToLower(strings.TrimPrefix(u.Host, "www."))

	trimmedPath := strings.Trim(u.Path, "/")
	if trimmedPath == "" {
		return host
	}

	segments := strings.Split(trimmedPath, "/")
	lastSegment := segments[len(segments)-1]
	lastSegment = strings.ReplaceAll(lastSegment, "-", " ")
	lastSegment = strings.ReplaceAll(lastSegment, "_", " ")
	lastSegment = strings.TrimSpace(lastSegment)

	if lastSegment == "" {
		return host
	}

	return fmt.Sprintf("%s - %s", host, titleCase(lastSegment))
}

// titleCase upper-cases the first letter of each whitespace-separated word.
// A small local helper rather than golang.org/x/text/cases, to keep
// internal/domain free of any non-standard-library dependency.
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if w == "" {
			continue
		}
		runes := []rune(w)
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}
