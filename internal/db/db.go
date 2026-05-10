package db

import (
	"context"
	"database/sql"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultBusyTimeoutMS          = 5000
	defaultCacheSizeKiB           = 64 * 1024
	defaultMmapSizeBytes          = 256 * 1024 * 1024
	defaultWALAutoCheckpointPages = 1000
	defaultConnectionMaxLifetime  = time.Hour
	defaultConnectionMaxIdleTime  = 10 * time.Minute
	minDefaultOpenConnections     = 4
	maxDefaultOpenConnections     = 16
	maxDefaultIdleConnections     = 4
	sqliteConnectTimeout          = 5 * time.Second
)

type Options struct {
	MaxOpenConns int
	MaxIdleConns int
}

func Open(path string) (*sql.DB, error) {
	return OpenWithOptions(path, Options{})
}

func OpenWithOptions(path string, opts Options) (*sql.DB, error) {
	database, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, err
	}
	maxOpen := opts.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = defaultMaxOpenConns()
	}
	maxIdle := opts.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = min(maxOpen, maxDefaultIdleConnections)
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	database.SetMaxOpenConns(maxOpen)
	database.SetMaxIdleConns(maxIdle)
	database.SetConnMaxLifetime(defaultConnectionMaxLifetime)
	database.SetConnMaxIdleTime(defaultConnectionMaxIdleTime)
	ctx, cancel := context.WithTimeout(context.Background(), sqliteConnectTimeout)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, err
	}
	return database, nil
}

func sqliteDSN(path string) string {
	pragmas := url.Values{}
	pragmas.Add("_pragma", "journal_mode(WAL)")
	pragmas.Add("_pragma", "synchronous(NORMAL)")
	pragmas.Add("_pragma", "busy_timeout("+itoa(defaultBusyTimeoutMS)+")")
	pragmas.Add("_pragma", "foreign_keys(ON)")
	pragmas.Add("_pragma", "cache_size(-"+itoa(defaultCacheSizeKiB)+")")
	pragmas.Add("_pragma", "temp_store(MEMORY)")
	pragmas.Add("_pragma", "mmap_size("+itoa(defaultMmapSizeBytes)+")")
	pragmas.Add("_pragma", "wal_autocheckpoint("+itoa(defaultWALAutoCheckpointPages)+")")
	pragmas.Set("_txlock", "immediate")

	raw := strings.TrimSpace(path)
	if raw == "" {
		raw = "deadcomments.db"
	}
	if raw == ":memory:" {
		return "file::memory:?cache=shared&" + pragmas.Encode()
	}
	if strings.HasPrefix(raw, "file:") {
		return appendQuery(raw, pragmas.Encode())
	}
	return appendQuery("file:"+raw, pragmas.Encode())
}

func appendQuery(base, query string) string {
	if strings.Contains(base, "?") {
		return base + "&" + query
	}
	return base + "?" + query
}

func defaultMaxOpenConns() int {
	n := runtime.NumCPU() * 2
	if n < minDefaultOpenConnections {
		return minDefaultOpenConnections
	}
	if n > maxDefaultOpenConnections {
		return maxDefaultOpenConnections
	}
	return n
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
