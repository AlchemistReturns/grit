package store

import (
	"database/sql"
)

type Answer struct {
	ID       string
	EventID  string
	Question string
	Answer   string
	Tag      string
}

func InsertAnswer(db *sql.DB, eventID, question, answer, tag string) error {
	_, err := db.Exec(
		`INSERT INTO answers (id, event_id, question, answer, tag) VALUES (?, ?, ?, ?, ?)`,
		newID(), eventID, question, answer, tag,
	)
	return err
}

func QueryAnswersForEvent(db *sql.DB, eventID string) ([]Answer, error) {
	rows, err := db.Query(
		`SELECT id, event_id, question, answer, tag FROM answers WHERE event_id = ? ORDER BY rowid`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var answers []Answer
	for rows.Next() {
		var a Answer
		if err := rows.Scan(&a.ID, &a.EventID, &a.Question, &a.Answer, &a.Tag); err != nil {
			return nil, err
		}
		answers = append(answers, a)
	}
	return answers, rows.Err()
}

func InsertComplexity(db *sql.DB, path string, score float64) error {
	_, err := db.Exec(
		`INSERT INTO complexity_history (id, path, score, recorded_at) VALUES (?, ?, ?, strftime('%s','now'))`,
		newID(), path, score,
	)
	return err
}

func AvgComplexity(db *sql.DB, path string) (float64, error) {
	var avg sql.NullFloat64
	err := db.QueryRow(`SELECT AVG(score) FROM complexity_history WHERE path = ?`, path).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if !avg.Valid {
		return 0, nil
	}
	return avg.Float64, nil
}

// ComplexityHistory returns the last n complexity scores for a file, most recent first.
func ComplexityHistory(db *sql.DB, path string, limit int) ([]float64, error) {
	rows, err := db.Query(
		`SELECT score FROM complexity_history WHERE path = ? ORDER BY recorded_at DESC LIMIT ?`,
		path, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var scores []float64
	for rows.Next() {
		var s float64
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	return scores, rows.Err()
}

// RecentQuestions returns distinct question texts from the last windowSize interview events.
func RecentQuestions(db *sql.DB, windowSize int) ([]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT a.question
		FROM answers a
		JOIN events e ON a.event_id = e.id
		WHERE e.hook = 'interview'
		  AND e.id IN (
		      SELECT id FROM events WHERE hook = 'interview'
		      ORDER BY occurred_at DESC LIMIT ?
		  )
	`, windowSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var questions []string
	for rows.Next() {
		var q string
		if err := rows.Scan(&q); err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}
	return questions, rows.Err()
}

// TagCounts returns a map of tag -> count for answers since the given unix timestamp.
func TagCounts(db *sql.DB, since int64) (map[string]int, error) {
	rows, err := db.Query(`
		SELECT a.tag, COUNT(*)
		FROM answers a
		JOIN events e ON a.event_id = e.id
		WHERE a.tag != '' AND e.occurred_at >= ?
		GROUP BY a.tag
		ORDER BY COUNT(*) DESC
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var tag string
		var count int
		if err := rows.Scan(&tag, &count); err != nil {
			return nil, err
		}
		counts[tag] = count
	}
	return counts, rows.Err()
}

// AnswersForPath returns friction answers whose event commit_msg contains the given path.
func AnswersForPath(db *sql.DB, path string) ([]Answer, error) {
	rows, err := db.Query(`
		SELECT a.id, a.event_id, a.question, a.answer, a.tag
		FROM answers a
		JOIN events e ON a.event_id = e.id
		WHERE e.commit_msg LIKE ?
		ORDER BY e.occurred_at DESC
		LIMIT 20
	`, "%"+path+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var answers []Answer
	for rows.Next() {
		var a Answer
		if err := rows.Scan(&a.ID, &a.EventID, &a.Question, &a.Answer, &a.Tag); err != nil {
			return nil, err
		}
		answers = append(answers, a)
	}
	return answers, rows.Err()
}
