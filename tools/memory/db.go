package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const dbFilename = "memory.db"

// DB holds the SQLite database connection and workspace path.
type DB struct {
	db      *sql.DB
	baseDir string
	dbPath  string
}

// NewDB creates a new database instance and initializes the schema.
func NewDB(baseDir string) (*DB, error) {
	dbPath := filepath.Join(baseDir, dbFilename)

	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{db: db, baseDir: baseDir, dbPath: dbPath}, nil
}

// initSchema creates the memory and history tables if they don't exist.
func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS memory (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT UNIQUE NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		entry TEXT NOT NULL
	);
	
	CREATE INDEX IF NOT EXISTS idx_history_timestamp ON history(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_memory_key ON memory(key);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// Memory Operations

// MemoryGet retrieves a memory entry by key.
func (d *DB) MemoryGet(key string) (string, bool, error) {
	var content string
	err := d.db.QueryRow("SELECT content FROM memory WHERE key = ?", key).Scan(&content)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to query memory: %w", err)
	}
	return content, true, nil
}

// MemoryList returns all memory keys.
func (d *DB) MemoryList() ([]string, error) {
	rows, err := d.db.Query("SELECT key FROM memory ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("failed to query memory: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return keys, nil
}

// MemoryWrite performs a compare-exchange write.
// Returns (success, error).
func (d *DB) MemoryWrite(key, oldValue, newValue string) (bool, error) {
	if oldValue == "" {
		// Key doesn't exist - insert
		_, err := d.db.Exec("INSERT INTO memory (key, content) VALUES (?, ?)", key, newValue)
		if err != nil {
			return false, fmt.Errorf("failed to insert memory: %w", err)
		}
		return true, nil
	}

	// Key exists - compare-exchange
	result, err := d.db.Exec("UPDATE memory SET content = ?, updated_at = ? WHERE key = ? AND content = ?",
		newValue, time.Now(), key, oldValue)
	if err != nil {
		return false, fmt.Errorf("failed to update memory: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// No rows updated - old value didn't match
		return false, nil
	}

	return true, nil
}

// History Operations

// HistoryAdd adds a new entry to history.
func (d *DB) HistoryAdd(entry string) error {
	_, err := d.db.Exec("INSERT INTO history (entry) VALUES (?)", entry)
	if err != nil {
		return fmt.Errorf("failed to add history entry: %w", err)
	}
	return nil
}

// HistorySearch searches history entries by text match.
func (d *DB) HistorySearch(query string) ([]string, error) {
	rows, err := d.db.Query("SELECT entry FROM history WHERE entry LIKE ? ORDER BY timestamp DESC LIMIT 50", "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var entries []string
	for rows.Next() {
		var entry string
		if err := rows.Scan(&entry); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return entries, nil
}

// HistorySince returns entries since a timestamp (ISO8601 format).
func (d *DB) HistorySince(timestamp string) ([]string, error) {
	rows, err := d.db.Query("SELECT entry FROM history WHERE timestamp >= ? ORDER BY timestamp DESC", timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var entries []string
	for rows.Next() {
		var entry string
		if err := rows.Scan(&entry); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return entries, nil
}
