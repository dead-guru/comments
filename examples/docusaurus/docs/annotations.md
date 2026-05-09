---
sidebar_position: 4
title: Inline annotations
---

# Inline annotations

This page enables Medium-style inline comments for the article body. Select any text in this document and deadcomments will open a small form anchored to that selection.

The annotation script is separate from the iframe widget. It uses the same comments backend, moderation rules, tripcode identity support, Markdown renderer, and origin allowlist. Approved annotations are restored as highlights when the page loads again.

## Try a selection

Select this paragraph and leave a short note. The selected quote, surrounding context, and text offsets are stored in a dedicated annotations table, while the actual message is stored as a normal comment.

Deadcomments keeps the text highlight independent from the public iframe thread. That means a blog can use page-level comments, inline annotations, or both on the same article without mixing the rendering models.

```html
<script
  src="http://localhost:8080/annotations.js"
  data-site="docs-demo"
  data-page="/docs/annotations"
  data-content-selector=".theme-doc-markdown"
  data-theme="auto"
  data-locale="en">
</script>
```

<div data-deadcomments-annotations="/docs/annotations" data-content-selector=".theme-doc-markdown"></div>

<div className="deadcomments-demo" data-deadcomments-page="/docs/annotations" data-input-position="bottom"></div>
