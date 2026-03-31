// Package sessions tracks running action sessions in a SQLite database.
// It records start/end times by diffing successive running session scans,
// and provides elapsed time and daily totals for active sessions.
package sessions

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func logErr(context string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "places-app: sessions: %s: %v\n", context, err)
	}
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
	_, err := t.db.Exec(`
		UPDATE sessions SET ended_at = last_seen_at
		WHERE ended_at IS NULL
	`)
	logErr("close stale", err)
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
			_, err := t.db.Exec(`
				INSERT INTO sessions (place, action, started_at, last_seen_at)
				VALUES (?, ?, ?, ?)
			`, r.Place, r.Action, now, now)
			logErr("insert session", err)
		} else {
			_, err := t.db.Exec(`
				UPDATE sessions SET last_seen_at = ?
				WHERE place = ? AND action = ? AND ended_at IS NULL
			`, now, r.Place, r.Action)
			logErr("update last_seen", err)
		}
	}

	// Sessions that ended.
	for k := range t.previous {
		if !current[k] {
			place, action, ok := strings.Cut(k, ":")
			if !ok {
				continue
			}
			_, err := t.db.Exec(`
				UPDATE sessions SET ended_at = ?, last_seen_at = ?
				WHERE place = ? AND action = ? AND ended_at IS NULL
			`, now, now, place, action)
			logErr("end session", err)
		}
	}

	t.previous = current
}

// CloseAll ends all open sessions (called on graceful shutdown).
func (t *Tracker) CloseAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now().Unix()
	_, err := t.db.Exec(`
		UPDATE sessions SET ended_at = ?, last_seen_at = ?
		WHERE ended_at IS NULL
	`, now, now)
	logErr("close all", err)
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
	err = t.db.QueryRow(`
		SELECT SUM(
			COALESCE(ended_at, ?) - started_at
		) FROM sessions
		WHERE place = ? AND action = ? AND started_at >= ?
	`, nowUnix, place, action, todayStart).Scan(&totalSeconds)
	if err != nil && err != sql.ErrNoRows {
		logErr("query today total", err)
	}

	todayTotal := 0
	if totalSeconds.Valid {
		todayTotal = int(totalSeconds.Int64)
	}

	return &ActiveInfo{
		Elapsed: elapsed,
		Today:   todayTotal,
	}
}

// Session represents a completed or active session for history queries.
type Session struct {
	Place     string `json:"place"`
	Action    string `json:"action"`
	StartedAt int64  `json:"started_at"`
	EndedAt   int64  `json:"ended_at"`  // 0 if still active
	Duration  int    `json:"duration"`  // seconds
}

// QueryHistory returns all sessions that overlap the given time range.
func (t *Tracker) QueryHistory(from, to int64) []Session {
	t.mu.Lock()
	defer t.mu.Unlock()

	rows, err := t.db.Query(`
		SELECT place, action, started_at, COALESCE(ended_at, ?), COALESCE(ended_at, ?) - started_at
		FROM sessions
		WHERE started_at < ? AND COALESCE(ended_at, ?) >= ?
		ORDER BY started_at ASC
	`, time.Now().Unix(), time.Now().Unix(), to, time.Now().Unix(), from)
	if err != nil {
		logErr("query history", err)
		return nil
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.Place, &s.Action, &s.StartedAt, &s.EndedAt, &s.Duration); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions
}
