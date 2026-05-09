package observability

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

const readinessTimeout = 2 * time.Second

type HealthHandler struct {
	db      *sql.DB
	metrics *Metrics
}

func NewHealthHandler(db *sql.DB, metrics *Metrics) *HealthHandler {
	return &HealthHandler{db: db, metrics: metrics}
}

func (h *HealthHandler) Livez(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	h.writeStatus(w, r, false)
}

func (h *HealthHandler) Status(w http.ResponseWriter, r *http.Request) {
	h.writeStatus(w, r, true)
}

func (h *HealthHandler) writeStatus(w http.ResponseWriter, r *http.Request, includeTime bool) {
	ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
	defer cancel()

	dbOK := h.db != nil && h.db.PingContext(ctx) == nil
	if h.metrics != nil {
		h.metrics.RecordReadiness("database", dbOK)
	}

	statusCode := http.StatusOK
	status := "ok"
	if !dbOK {
		statusCode = http.StatusServiceUnavailable
		status = "unavailable"
	}

	response := map[string]any{
		"status": status,
		"checks": map[string]string{
			"database": checkStatus(dbOK),
		},
	}
	if includeTime {
		response["time"] = time.Now().UTC().Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

func checkStatus(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}
