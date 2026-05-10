package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenAppliesSQLitePragmas(t *testing.T) {
	database, err := OpenWithOptions(filepath.Join(t.TempDir(), "test.db"), Options{MaxOpenConns: 4, MaxIdleConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	if got := queryStringPragma(t, database, "PRAGMA journal_mode;"); got != "wal" {
		t.Fatalf("expected WAL journal mode, got %q", got)
	}
	if got := queryIntPragma(t, database, "PRAGMA foreign_keys;"); got != 1 {
		t.Fatalf("expected foreign_keys=1, got %d", got)
	}
	if got := queryIntPragma(t, database, "PRAGMA synchronous;"); got != 1 {
		t.Fatalf("expected synchronous=NORMAL(1), got %d", got)
	}
	if got := queryIntPragma(t, database, "PRAGMA busy_timeout;"); got != defaultBusyTimeoutMS {
		t.Fatalf("expected busy_timeout=%d, got %d", defaultBusyTimeoutMS, got)
	}
	if got := queryIntPragma(t, database, "PRAGMA cache_size;"); got != -defaultCacheSizeKiB {
		t.Fatalf("expected cache_size=%d, got %d", -defaultCacheSizeKiB, got)
	}
	if got := queryIntPragma(t, database, "PRAGMA temp_store;"); got != 2 {
		t.Fatalf("expected temp_store=MEMORY(2), got %d", got)
	}
	if got := queryInt64Pragma(t, database, "PRAGMA mmap_size;"); got != int64(defaultMmapSizeBytes) {
		t.Fatalf("expected mmap_size=%d, got %d", defaultMmapSizeBytes, got)
	}
}

func TestOpenConfiguresPoolLimit(t *testing.T) {
	database, err := OpenWithOptions(filepath.Join(t.TempDir(), "test.db"), Options{MaxOpenConns: 6, MaxIdleConns: 3})
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	if got := database.Stats().MaxOpenConnections; got != 6 {
		t.Fatalf("expected max open connections 6, got %d", got)
	}
}

func queryStringPragma(t *testing.T, database *sql.DB, sql string) string {
	t.Helper()
	var got string
	if err := database.QueryRowContext(context.Background(), sql).Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

func queryIntPragma(t *testing.T, database *sql.DB, sql string) int {
	t.Helper()
	var got int
	if err := database.QueryRowContext(context.Background(), sql).Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

func queryInt64Pragma(t *testing.T, database *sql.DB, sql string) int64 {
	t.Helper()
	var got int64
	if err := database.QueryRowContext(context.Background(), sql).Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}
