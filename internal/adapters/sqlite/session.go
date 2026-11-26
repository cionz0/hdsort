package sqlite

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/cionz0/hdsort/internal/domain"
)

// Save stores the session and returns a new id.
func (s *Store) Save(session domain.Session) (string, error) {
	id, err := newID()
	if err != nil {
		return "", err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO sessions (id, root, status) VALUES (?, ?, ?)`,
		id, session.Root, string(session.Status()),
	); err != nil {
		return "", err
	}
	if err := insertExact(tx, id, session.Exact); err != nil {
		return "", err
	}
	if err := insertSimilar(tx, id, session.Similar); err != nil {
		return "", err
	}
	for i, decision := range session.Decisions {
		if _, err := tx.Exec(
			`INSERT INTO decisions (session_id, decision_no, path, choice) VALUES (?, ?, ?, ?)`,
			id, i, decision.Path, string(decision.Choice),
		); err != nil {
			return "", err
		}
	}
	for i, move := range session.Moves {
		if _, err := tx.Exec(
			`INSERT INTO moves (session_id, move_no, from_path, destination, confirmed) VALUES (?, ?, ?, ?, ?)`,
			id, i, move.From, move.Destination, boolInt(move.Confirmed),
		); err != nil {
			return "", err
		}
	}
	for i, skipped := range session.Skipped {
		if _, err := tx.Exec(
			`INSERT INTO skipped (session_id, skip_no, path, reason, target) VALUES (?, ?, ?, ?, ?)`,
			id, i, skipped.Path, string(skipped.Reason), skipped.Target,
		); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return id, nil
}

// Load rebuilds a session saved by Save.
func (s *Store) Load(id string) (domain.Session, error) {
	var root, status string
	err := s.db.QueryRow(`SELECT root, status FROM sessions WHERE id = ?`, id).Scan(&root, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Session{}, fmt.Errorf("sqlite: session %s: %w", id, err)
	}
	if err != nil {
		return domain.Session{}, err
	}
	session := domain.NewSession(root)
	session.Exact, err = loadExact(s.db, id)
	if err != nil {
		return domain.Session{}, err
	}
	session.Similar, err = loadSimilar(s.db, id)
	if err != nil {
		return domain.Session{}, err
	}
	session.Decisions, err = loadDecisions(s.db, id)
	if err != nil {
		return domain.Session{}, err
	}
	session.Moves, err = loadMoves(s.db, id)
	if err != nil {
		return domain.Session{}, err
	}
	session.Skipped, err = loadSkipped(s.db, id)
	if err != nil {
		return domain.Session{}, err
	}
	return domain.RestoreSession(session, domain.SessionStatus(status))
}

func newID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func insertExact(tx *sql.Tx, id string, groups []domain.ExactGroup) error {
	for groupNo, group := range groups {
		for memberNo, member := range group.Members() {
			if _, err := tx.Exec(`
INSERT INTO exact_members
  (session_id, group_no, member_no, path, size, mod_time, content_id, readable)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				id, groupNo, memberNo, member.Path, member.Size, member.ModTime.UnixNano(),
				string(member.ContentID), boolInt(member.Readable),
			); err != nil {
				return err
			}
		}
		rec, ok := group.Recommendation()
		if !ok {
			continue
		}
		if _, err := tx.Exec(`
INSERT INTO exact_recommendations (session_id, group_no, keeper, reason, text)
VALUES (?, ?, ?, ?, ?)`,
			id, groupNo, rec.Keeper, string(rec.Reason), rec.Text,
		); err != nil {
			return err
		}
		for trashNo, path := range rec.Trash {
			if _, err := tx.Exec(`
INSERT INTO exact_trash (session_id, group_no, trash_no, path)
VALUES (?, ?, ?, ?)`,
				id, groupNo, trashNo, path,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func insertSimilar(tx *sql.Tx, id string, groups []domain.SimilarGroup) error {
	for groupNo, group := range groups {
		for memberNo, member := range group.Members() {
			if _, err := tx.Exec(`
INSERT INTO similar_members
  (session_id, group_no, member_no, path, size, mod_time, content_id, readable)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				id, groupNo, memberNo, member.Path, member.Size, member.ModTime.UnixNano(),
				string(member.ContentID), boolInt(member.Readable),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func loadExact(db *sql.DB, id string) ([]domain.ExactGroup, error) {
	rows, err := db.Query(`
SELECT group_no, path, size, mod_time, content_id, readable
FROM exact_members
WHERE session_id = ?
ORDER BY group_no, member_no`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.ExactGroup
	var members []domain.FileRef
	current := -1
	flush := func() error {
		if len(members) == 0 {
			return nil
		}
		group, err := domain.NewExactGroup(members)
		if err != nil {
			return err
		}
		groups = append(groups, group)
		members = nil
		return nil
	}
	for rows.Next() {
		var groupNo int
		member, err := scanMember(rows, &groupNo)
		if err != nil {
			return nil, err
		}
		if current != -1 && groupNo != current {
			if err := flush(); err != nil {
				return nil, err
			}
		}
		current = groupNo
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := flush(); err != nil {
		return nil, err
	}

	for i := range groups {
		rec, ok, err := loadRecommendation(db, id, i)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		groups[i], err = groups[i].WithRecommendation(rec)
		if err != nil {
			return nil, err
		}
	}
	return groups, nil
}

func loadRecommendation(db *sql.DB, id string, groupNo int) (domain.Recommendation, bool, error) {
	var keeper, reason, text string
	err := db.QueryRow(`
SELECT keeper, reason, text
FROM exact_recommendations
WHERE session_id = ? AND group_no = ?`, id, groupNo).Scan(&keeper, &reason, &text)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Recommendation{}, false, nil
	}
	if err != nil {
		return domain.Recommendation{}, false, err
	}
	rows, err := db.Query(`
SELECT path FROM exact_trash
WHERE session_id = ? AND group_no = ?
ORDER BY trash_no`, id, groupNo)
	if err != nil {
		return domain.Recommendation{}, false, err
	}
	defer rows.Close()
	var trash []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return domain.Recommendation{}, false, err
		}
		trash = append(trash, path)
	}
	if err := rows.Err(); err != nil {
		return domain.Recommendation{}, false, err
	}
	rec, err := domain.NewRecommendation(keeper, trash, domain.ReasonCode(reason))
	if err != nil {
		return domain.Recommendation{}, false, err
	}
	rec.Text = text
	return rec, true, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMember(rows rowScanner, groupNo *int) (domain.FileRef, error) {
	var (
		member   domain.FileRef
		id       string
		mod      int64
		readable int
	)
	if err := rows.Scan(groupNo, &member.Path, &member.Size, &mod, &id, &readable); err != nil {
		return domain.FileRef{}, err
	}
	member.ModTime = time.Unix(0, mod).UTC()
	member.ContentID = domain.ContentID(id)
	member.Readable = readable != 0
	return member, nil
}

func loadSimilar(db *sql.DB, id string) ([]domain.SimilarGroup, error) {
	rows, err := db.Query(`
SELECT group_no, path, size, mod_time, content_id, readable
FROM similar_members
WHERE session_id = ?
ORDER BY group_no, member_no`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []domain.SimilarGroup
	var members []domain.FileRef
	current := -1
	flush := func() error {
		if len(members) == 0 {
			return nil
		}
		group, err := domain.NewSimilarGroup(members)
		if err != nil {
			return err
		}
		groups = append(groups, group)
		members = nil
		return nil
	}
	for rows.Next() {
		var groupNo int
		member, err := scanMember(rows, &groupNo)
		if err != nil {
			return nil, err
		}
		if current != -1 && groupNo != current {
			if err := flush(); err != nil {
				return nil, err
			}
		}
		current = groupNo
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return groups, nil
}

func loadDecisions(db *sql.DB, id string) ([]domain.Decision, error) {
	rows, err := db.Query(`
SELECT path, choice FROM decisions
WHERE session_id = ?
ORDER BY decision_no`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var decisions []domain.Decision
	for rows.Next() {
		var path, choice string
		if err := rows.Scan(&path, &choice); err != nil {
			return nil, err
		}
		decision, err := domain.NewDecision(path, domain.Choice(choice))
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, decision)
	}
	return decisions, rows.Err()
}

func loadMoves(db *sql.DB, id string) ([]domain.LayoutMove, error) {
	rows, err := db.Query(`
SELECT from_path, destination, confirmed FROM moves
WHERE session_id = ?
ORDER BY move_no`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var moves []domain.LayoutMove
	for rows.Next() {
		var (
			move      domain.LayoutMove
			confirmed int
		)
		if err := rows.Scan(&move.From, &move.Destination, &confirmed); err != nil {
			return nil, err
		}
		move.Confirmed = confirmed != 0
		moves = append(moves, move)
	}
	return moves, rows.Err()
}

func loadSkipped(db *sql.DB, id string) ([]domain.Skipped, error) {
	rows, err := db.Query(`
SELECT path, reason, target FROM skipped
WHERE session_id = ?
ORDER BY skip_no`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var skipped []domain.Skipped
	for rows.Next() {
		var item domain.Skipped
		var reason string
		if err := rows.Scan(&item.Path, &reason, &item.Target); err != nil {
			return nil, err
		}
		item.Reason = domain.SkipReason(reason)
		skipped = append(skipped, item)
	}
	return skipped, rows.Err()
}
