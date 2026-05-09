CREATE TABLE IF NOT EXISTS annotations (
  id TEXT PRIMARY KEY,
  site_id INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
  page_id INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
  comment_id TEXT NOT NULL UNIQUE REFERENCES comments(id) ON DELETE CASCADE,
  selector TEXT NOT NULL,
  selected_text TEXT NOT NULL,
  selection_prefix TEXT NOT NULL DEFAULT '',
  selection_suffix TEXT NOT NULL DEFAULT '',
  text_start INTEGER,
  text_end INTEGER,
  text_hash TEXT NOT NULL,
  metadata_json TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_annotations_page_created ON annotations(page_id, created_at);
CREATE INDEX IF NOT EXISTS idx_annotations_page_selector ON annotations(page_id, selector);
CREATE INDEX IF NOT EXISTS idx_annotations_comment ON annotations(comment_id);
