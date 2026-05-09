package service

import "testing"

func TestWordBanMatchesPlainSubstringOnly(t *testing.T) {
	if !wordBanMatches("this body has a.*b literally", "a.*b") {
		t.Fatal("expected literal word ban pattern to match")
	}
	if wordBanMatches("this body has axxxb", "a.*b") {
		t.Fatal("word ban patterns must not be treated as regex")
	}
	if wordBanMatches("anything", " ") {
		t.Fatal("blank word ban patterns must not match")
	}
}

func TestLinkCountCountsRawURLsOnce(t *testing.T) {
	if got := linkCount("http://example.com/https://path HTTPS://other.example"); got != 2 {
		t.Fatalf("expected 2 raw links, got %d", got)
	}
}

func TestLinkCountCountsMarkdownLinksWithoutDoubleCounting(t *testing.T) {
	body := "[one](https://one.example) and [two](http://two.example/path)"
	if got := linkCount(body); got != 2 {
		t.Fatalf("expected 2 markdown links, got %d", got)
	}
}

func TestLinkCountIgnoresMarkdownImages(t *testing.T) {
	body := "![alt](https://img.example/a.png) and text"
	if got := linkCount(body); got != 0 {
		t.Fatalf("expected markdown image not to count as a link, got %d", got)
	}
}

func TestLinkCountCountsRawAndMarkdownLinks(t *testing.T) {
	body := "https://raw.example and [one](https://one.example) and ![alt](https://img.example/a.png)"
	if got := linkCount(body); got != 2 {
		t.Fatalf("expected raw plus markdown link only, got %d", got)
	}
}
