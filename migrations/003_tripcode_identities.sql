CREATE TABLE IF NOT EXISTS identities (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  site_id INTEGER REFERENCES sites(id) ON DELETE CASCADE,
  display_name TEXT NOT NULL,
  normalized_name TEXT NOT NULL,
  type TEXT NOT NULL DEFAULT 'reserved' CHECK (type IN ('reserved','admin','system')),
  secret_hash TEXT NOT NULL,
  public_tripcode TEXT NOT NULL,
  badge_type TEXT NOT NULL DEFAULT 'verified' CHECK (badge_type IN ('verified','admin','author','custom')),
  badge_label TEXT,
  created_by_admin_id INTEGER REFERENCES admins(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_identities_global_name ON identities(normalized_name) WHERE site_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_identities_site_name ON identities(site_id, normalized_name) WHERE site_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_identities_site ON identities(site_id, normalized_name);

ALTER TABLE comments ADD COLUMN identity_id INTEGER REFERENCES identities(id) ON DELETE SET NULL;
ALTER TABLE comments ADD COLUMN tripcode_public TEXT;
ALTER TABLE comments ADD COLUMN tripcode_kind TEXT NOT NULL DEFAULT 'none' CHECK (tripcode_kind IN ('none','anonymous','reserved'));
ALTER TABLE comments ADD COLUMN author_display_name TEXT;

UPDATE comments SET author_display_name = author_name WHERE author_display_name IS NULL;

CREATE INDEX IF NOT EXISTS idx_comments_identity_id ON comments(identity_id);
CREATE INDEX IF NOT EXISTS idx_comments_tripcode_kind ON comments(tripcode_kind);
