package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"trailhead/internal/adapter"
	"trailhead/internal/domain"
)

// identityHashConstraint is the unique index whose violation means "this
// URL already exists" — see ARCHITECTURE_RFC.md "Persistence Schema".
// A 23505 on any other constraint is an infrastructure failure, not a
// duplicate-URL classification.
const identityHashConstraint = "bookmarks_identity_hash_idx"

// Repository implements adapter.BookmarkRepository against Postgres using
// pgx/v5 directly over a pgxpool.Pool — see ARCHITECTURE_RFC.md "Postgres
// Driver". now is an injectable clock (DECISIONS.md "Timestamp source —
// injectable clock") so tests can freeze time instead of asserting against
// a moving target.
type Repository struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

// New wires a Repository over pool using now as its clock source.
// Production callers pass func() time.Time { return time.Now().UTC() }.
func New(pool *pgxpool.Pool, now func() time.Time) *Repository {
	return &Repository{pool: pool, now: now}
}

var _ adapter.BookmarkRepository = (*Repository)(nil)

const bookmarkColumns = `id::text, original_url, canonical_url, identity_hash, title, tags, status, position, finished_at, author, created_at, updated_at`

// Create persists a new Bookmark — see adapter.BookmarkRepository.Create.
func (r *Repository) Create(ctx context.Context, b domain.NewBookmark) (domain.Bookmark, error) {
	slog.DebugContext(ctx, "postgres.Create: enter")

	canonical, identityHash, err := canonicalizeForCreate(b.OriginalURL)
	if err != nil {
		// err from domain.Canonicalize embeds the raw URL via %q — never
		// log or wrap it directly. Only the sentinel (domain.ErrInvalidURL)
		// and a non-reversible fingerprint are safe to surface.
		fingerprint := fingerprintURL(b.OriginalURL)
		slog.DebugContext(ctx, "postgres.Create: invalid url", "url_fingerprint", fingerprint, "error", domain.ErrInvalidURL)
		return domain.Bookmark{}, &adapter.RepositoryError{
			Kind:    adapter.ErrKindInvalidURL,
			Message: fmt.Sprintf("Create(fingerprint=%s): invalid URL", fingerprint),
			Wrapped: domain.ErrInvalidURL,
		}
	}

	row := r.insertBookmarkRow(ctx, b, canonical, identityHash)
	bookmark, err := scanBookmark(row)
	if err != nil {
		return domain.Bookmark{}, r.mapCreateError(ctx, identityHash, err)
	}

	slog.DebugContext(ctx, "postgres.Create: exit", "id", bookmark.ID)
	return bookmark, nil
}

// canonicalizeForCreate derives the CanonicalURL and IdentityHash Create
// needs before touching the database — split out so Create itself stays
// focused on orchestration.
func canonicalizeForCreate(originalURL string) (domain.CanonicalURL, domain.IdentityHash, error) {
	canonical, err := domain.Canonicalize(originalURL)
	if err != nil {
		return "", "", err
	}
	return canonical, domain.DeriveIdentityHash(canonical), nil
}

// insertBookmarkRow executes Create's INSERT ... RETURNING — one now()
// value written to both created_at and updated_at (DECISIONS.md "UpdatedAt
// - Write-Path Contract"), NULL for finished_at/author on every new
// Bookmark (always enters StatusInbox).
func (r *Repository) insertBookmarkRow(ctx context.Context, b domain.NewBookmark, canonical domain.CanonicalURL, identityHash domain.IdentityHash) scannable {
	resolvedTitle := domain.DefaultTitle(b.OriginalURL)
	if b.Title != nil {
		resolvedTitle = *b.Title
	}

	const q = `
		INSERT INTO bookmarks (original_url, canonical_url, identity_hash, title, tags, status, position, finished_at, author, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10, $10)
		RETURNING ` + bookmarkColumns

	return r.pool.QueryRow(ctx, q,
		b.OriginalURL,
		string(canonical),
		string(identityHash),
		resolvedTitle,
		tagsToJSON(normalizeTags(b.Tags)),
		string(domain.StatusInbox),
		initialPosition,
		(*time.Time)(nil), // finished_at
		(*string)(nil),    // author
		r.now(),
	)
}

// mapCreateError classifies an INSERT failure: a duplicate identity_hash
// re-queries the pre-existing row and returns *adapter.RepositoryError;
// anything else is an infrastructure failure, wrapped plain. Logs and
// error messages identify the bookmark by IdentityHash, never the raw
// URL — original_url may carry credentials or tokens in its query string
// and must not land in logs or error text.
func (r *Repository) mapCreateError(ctx context.Context, identityHash domain.IdentityHash, err error) error {
	if !IsDuplicateConstraintViolation(err) {
		slog.ErrorContext(ctx, "postgres.Create: insert failed", "identity_hash", string(identityHash), "error", err)
		return fmt.Errorf("Create(identity_hash=%s): %w", identityHash, err)
	}

	slog.DebugContext(ctx, "postgres.Create: duplicate identity hash", "identity_hash", string(identityHash))
	existing, findErr := r.findByIdentityHash(ctx, identityHash)
	if findErr != nil {
		return fmt.Errorf("Create(identity_hash=%s): duplicate insert but existing row lookup failed: %w", identityHash, findErr)
	}
	return &adapter.RepositoryError{
		Kind:     adapter.ErrKindDuplicate,
		Existing: &existing,
		Message:  fmt.Sprintf("Create(identity_hash=%s): duplicate URL", identityHash),
		Wrapped:  err,
	}
}

// fingerprintURL derives a short, non-reversible correlation token for a
// URL that failed canonicalization (so no CanonicalURL/IdentityHash is
// available yet) — logs and error text must never carry the raw URL, which
// may contain credentials or tokens in its query string.
func fingerprintURL(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return hex.EncodeToString(sum[:6])
}

// findByIdentityHash re-queries the pre-existing row after a duplicate
// insert — see ARCHITECTURE_RFC.md "Persistence Schema" notes on Create's
// error-mapping responsibility.
func (r *Repository) findByIdentityHash(ctx context.Context, hash domain.IdentityHash) (domain.Bookmark, error) {
	const q = `SELECT ` + bookmarkColumns + ` FROM bookmarks WHERE identity_hash = $1`
	row := r.pool.QueryRow(ctx, q, string(hash))
	return scanBookmark(row)
}

// Board returns every Bookmark grouped by Status and ordered by Position —
// see adapter.BookmarkRepository.Board.
//
// Tag filtering (BoardFilter.Tags) is out of scope for issue #2 — see the
// issue body's "Out of Scope" section ("Tag filtering in Board beyond an
// empty filter (Phase D)"). Rather than silently ignoring a non-empty
// filter, Board fails loudly so a future caller can't mistake "not yet
// implemented" for "no bookmarks matched."
func (r *Repository) Board(ctx context.Context, filter adapter.BoardFilter) (domain.Board, error) {
	slog.DebugContext(ctx, "postgres.Board: enter", "tags", filter.Tags)

	if len(filter.Tags) > 0 {
		return domain.Board{}, fmt.Errorf("Board(%v): tag filtering is unsupported until Phase D (issue #2 scope)", filter)
	}

	const q = `SELECT ` + bookmarkColumns + ` FROM bookmarks ORDER BY status, position, id`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		slog.ErrorContext(ctx, "postgres.Board: query failed", "error", err)
		return domain.Board{}, fmt.Errorf("Board(%v): %w", filter, err)
	}
	defer rows.Close()

	board := domain.Board{
		Inbox:   []domain.Bookmark{},
		Reading: []domain.Bookmark{},
		Done:    []domain.Bookmark{},
	}

	for rows.Next() {
		bookmark, err := scanBookmarkRow(rows)
		if err != nil {
			return domain.Board{}, fmt.Errorf("Board(%v): scan row: %w", filter, err)
		}
		switch bookmark.Status {
		case domain.StatusInbox:
			board.Inbox = append(board.Inbox, bookmark)
		case domain.StatusReading:
			board.Reading = append(board.Reading, bookmark)
		case domain.StatusDone:
			board.Done = append(board.Done, bookmark)
		default:
			return domain.Board{}, fmt.Errorf("Board(%v): row %s has unknown status %q", filter, bookmark.ID, bookmark.Status)
		}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "postgres.Board: row iteration failed", "error", err)
		return domain.Board{}, fmt.Errorf("Board(%v): %w", filter, err)
	}

	slog.DebugContext(ctx, "postgres.Board: exit", "inbox", len(board.Inbox), "reading", len(board.Reading), "done", len(board.Done))
	return board, nil
}

// Move is issue #4 scope — not implemented here.
func (r *Repository) Move(_ context.Context, _ adapter.MoveCommand) (domain.Bookmark, error) {
	return domain.Bookmark{}, errors.New("postgres.Repository.Move: not implemented — issue #4")
}

// Update is issue #5 scope — not implemented here.
func (r *Repository) Update(_ context.Context, _ domain.BookmarkID, _ domain.BookmarkPatch) (domain.Bookmark, error) {
	return domain.Bookmark{}, errors.New("postgres.Repository.Update: not implemented — issue #5")
}

// Delete is issue #5 scope — not implemented here.
func (r *Repository) Delete(_ context.Context, _ domain.BookmarkID) error {
	return errors.New("postgres.Repository.Delete: not implemented — issue #5")
}

// scannable abstracts over pgx.Row and pgx.Rows for scanBookmark reuse.
type scannable interface {
	Scan(dest ...any) error
}

func scanBookmark(row scannable) (domain.Bookmark, error) {
	return scanBookmarkRow(row)
}

func scanBookmarkRow(row scannable) (domain.Bookmark, error) {
	var (
		b         domain.Bookmark
		tagsJSON  []byte
		status    string
		position  string
		originalU string
	)

	err := row.Scan(
		&b.ID,
		&originalU,
		&b.CanonicalURL,
		&b.IdentityHash,
		&b.Title,
		&tagsJSON,
		&status,
		&position,
		&b.FinishedAt,
		&b.Author,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		return domain.Bookmark{}, err
	}

	tags, err := jsonToTags(tagsJSON)
	if err != nil {
		return domain.Bookmark{}, fmt.Errorf("scan bookmark %s: %w", b.ID, err)
	}

	b.OriginalURL = originalU
	b.Status = domain.Status(status)
	b.Position = domain.Position(position)
	b.Tags = tags

	return b, nil
}

// normalizeTags lowercases, deduplicates, and drops empty-string tags —
// see DECISIONS.md "Locked From Brief".
func normalizeTags(raw domain.Tags) domain.Tags {
	seen := make(map[string]struct{}, len(raw))
	normalized := make(domain.Tags, 0, len(raw))
	for _, tag := range raw {
		lower := strings.ToLower(strings.TrimSpace(tag))
		if lower == "" {
			continue
		}
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		normalized = append(normalized, lower)
	}
	sort.Strings(normalized)
	return normalized
}

// tagsToJSON marshals a normalized Tags slice to JSON for the jsonb
// column — never nil, per the locked "tags jsonb NOT NULL DEFAULT
// '[]'::jsonb" schema.
func tagsToJSON(tags domain.Tags) []byte {
	if tags == nil {
		tags = domain.Tags{}
	}
	b, err := json.Marshal(tags)
	if err != nil {
		// tags is always []string of plain lowercase strings — marshal
		// cannot fail for this input shape.
		panic(fmt.Sprintf("tagsToJSON: unexpected marshal failure: %v", err))
	}
	return b
}

// jsonToTags unmarshals the jsonb tags column back into domain.Tags. A
// malformed tags column is data corruption, not an empty tag set — it
// must surface as an error rather than being silently dropped.
func jsonToTags(raw []byte) (domain.Tags, error) {
	var tags domain.Tags
	if len(raw) == 0 {
		return domain.Tags{}, nil
	}
	if err := json.Unmarshal(raw, &tags); err != nil {
		return nil, fmt.Errorf("jsonToTags: malformed tags column %q: %w", raw, err)
	}
	if tags == nil {
		tags = domain.Tags{}
	}
	return tags, nil
}

// IsDuplicateConstraintViolation reports whether err is a Postgres 23505
// (unique_violation) specifically on bookmarks_identity_hash_idx — the
// only constraint whose violation means "duplicate URL." A 23505 on any
// other constraint (e.g. bookmarks_pkey) is not a duplicate-URL error.
func IsDuplicateConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == identityHashConstraint
}

// initialPosition is the Position assigned to every newly Created
// Bookmark. Distinct-position enforcement within a Status is deferred to
// issue #4 (Move) — see ARCHITECTURE_RFC.md "Persistence Schema" and
// DECISIONS.md "Position / Ordering Representation".
const initialPosition = "m"
