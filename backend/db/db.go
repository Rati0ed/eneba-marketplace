// Package db opens the SQLite database, creates the tables and fills them with sample data.
package db

import (
	"database/sql"
	_ "embed" // lets us use //go:embed below
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3" // registers the "sqlite3" driver with database/sql
)

// schemaSQL holds the contents of schema.sql. //go:embed copies the file INTO the
// compiled program, so the server never needs to find schema.sql on disk at runtime.
//
//go:embed schema.sql
var schemaSQL string

// Open creates a FRESH database at path: it deletes any old file, creates all
// tables and seeds sample data. We rebuild on every start on purpose:
//   - Render's free tier has no permanent disk anyway,
//   - every start gives the same known data, and
//   - schema changes never need migrations.
func Open(path string) (*sql.DB, error) {
	// Remove the old database and SQLite's helper files. Errors are ignored
	// because the files may simply not exist yet.
	for _, f := range []string{path, path + "-wal", path + "-shm"} {
		os.Remove(f)
	}

	// Connection string options (handled by the mattn driver, applied to EVERY connection):
	//   _foreign_keys=on   enforce REFERENCES (SQLite ignores them by default)
	//   _journal_mode=WAL  readers don't block the writer and vice versa
	//   _busy_timeout=5000 if the DB is locked, wait up to 5 s instead of failing at once
	//   _txlock=immediate  every transaction takes the write lock at BEGIN, not at its first write.
	//                      SQLite allows one writer at a time; locking up front means two checkouts
	//                      queue up cleanly instead of both reading stock and then colliding.
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000&_txlock=immediate", path)

	// sql.Open does not connect yet; it only prepares a connection pool.
	database, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := database.Exec(schemaSQL); err != nil {
		database.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	if err := seed(database); err != nil {
		database.Close()
		return nil, fmt.Errorf("seed data: %w", err)
	}

	return database, nil
}
