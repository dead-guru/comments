CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  actor_admin_id INTEGER REFERENCES admins(id) ON DELETE SET NULL,
  site_id INTEGER REFERENCES sites(id) ON DELETE SET NULL,
  page_id INTEGER REFERENCES pages(id) ON DELETE SET NULL,
  comment_id TEXT REFERENCES comments(id) ON DELETE SET NULL,
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  payload_json TEXT NOT NULL DEFAULT '{}',
  occurred_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_type_occurred ON events(type, occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_site_occurred ON events(site_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_page_occurred ON events(page_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_comment_occurred ON events(comment_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_actor_occurred ON events(actor_admin_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(aggregate_type, aggregate_id, occurred_at);

CREATE TABLE IF NOT EXISTS event_deliveries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  handler_key TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending','delivered','failed')),
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  delivered_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(event_id, handler_key)
);

CREATE INDEX IF NOT EXISTS idx_event_deliveries_handler_status ON event_deliveries(handler_key, status, updated_at);
