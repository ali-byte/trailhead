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

	"github.com/jackc/pgx/v5"
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

// statusPositionConstraint is the unique index added by migration 000002
// enforcing distinct (status, position) pairs — see DECISIONS.md "Position
// Collision Handling (Decision B, resolved)". A 23505 on this constraint
// means a concurrent writer landed in the same rank gap first; Create and
// Move both re-resolve current adjacency and retry rather than failing.
const statusPositionConstraint = "bookmarks_status_position_unique_idx"

// maxRankRetries bounds the retry loop Create and Move both run on a
// statusPositionConstraint collision — see DECISIONS.md "Position
// Collision Handling (Decision B, resolved)" for why a bounded retry is
// preferred over re-spacing. DECISIONS.md doesn't pin an exact count
// ("bounded (~5)" is the interview's rough estimate for the two-writer
// Move case); 20 is set higher deliberately — repository_test.go's
// TestCreateBookmark_ConcurrentDifferentURLs_StableOrderByPositionThenID
// runs 10 goroutines racing Create's front-of-Inbox insert simultaneously,
// and a bound of 5 was empirically too tight for that contention level
// and produced real (non-flaky-test-only) failures.
const maxRankRetries = 20

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

	bookmark, err := r.insertWithRetry(ctx, b, canonical, identityHash)
	if err != nil {
		return domain.Bookmark{}, err
	}

	slog.DebugContext(ctx, "postgres.Create: exit", "id", bookmark.ID)
	return bookmark, nil
}

// insertWithRetry computes a front-insert rank and attempts the INSERT,
// retrying on a statusPositionConstraint collision (Decision B) — a
// concurrent Create landed in the same front-of-Inbox gap first, so this
// re-reads the current front rank and recomputes before trying again.
// Bounded to maxRankRetries, matching Move's retry contract.
func (r *Repository) insertWithRetry(ctx context.Context, b domain.NewBookmark, canonical domain.CanonicalURL, identityHash domain.IdentityHash) (domain.Bookmark, error) {
	var lastErr error
	for range maxRankRetries {
		position, err := r.frontInsertPosition(ctx)
		if err != nil {
			return domain.Bookmark{}, fmt.Errorf("Create(identity_hash=%s): resolve front rank: %w", identityHash, err)
		}

		row := r.insertBookmarkRow(ctx, b, canonical, identityHash, position)
		bookmark, err := scanBookmark(row)
		if err == nil {
			return bookmark, nil
		}
		if IsStatusPositionConstraintViolation(err) {
			lastErr = err
			continue
		}
		return domain.Bookmark{}, r.mapCreateError(ctx, identityHash, err)
	}
	return domain.Bookmark{}, fmt.Errorf("Create(identity_hash=%s): exceeded retry bound after position collisions: %w", identityHash, lastErr)
}

// frontInsertPosition computes a rank strictly before the current first
// Inbox row (or a starting rank if Inbox is empty) — see DECISIONS.md
// "Position / Ordering Representation" and "Position Collision Handling
// (Decision B, resolved)".
func (r *Repository) frontInsertPosition(ctx context.Context) (string, error) {
	const q = `SELECT position FROM bookmarks WHERE status = $1 ORDER BY position ASC LIMIT 1`
	var first string
	err := r.pool.QueryRow(ctx, q, string(domain.StatusInbox)).Scan(&first)
	if errors.Is(err, pgx.ErrNoRows) {
		return midpoint("", ""), nil
	}
	if err != nil {
		return "", fmt.Errorf("frontInsertPosition: %w", err)
	}
	return midpoint("", first), nil
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
// Bookmark (always enters StatusInbox). position is the caller-computed
// front-insert rank — see frontInsertPosition.
func (r *Repository) insertBookmarkRow(ctx context.Context, b domain.NewBookmark, canonical domain.CanonicalURL, identityHash domain.IdentityHash, position string) scannable {
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
		position,
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

// Move changes a Bookmark's Status and/or Position — see
// adapter.BookmarkRepository.Move. The write itself is a single
// UPDATE ... RETURNING (moveRow) so a pre-commit failure leaves the row
// completely unchanged; existence folds into that same statement (0 rows
// updated => ErrKindNotFound). Neighbor bounds are resolved from current
// DB state before each attempt and re-resolved on every retry — see
// resolveNeighborBounds and DECISIONS.md "Position Collision Handling
// (Decision B, resolved)".
func (r *Repository) Move(ctx context.Context, cmd adapter.MoveCommand) (domain.Bookmark, error) {
	slog.DebugContext(ctx, "postgres.Move: enter", "id", cmd.ID, "target_status", cmd.TargetStatus)

	var lastErr error
	for range maxRankRetries {
		lo, hi, err := r.resolveNeighborBounds(ctx, cmd)
		if err != nil {
			return domain.Bookmark{}, fmt.Errorf("Move(id=%s): resolve neighbors: %w", cmd.ID, err)
		}
		position := midpoint(lo, hi)

		bookmark, err := r.moveRow(ctx, cmd, position)
		if err == nil {
			slog.DebugContext(ctx, "postgres.Move: exit", "id", cmd.ID)
			return bookmark, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Bookmark{}, &adapter.RepositoryError{
				Kind:    adapter.ErrKindNotFound,
				Message: fmt.Sprintf("Move(id=%s): bookmark not found", cmd.ID),
				Wrapped: err,
			}
		}
		if IsStatusPositionConstraintViolation(err) {
			lastErr = err
			continue
		}
		slog.ErrorContext(ctx, "postgres.Move: update failed", "id", cmd.ID, "error", err)
		return domain.Bookmark{}, fmt.Errorf("Move(id=%s): %w", cmd.ID, err)
	}
	return domain.Bookmark{}, fmt.Errorf("Move(id=%s): exceeded retry bound after position collisions: %w", cmd.ID, lastErr)
}

// moveRow executes Move's single UPDATE ... RETURNING. The finished_at
// CASE enforces the FinishedAt <-> Done invariant atomically against the
// row's pre-update status (DECISIONS.md "FinishedAt <-> Done invariant"):
// entering Done stamps finished_at to now, leaving Done clears it to NULL,
// and Done -> Done preserves the existing value while still bumping
// updated_at. cmd.TargetStatus is trusted, not validated — see
// DECISIONS.md "MoveCommand.TargetStatus - Validation Ownership (E1,
// resolved)"; a garbage value fails via the status CHECK constraint here,
// surfacing as a plain wrapped error, never a *RepositoryError.
func (r *Repository) moveRow(ctx context.Context, cmd adapter.MoveCommand, position string) (domain.Bookmark, error) {
	const q = `
		UPDATE bookmarks
		SET status = $2,
		    position = $3,
		    finished_at = CASE
		        WHEN $2 = 'done' AND status <> 'done' THEN $4
		        WHEN $2 <> 'done' THEN NULL
		        ELSE finished_at
		    END,
		    updated_at = $4
		WHERE id = $1
		RETURNING ` + bookmarkColumns

	now := r.now()
	row := r.pool.QueryRow(ctx, q, string(cmd.ID), string(cmd.TargetStatus), position, now)
	return scanBookmark(row)
}

// resolveNeighborBounds derives the (lo, hi) rank bounds midpoint should
// compute between, per the locked neighbor-fallback rule (DECISIONS.md
// "Move - Neighbor Fallback (Generalized)"): Before wins unconditionally
// when set, ignoring After outright; a missing/stale, cross-status, or
// self-referential neighbor falls back to end-of-column. Re-run on every
// retry so a losing writer sees the winner's just-committed row.
func (r *Repository) resolveNeighborBounds(ctx context.Context, cmd adapter.MoveCommand) (lo, hi string, err error) {
	if cmd.Before != nil {
		pos, ok, err := r.validNeighborPosition(ctx, *cmd.Before, cmd.TargetStatus, cmd.ID)
		if err != nil {
			return "", "", err
		}
		if !ok {
			return r.endOfColumnBounds(ctx, cmd.TargetStatus, cmd.ID)
		}
		succ, hasSucc, err := r.adjacentPosition(ctx, cmd.TargetStatus, cmd.ID, pos, true)
		if err != nil {
			return "", "", err
		}
		if !hasSucc {
			return pos, "", nil
		}
		return pos, succ, nil
	}

	if cmd.After != nil {
		pos, ok, err := r.validNeighborPosition(ctx, *cmd.After, cmd.TargetStatus, cmd.ID)
		if err != nil {
			return "", "", err
		}
		if !ok {
			return r.endOfColumnBounds(ctx, cmd.TargetStatus, cmd.ID)
		}
		pred, hasPred, err := r.adjacentPosition(ctx, cmd.TargetStatus, cmd.ID, pos, false)
		if err != nil {
			return "", "", err
		}
		if !hasPred {
			return "", pos, nil
		}
		return pred, pos, nil
	}

	return r.endOfColumnBounds(ctx, cmd.TargetStatus, cmd.ID)
}

// validNeighborPosition reports whether id is usable as a Move neighbor:
// it must exist, currently be in targetStatus, and not equal movingID —
// see DECISIONS.md "Move - Neighbor Fallback (Generalized)" for the three
// ways a neighbor reference can fail to resolve.
func (r *Repository) validNeighborPosition(ctx context.Context, id domain.BookmarkID, targetStatus domain.Status, movingID domain.BookmarkID) (position string, ok bool, err error) {
	if id == movingID {
		return "", false, nil
	}

	const q = `SELECT position, status FROM bookmarks WHERE id = $1`
	var status string
	err = r.pool.QueryRow(ctx, q, string(id)).Scan(&position, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("validNeighborPosition(id=%s): %w", id, err)
	}
	if status != string(targetStatus) {
		return "", false, nil
	}
	return position, true, nil
}

// adjacentPosition finds the row immediately after (forward=true) or
// before (forward=false) at in the target column, excluding excludeID (the
// bookmark being moved, in case it's already in that column). Used to
// derive the far bound of the (lo, hi) pair once the near bound (the
// caller-supplied Before/After neighbor) is fixed.
func (r *Repository) adjacentPosition(ctx context.Context, status domain.Status, excludeID domain.BookmarkID, at string, forward bool) (string, bool, error) {
	q := `SELECT position FROM bookmarks WHERE status = $1 AND id <> $2 AND position > $3 ORDER BY position ASC LIMIT 1`
	if !forward {
		q = `SELECT position FROM bookmarks WHERE status = $1 AND id <> $2 AND position < $3 ORDER BY position DESC LIMIT 1`
	}

	var position string
	err := r.pool.QueryRow(ctx, q, string(status), string(excludeID), at).Scan(&position)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("adjacentPosition(status=%s): %w", status, err)
	}
	return position, true, nil
}

// endOfColumnBounds returns the (lo, hi) pair for an end-of-column
// insert: lo is the current last row's position in status (excluding
// excludeID), or "" if the column is empty; hi is always "" (unbounded
// above).
func (r *Repository) endOfColumnBounds(ctx context.Context, status domain.Status, excludeID domain.BookmarkID) (lo, hi string, err error) {
	const q = `SELECT position FROM bookmarks WHERE status = $1 AND id <> $2 ORDER BY position DESC LIMIT 1`
	var position string
	err = r.pool.QueryRow(ctx, q, string(status), string(excludeID)).Scan(&position)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("endOfColumnBounds(status=%s): %w", status, err)
	}
	return position, "", nil
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

// IsStatusPositionConstraintViolation reports whether err is a Postgres
// 23505 specifically on statusPositionConstraint — the signal that Create
// or Move must re-resolve current rank adjacency and retry, rather than
// fail the request. See DECISIONS.md "Position Collision Handling
// (Decision B, resolved)".
func IsStatusPositionConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == statusPositionConstraint
}
