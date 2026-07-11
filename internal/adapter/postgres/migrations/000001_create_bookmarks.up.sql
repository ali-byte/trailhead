CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE bookmarks (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    original_url   text NOT NULL,
    canonical_url  text NOT NULL,
    identity_hash  text NOT NULL,
    title          text NOT NULL,
    tags           jsonb NOT NULL DEFAULT '[]'::jsonb,
    status         text NOT NULL CHECK (status IN ('inbox', 'reading', 'done')),
    position       text COLLATE "C" NOT NULL,
    finished_at    timestamptz,
    author         text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX bookmarks_identity_hash_idx ON bookmarks (identity_hash);
CREATE INDEX bookmarks_status_position_idx ON bookmarks (status, position);
CREATE INDEX bookmarks_tags_gin_idx ON bookmarks USING gin (tags);
