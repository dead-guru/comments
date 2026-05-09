package observability

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestReadinessReportsDatabaseState(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	handler := NewHealthHandler(database, NewMetrics("deadcomments_test"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	handler.Readyz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected ready status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"database":"ok"`) {
		t.Fatalf("expected database ok response, got %s", rec.Body.String())
	}

	_ = database.Close()
	rec = httptest.NewRecorder()
	handler.Readyz(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected failed ready status 503, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"database":"fail"`) {
		t.Fatalf("expected database fail response, got %s", rec.Body.String())
	}
}
