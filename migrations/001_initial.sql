PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS admins (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  github_id INTEGER NOT NULL UNIQUE,
  github_login TEXT NOT NULL UNIQUE,
  email TEXT,
  name TEXT,
  avatar_url TEXT,
  role TEXT NOT NULL CHECK (role IN ('owner','admin','moderator')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_login_at TEXT
);

CREATE TABLE IF NOT EXISTS admin_sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  admin_id INTEGER NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_admin_sessions_token_hash ON admin_sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_admin_sessions_expires_at ON admin_sessions(expires_at);

CREATE TABLE IF NOT EXISTS sites (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  allowed_origins_json TEXT NOT NULL,
  default_moderation_mode TEXT NOT NULL CHECK (default_moderation_mode IN ('manual','auto')),
  default_page_state TEXT NOT NULL CHECK (default_page_state IN ('open','locked','hidden','archived')),
  default_theme TEXT NOT NULL CHECK (default_theme IN ('auto','light','dark','minimal')),
  max_comment_length INTEGER NOT NULL,
  allow_replies INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  site_id INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
  page_key TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL CHECK (state IN ('open','locked','hidden','archived')),
  comments_count INTEGER NOT NULL DEFAULT 0,
  approved_count INTEGER NOT NULL DEFAULT 0,
  pending_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(site_id, page_key)
);

CREATE INDEX IF NOT EXISTS idx_pages_site_state ON pages(site_id, state);
CREATE INDEX IF NOT EXISTS idx_pages_page_key ON pages(page_key);

CREATE TABLE IF NOT EXISTS comments (
  id TEXT PRIMARY KEY,
  site_id INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
  page_id INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
  parent_id TEXT REFERENCES comments(id) ON DELETE SET NULL,
  root_id TEXT REFERENCES comments(id) ON DELETE SET NULL,
  depth INTEGER NOT NULL DEFAULT 0,
  path TEXT NOT NULL DEFAULT '',
  author_name TEXT NOT NULL,
  author_email_hash TEXT,
  author_website TEXT,
  body_markdown TEXT NOT NULL,
  body_html TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending','approved','rejected','spam','deleted')),
  ip_hash TEXT,
  user_agent_hash TEXT,
  metadata_json TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  edited_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_comments_page_status_created ON comments(page_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_comments_site_status_created ON comments(site_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_id);
CREATE INDEX IF NOT EXISTS idx_comments_root_path ON comments(root_id, path);
CREATE INDEX IF NOT EXISTS idx_comments_ip_hash ON comments(ip_hash);

CREATE TABLE IF NOT EXISTS ip_bans (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  site_id INTEGER REFERENCES sites(id) ON DELETE CASCADE,
  ip_hash TEXT NOT NULL,
  reason TEXT,
  created_by_admin_id INTEGER REFERENCES admins(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ip_bans_site_ip ON ip_bans(site_id, ip_hash);

CREATE TABLE IF NOT EXISTS word_bans (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  site_id INTEGER REFERENCES sites(id) ON DELETE CASCADE,
  pattern TEXT NOT NULL,
  action TEXT NOT NULL CHECK (action IN ('pending','reject','spam')),
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_word_bans_site ON word_bans(site_id);

CREATE TABLE IF NOT EXISTS moderation_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  comment_id TEXT NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
  admin_id INTEGER REFERENCES admins(id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  reason TEXT,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_moderation_events_comment ON moderation_events(comment_id, created_at);

CREATE TABLE IF NOT EXISTS comment_reactions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  comment_id TEXT NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
  reaction_key TEXT NOT NULL,
  actor_hash TEXT,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_comment_reactions_comment ON comment_reactions(comment_id, reaction_key);

CREATE TABLE IF NOT EXISTS comment_ratings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  comment_id TEXT NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
  actor_hash TEXT,
  value INTEGER NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_comment_ratings_comment ON comment_ratings(comment_id);

CREATE TABLE IF NOT EXISTS page_ratings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  page_id INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
  actor_hash TEXT,
  value INTEGER NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_page_ratings_page ON page_ratings(page_id);
