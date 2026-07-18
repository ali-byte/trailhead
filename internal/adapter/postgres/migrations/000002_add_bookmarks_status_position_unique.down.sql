DROP INDEX bookmarks_status_position_unique_idx;

CREATE INDEX bookmarks_status_position_idx ON bookmarks (status, position);
