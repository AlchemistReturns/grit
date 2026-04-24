package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"
)

type Event struct {
	ID            string
	Hook          string
	OccurredAt    time.Time
	Skipped       bool
	CommitMsg     string
	RelatedCommit string
}

type Filter struct {
	Hook    string
	Since   time.Time
	Skipped *bool
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func InsertEvent(db *sql.DB, hook string, skipped bool, commitMsg string) (string, error) {
	return InsertEventFull(db, hook, skipped, commitMsg, "")
}

func InsertEventFull(db *sql.DB, hook string, skipped bool, commitMsg, relatedCommit string) (string, error) {
	id := newID()
	_, err := db.Exec(
		`INSERT INTO events (id, hook, occurred_at, skipped, commit_msg, related_commit) VALUES (?, ?, ?, ?, ?, ?)`,
		id, hook, time.Now().Unix(), btoi(skipped), commitMsg, relatedCommit,
	)
	return id, err
}

func QueryEvents(db *sql.DB, f Filter) ([]Event, error) {
	query := `SELECT id, hook, occurred_at, skipped, commit_msg, COALESCE(related_commit, '') FROM events WHERE 1=1`
	args := []any{}

	if f.Hook != "" {
		query += ` AND hook = ?`
		args = append(args, f.Hook)
	}
	if !f.Since.IsZero() {
		query += ` AND occurred_at >= ?`
		args = append(args, f.Since.Unix())
	}
	if f.Skipped != nil {
		query += ` AND skipped = ?`
		args = append(args, btoi(*f.Skipped))
	}
	query += ` ORDER BY occurred_at DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var ts int64
		var skip int
		var msg sql.NullString
		if err := rows.Scan(&e.ID, &e.Hook, &ts, &skip, &msg, &e.RelatedCommit); err != nil {
			return nil, err
		}
		e.OccurredAt = time.Unix(ts, 0)
		e.Skipped = skip != 0
		if msg.Valid {
			e.CommitMsg = msg.String
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// CountEvents returns total and skipped event counts since the given unix timestamp.
func CountEvents(db *sql.DB, since int64) (total, skipped int, err error) {
	var sk sql.NullInt64
	err = db.QueryRow(
		`SELECT COUNT(*), SUM(CASE WHEN skipped = 1 THEN 1 ELSE 0 END) FROM events WHERE occurred_at >= ?`,
		since,
	).Scan(&total, &sk)
	if sk.Valid {
		skipped = int(sk.Int64)
	}
	return
}

// EventsPerDay returns a map of "YYYY-MM-DD" -> event count since the given unix timestamp.
func EventsPerDay(db *sql.DB, since int64) (map[string]int, error) {
	rows, err := db.Query(`
		SELECT date(occurred_at, 'unixepoch', 'localtime') as day, COUNT(*)
		FROM events
		WHERE occurred_at >= ?
		GROUP BY day
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var day string
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return nil, err
		}
		result[day] = count
	}
	return result, rows.Err()
}

// StreakDays returns the number of consecutive days ending today with at least one completed interview.
func StreakDays(db *sql.DB) (int, error) {
	rows, err := db.Query(`
		SELECT DISTINCT date(occurred_at, 'unixepoch', 'localtime') as day
		FROM events
		WHERE hook = 'interview' AND skipped = 0
		ORDER BY day DESC
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var days []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return 0, err
		}
		days = append(days, d)
	}

	if len(days) == 0 {
		return 0, nil
	}

	today := time.Now().Format("2006-01-02")
	streak := 0
	current := today
	for _, d := range days {
		if d == current {
			streak++
			t, _ := time.Parse("2006-01-02", current)
			current = t.AddDate(0, 0, -1).Format("2006-01-02")
		} else {
			break
		}
	}
	return streak, nil
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
