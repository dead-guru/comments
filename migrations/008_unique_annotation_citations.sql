WITH duplicate_annotations AS (
  SELECT
    annotations.id,
    annotations.comment_id,
    first_value(annotations.comment_id) OVER (
      PARTITION BY annotations.page_id, annotations.citation_key
      ORDER BY annotations.created_at, annotations.id
    ) AS canonical_comment_id,
    row_number() OVER (
      PARTITION BY annotations.page_id, annotations.citation_key
      ORDER BY annotations.created_at, annotations.id
    ) AS duplicate_rank
  FROM annotations
  WHERE annotations.citation_key IS NOT NULL
    AND annotations.citation_key != ''
),
duplicate_roots AS (
  SELECT comment_id, canonical_comment_id
  FROM duplicate_annotations
  WHERE duplicate_rank > 1
)
UPDATE comments
SET
  root_id = (
    SELECT duplicate_roots.canonical_comment_id
    FROM duplicate_roots
    WHERE duplicate_roots.comment_id = comments.root_id
  ),
  updated_at = datetime('now')
WHERE root_id IN (SELECT comment_id FROM duplicate_roots);

WITH duplicate_annotations AS (
  SELECT
    annotations.id,
    annotations.comment_id,
    first_value(annotations.comment_id) OVER (
      PARTITION BY annotations.page_id, annotations.citation_key
      ORDER BY annotations.created_at, annotations.id
    ) AS canonical_comment_id,
    row_number() OVER (
      PARTITION BY annotations.page_id, annotations.citation_key
      ORDER BY annotations.created_at, annotations.id
    ) AS duplicate_rank
  FROM annotations
  WHERE annotations.citation_key IS NOT NULL
    AND annotations.citation_key != ''
),
duplicate_roots AS (
  SELECT comment_id, canonical_comment_id
  FROM duplicate_annotations
  WHERE duplicate_rank > 1
)
UPDATE comments
SET
  parent_id = (
    SELECT duplicate_roots.canonical_comment_id
    FROM duplicate_roots
    WHERE duplicate_roots.comment_id = comments.id
  ),
  root_id = (
    SELECT duplicate_roots.canonical_comment_id
    FROM duplicate_roots
    WHERE duplicate_roots.comment_id = comments.id
  ),
  depth = 1,
  path = (
    SELECT duplicate_roots.canonical_comment_id || '/' || comments.id
    FROM duplicate_roots
    WHERE duplicate_roots.comment_id = comments.id
  ),
  updated_at = datetime('now')
WHERE id IN (SELECT comment_id FROM duplicate_roots);

WITH duplicate_annotations AS (
  SELECT
    annotations.id,
    row_number() OVER (
      PARTITION BY annotations.page_id, annotations.citation_key
      ORDER BY annotations.created_at, annotations.id
    ) AS duplicate_rank
  FROM annotations
  WHERE annotations.citation_key IS NOT NULL
    AND annotations.citation_key != ''
)
DELETE FROM annotations
WHERE id IN (
  SELECT id
  FROM duplicate_annotations
  WHERE duplicate_rank > 1
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_annotations_page_citation_key_unique
ON annotations(page_id, citation_key)
WHERE citation_key IS NOT NULL AND citation_key != '';
