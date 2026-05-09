package markdown

import "github.com/microcosm-cc/bluemonday"

type Sanitizer struct {
	policy *bluemonday.Policy
}

func NewSanitizer() *Sanitizer {
	p := bluemonday.UGCPolicy()
	p.AllowElements("table", "thead", "tbody", "tfoot", "tr", "th", "td", "del", "pre", "code")
	p.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("code")
	p.AllowAttrs("rel").OnElements("a")
	p.RequireNoFollowOnLinks(true)
	p.RequireNoReferrerOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	return &Sanitizer{policy: p}
}

func (s *Sanitizer) Sanitize(input string) string {
	return s.policy.Sanitize(input)
}
