package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const migrationSQL = `
CREATE TABLE IF NOT EXISTS videos (
    id                 TEXT PRIMARY KEY,
    user_id            TEXT NOT NULL,
    user_name          TEXT NOT NULL,
    user_email         TEXT NOT NULL,
    file_name          TEXT NOT NULL,
    duration_seconds   INTEGER NOT NULL,
    size_bytes         INTEGER NOT NULL,
    status             TEXT NOT NULL DEFAULT 'PENDING',
    bucket_name        TEXT NOT NULL,
    video_chunk_folder TEXT NOT NULL,
    image_folder       TEXT NOT NULL,
    download_url       TEXT,
    error_cause        TEXT,
    created_at         TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at         TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS chunks (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    video_id         TEXT NOT NULL REFERENCES videos(id),
    part_number      INTEGER NOT NULL,
    start_time       INTEGER NOT NULL,
    end_time         INTEGER NOT NULL,
    frame_per_second INTEGER NOT NULL,
    status           TEXT NOT NULL DEFAULT 'PENDING',
    video_object_id  TEXT NOT NULL,
    UNIQUE(video_id, part_number)
);
`

func NewSQLiteDB(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite db: %w", err)
	}

	return db, nil
}

func RunMigrations(db *sql.DB) error {
	if _, err := db.Exec(migrationSQL); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
