ALTER TABLE annotations ADD COLUMN citation_key TEXT;

UPDATE annotations
SET citation_key = lower(trim(selector)) || '|' || text_hash
WHERE citation_key IS NULL OR citation_key = '';

CREATE INDEX IF NOT EXISTS idx_annotations_page_citation_key ON annotations(page_id, citation_key);
