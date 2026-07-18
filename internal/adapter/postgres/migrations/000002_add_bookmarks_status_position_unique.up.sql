DROP INDEX bookmarks_status_position_idx;

CREATE UNIQUE INDEX bookmarks_status_position_unique_idx ON bookmarks (status, position);
