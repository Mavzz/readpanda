// Package testdb gives integration tests a real, throwaway PostgreSQL database
// built from scripts/readpanda_schema.sql.
//
// Set TEST_DATABASE_URL to a server the tests may create databases on, e.g.
//
//	TEST_DATABASE_URL="postgres://localhost:5432/postgres?sslmode=disable" go test ./...
//
// Without it, every test that asks for a database is skipped, so `go test`
// still runs the unit tests on a machine with no Postgres.
package testdb

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Mavzz/readpanda/api-go/internal/database"
	_ "github.com/lib/pq"
)

const envURL = "TEST_DATABASE_URL"

var (
	adminURL string
	dbName   string
	ready    bool
)

// Main wraps a package's TestMain: it creates a fresh database, loads the
// schema, points database.DB at it, runs the tests and drops it again.
//
//	func TestMain(m *testing.M) { testdb.Main(m) }
func Main(m *testing.M) {
	adminURL = os.Getenv(envURL)
	if adminURL == "" {
		os.Exit(m.Run()) // Require will skip
	}
	if err := create(); err != nil {
		log.Fatalf("testdb: %v", err)
	}
	code := m.Run()
	if err := drop(); err != nil {
		log.Printf("testdb: dropping %s: %v", dbName, err)
	}
	os.Exit(code)
}

// Require skips the test when no database is configured, and otherwise
// empties every table so each test starts from the bare schema.
func Require(t *testing.T) *sql.DB {
	t.Helper()
	if !ready {
		t.Skipf("%s not set; skipping database test", envURL)
	}
	// The request logger prints a line per call; only show it with -v.
	if !testing.Verbose() {
		log.SetOutput(io.Discard)
	}
	if _, err := database.DB.Exec(truncateAll); err != nil {
		t.Fatalf("testdb: truncating: %v", err)
	}
	return database.DB
}

// truncateAll empties every table in public without needing a list that
// drifts from the schema.
const truncateAll = `DO $$
DECLARE stmt text;
BEGIN
  SELECT 'TRUNCATE ' || string_agg(format('%I.%I', schemaname, tablename), ', ') || ' RESTART IDENTITY CASCADE'
    INTO stmt FROM pg_tables WHERE schemaname = 'public';
  IF stmt IS NOT NULL THEN EXECUTE stmt; END IF;
END $$;`

func create() error {
	buf := make([]byte, 4)
	rand.Read(buf)
	dbName = "readpanda_test_" + hex.EncodeToString(buf)

	admin, err := sql.Open("postgres", adminURL)
	if err != nil {
		return err
	}
	defer admin.Close()
	if _, err := admin.Exec("CREATE DATABASE " + dbName); err != nil {
		return fmt.Errorf("creating %s: %w", dbName, err)
	}

	testURL, err := withDatabase(adminURL, dbName)
	if err != nil {
		return err
	}

	schema, err := os.ReadFile(schemaPath())
	if err != nil {
		return err
	}
	// The pg_dump header clears search_path for the session, so load it on a
	// connection of its own rather than one the tests will reuse.
	loader, err := sql.Open("postgres", testURL)
	if err != nil {
		return err
	}
	_, err = loader.Exec(string(schema))
	loader.Close()
	if err != nil {
		return fmt.Errorf("loading schema: %w", err)
	}

	database.DB, err = sql.Open("postgres", testURL)
	if err != nil {
		return err
	}
	ready = true
	return database.DB.Ping()
}

func drop() error {
	if database.DB != nil {
		database.DB.Close()
	}
	admin, err := sql.Open("postgres", adminURL)
	if err != nil {
		return err
	}
	defer admin.Close()
	_, err = admin.Exec("DROP DATABASE IF EXISTS " + dbName + " WITH (FORCE)")
	return err
}

func withDatabase(raw, name string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%s: %w", envURL, err)
	}
	u.Path = "/" + name
	return u.String(), nil
}

func schemaPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "scripts", "readpanda_schema.sql")
}
