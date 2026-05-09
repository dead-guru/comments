CREATE INDEX IF NOT EXISTS idx_comments_public_thread_sort
ON comments(page_id, status, COALESCE(root_id, id), created_at);
