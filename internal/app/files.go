package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/cionz0/hdsort/internal/domain"
)

// AttachFiles copies files onto the session under opaque ids.
// The ids are the only handle a later preview request may use.
func AttachFiles(session domain.Session, files []domain.FileRef) (domain.Session, error) {
	stored := make([]domain.SessionFile, len(files))
	seen := make(map[string]struct{}, len(files))
	for i, file := range files {
		id, err := newFileID()
		if err != nil {
			return domain.Session{}, err
		}
		if _, ok := seen[id]; ok {
			return domain.Session{}, fmt.Errorf("scan: duplicate file id")
		}
		seen[id] = struct{}{}
		stored[i] = domain.SessionFile{ID: id, File: file}
	}
	session.Files = stored
	return session, nil
}

func newFileID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
