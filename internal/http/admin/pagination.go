package admin

import (
	"net/http"
	"net/url"
	"strconv"
)

const (
	adminPageSize   = 30
	adminFetchLimit = adminPageSize + 1
)

type paginationView struct {
	Page    int
	PerPage int
	HasPrev bool
	HasNext bool
	PrevURL string
	NextURL string
}

func adminPage(r *http.Request, key string) (int, int, int) {
	page, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || page < 1 {
		page = 1
	}
	return page, adminFetchLimit, (page - 1) * adminPageSize
}

func trimAdminPage[T any](items []T) ([]T, bool) {
	if len(items) <= adminPageSize {
		return items, false
	}
	return items[:adminPageSize], true
}

func newPagination(r *http.Request, key string, page int, hasNext bool) *paginationView {
	if page <= 1 && !hasNext {
		return nil
	}
	p := &paginationView{
		Page:    page,
		PerPage: adminPageSize,
		HasPrev: page > 1,
		HasNext: hasNext,
	}
	if p.HasPrev {
		p.PrevURL = pageURL(r, key, page-1)
	}
	if p.HasNext {
		p.NextURL = pageURL(r, key, page+1)
	}
	return p
}

func pageURL(r *http.Request, key string, page int) string {
	q := r.URL.Query()
	if page <= 1 {
		q.Del(key)
	} else {
		q.Set(key, strconv.Itoa(page))
	}
	return (&url.URL{Path: r.URL.Path, RawQuery: q.Encode()}).RequestURI()
}
