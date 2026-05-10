package assets

import "embed"

//go:embed templates/*.html templates/admin/*.html templates/embed/*.html
var Templates embed.FS

//go:embed static/admin.css static/admin.js static/embed.css widget/widget.js widget/annotations.js
var Static embed.FS
