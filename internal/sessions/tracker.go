// Package sessions tracks running action sessions in a SQLite database.
// It records start/end times by diffing successive running session scans,
// and provides elapsed time and daily totals for active sessions.
package sessions

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// ActiveInfo contains timing info for an active session.
type ActiveInfo struct {
	Elapsed int `json:"elapsed"` // seconds since session start
	Today   int `json:"today"`   // total seconds today for this place+action
}

// Tracker manages session recording in SQLite.
type Tracker struct {
	db       *sql.DB
	mu       sync.Mutex
	previous map[string]bool // key = "place:action"
}

func dbPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "places")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "sessions.db"), nil
}

// Open creates or opens the sessions database.
func Open() (*Tracker, error) {
	path, err := dbPath()
	if err != nil {
		return nil, fmt.Errorf("sessions db path: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_time_format=sqlite")
	if err != nil {
		return nil, fmt.Errorf("open sessions db: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			place TEXT NOT NULL,
			action TEXT NOT NULL,
			started_at INTEGER NOT NULL,
			ended_at INTEGER,
			last_seen_at INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_active ON sessions(place, action, ended_at);
		CREATE INDEX IF NOT EXISTS idx_today ON sessions(started_at);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create sessions table: %w", err)
	}

	t := &Tracker{
		db:       db,
		previous: make(map[string]bool),
	}

	// Close stale sessions from previous runs (crash recovery).
	t.closeStale()

	return t, nil
}

// Close closes the database.
func (t *Tracker) Close() {
	if t.db != nil {
		t.db.Close()
	}
}

// closeStale ends any sessions left open from a previous run,
// using last_seen_at as the end time.
func (t *Tracker) closeStale() {
	t.db.Exec(`
		UPDATE sessions SET ended_at = last_seen_at
		WHERE ended_at IS NULL
	`)
}

// key returns a dedup key for a place+action pair.
func key(place, action string) string {
	return place + ":" + action
}

// Update diffs the current running sessions against the previous state,
// recording starts and ends in the database.
func (t *Tracker) Update(running []struct{ Place, Action string }) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now().Unix()
	current := make(map[string]bool, len(running))

	for _, r := range running {
		k := key(r.Place, r.Action)
		current[k] = true

		if !t.previous[k] {
			// New session started.
			t.db.Exec(`
				INSERT INTO sessions (place, action, started_at, last_seen_at)
				VALUES (?, ?, ?, ?)
			`, r.Place, r.Action, now, now)
		} else {
			// Still running — update last_seen_at.
			t.db.Exec(`
				UPDATE sessions SET last_seen_at = ?
				WHERE place = ? AND action = ? AND ended_at IS NULL
			`, now, r.Place, r.Action)
		}
	}

	// Sessions that ended.
	for k := range t.previous {
		if !current[k] {
			for i := 0; i < len(k); i++ {
				if k[i] == ':' {
					place, action := k[:i], k[i+1:]
					t.db.Exec(`
						UPDATE sessions SET ended_at = ?, last_seen_at = ?
						WHERE place = ? AND action = ? AND ended_at IS NULL
					`, now, now, place, action)
					break
				}
			}
		}
	}

	t.previous = current
}

// CloseAll ends all open sessions (called on graceful shutdown).
func (t *Tracker) CloseAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now().Unix()
	t.db.Exec(`
		UPDATE sessions SET ended_at = ?, last_seen_at = ?
		WHERE ended_at IS NULL
	`, now, now)
	t.previous = make(map[string]bool)
}

// GetActiveInfo returns elapsed time and today's total for an active session.
func (t *Tracker) GetActiveInfo(place, action string) *ActiveInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Current session elapsed.
	var startedAt int64
	err := t.db.QueryRow(`
		SELECT started_at FROM sessions
		WHERE place = ? AND action = ? AND ended_at IS NULL
		ORDER BY started_at DESC LIMIT 1
	`, place, action).Scan(&startedAt)
	if err != nil {
		return nil
	}

	elapsed := int(time.Now().Unix() - startedAt)

	// Today's total (completed sessions + current).
	todayStart := time.Now().Truncate(24 * time.Hour).Unix()
	nowUnix := time.Now().Unix()
	var totalSeconds sql.NullInt64
	t.db.QueryRow(`
		SELECT SUM(
			COALESCE(ended_at, ?) - started_at
		) FROM sessions
		WHERE place = ? AND action = ? AND started_at >= ?
	`, nowUnix, place, action, todayStart).Scan(&totalSeconds)

	todayTotal := 0
	if totalSeconds.Valid {
		todayTotal = int(totalSeconds.Int64)
	}

	return &ActiveInfo{
		Elapsed: elapsed,
		Today:   todayTotal,
	}
}
