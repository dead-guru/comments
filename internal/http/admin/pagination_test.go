package admin

import (
	"net/http/httptest"
	"testing"
)

func TestAdminPaginationBuildsSafePageLinks(t *testing.T) {
	r := httptest.NewRequest("GET", "/admin/comments?status=pending&page=2", nil)
	p := newPagination(r, "page", 2, true)
	if p == nil {
		t.Fatal("expected pagination")
	}
	if p.PerPage != adminPageSize {
		t.Fatalf("per page = %d, want %d", p.PerPage, adminPageSize)
	}
	if !p.HasPrev || !p.HasNext {
		t.Fatalf("expected prev and next links: %+v", p)
	}
	if p.PrevURL != "/admin/comments?status=pending" {
		t.Fatalf("prev URL = %q", p.PrevURL)
	}
	if p.NextURL != "/admin/comments?page=3&status=pending" {
		t.Fatalf("next URL = %q", p.NextURL)
	}
}

func TestTrimAdminPageShowsAtMostThirtyItems(t *testing.T) {
	items := make([]int, adminFetchLimit)
	trimmed, hasNext := trimAdminPage(items)
	if !hasNext {
		t.Fatal("expected hasNext for extra fetched item")
	}
	if len(trimmed) != adminPageSize {
		t.Fatalf("len = %d, want %d", len(trimmed), adminPageSize)
	}
}
