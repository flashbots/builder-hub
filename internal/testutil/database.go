package testutil

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
)

const TestDatabaseName = "builder_hub_test"

// PostgresDSN returns the explicitly configured test database DSN after
// verifying that it points to a dedicated database on the local machine.
// Database tests truncate tables, so they must fail closed if the target is
// not unmistakably disposable.
func PostgresDSN(t testing.TB) string {
	t.Helper()

	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("skipping database test; RUN_DB_TESTS is not set to 1")
	}

	dsn := strings.TrimSpace(os.Getenv("TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Fatal("refusing to run destructive database test: TEST_POSTGRES_DSN is not set")
	}
	if err := validatePostgresDSN(dsn); err != nil {
		t.Fatalf("refusing to run destructive database test: %v", err)
	}

	return dsn
}

func validatePostgresDSN(dsn string) error {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("invalid TEST_POSTGRES_DSN: %w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return fmt.Errorf("unsupported TEST_POSTGRES_DSN scheme %q", parsed.Scheme)
	}

	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("non-loopback host %q", host)
	}

	databaseName := strings.TrimPrefix(parsed.Path, "/")
	if databaseName != TestDatabaseName {
		return fmt.Errorf("database is %q; expected %q", databaseName, TestDatabaseName)
	}

	return nil
}
