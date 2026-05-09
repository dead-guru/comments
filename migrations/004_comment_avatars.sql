ALTER TABLE comments ADD COLUMN author_avatar_hash TEXT;

CREATE INDEX IF NOT EXISTS idx_comments_author_avatar_hash ON comments(author_avatar_hash);
