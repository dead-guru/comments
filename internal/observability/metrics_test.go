package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"deadcomments/internal/domain"
)

func TestHTTPMiddlewareExportsRouteMetrics(t *testing.T) {
	metrics := NewMetrics("deadcomments_test")
	router := chi.NewRouter()
	router.Use(metrics.Middleware)
	router.Get("/comments/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("accepted"))
	})
	router.Handle("/metrics", metrics.Handler())

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/comments/abc", nil))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	body := rec.Body.String()
	if !strings.Contains(body, `deadcomments_test_http_requests_total`) {
		t.Fatalf("expected http request counter in metrics output")
	}
	if !strings.Contains(body, `route="/comments/{id}"`) {
		t.Fatalf("expected route pattern label, got %s", body)
	}
	if !strings.Contains(body, `code="202"`) {
		t.Fatalf("expected response code label, got %s", body)
	}
}

func TestEventMetricsUseBoundedLabels(t *testing.T) {
	metrics := NewMetrics("deadcomments_test")
	handler := NewEventMetricsHandler(metrics)

	err := handler.Handle(t.Context(), domain.Event{
		Type:          domain.EventCommentCreated,
		AggregateType: "comment",
		AggregateID:   "comment-1",
		Payload: map[string]any{
			"status":        domain.CommentPending,
			"reason":        "user controlled text should not become a label",
			"tripcode_kind": domain.TripcodeAnonymous,
			"parent_id":     "parent-1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	body := rec.Body.String()
	if !strings.Contains(body, `deadcomments_test_comments_created_total`) {
		t.Fatalf("expected comments created metric")
	}
	if !strings.Contains(body, `reason="none"`) {
		t.Fatalf("expected unrecognized reason to use bounded fallback label, got %s", body)
	}
	if !strings.Contains(body, `tripcode_kind="anonymous"`) {
		t.Fatalf("expected tripcode kind label, got %s", body)
	}
	if !strings.Contains(body, `has_parent="true"`) {
		t.Fatalf("expected reply label, got %s", body)
	}
}
