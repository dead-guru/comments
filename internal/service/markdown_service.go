package service

import dcmarkdown "deadcomments/internal/markdown"

type MarkdownService struct {
	renderer *dcmarkdown.Renderer
}

func NewMarkdownService(renderer *dcmarkdown.Renderer) *MarkdownService {
	return &MarkdownService{renderer: renderer}
}

func (s *MarkdownService) Render(input string) (string, error) {
	return s.renderer.Render(input)
}
