package ledger

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Store struct {
	mu             sync.Mutex
	directory      string
	journalPath    string
	projectionPath string
	now            func() time.Time
	projection     Projection
}

func Open(directory string, now func() time.Time) (*Store, Recovery, error) {
	if strings.TrimSpace(directory) == "" {
		return nil, Recovery{}, errors.New("账本目录不能为空")
	}
	if now == nil {
		now = time.Now
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return nil, Recovery{}, fmt.Errorf("创建账本目录: %w", err)
	}
	store := &Store{
		directory: directory, journalPath: filepath.Join(directory, "events.jsonl"),
		projectionPath: filepath.Join(directory, "projection.json"), now: now,
	}
	recovery, err := recoverJournal(store.journalPath)
	if err != nil {
		return nil, Recovery{}, err
	}
	store.projection = Projection{
		SchemaVersion: SchemaVersion, LastSequence: recovery.LastSequence, LastDigest: recovery.LastDigest,
		Cases: recovery.Cases, Idempotency: recovery.Idempotency,
	}
	if err := store.compareOrRepairProjection(); err != nil {
		return nil, Recovery{}, err
	}
	return store, recovery, nil
}

func (s *Store) Commit(value Commit) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if value.CaseID == "" || value.EventType == "" || value.State == nil {
		return Event{}, errors.New("提交事件字段不完整")
	}
	state, err := json.Marshal(value.State)
	if err != nil {
		return Event{}, fmt.Errorf("编码聚合投影: %w", err)
	}
	eventID, err := randomID()
	if err != nil {
		return Event{}, err
	}
	event := Event{
		SchemaVersion: SchemaVersion, Sequence: s.projection.LastSequence + 1,
		EventID: eventID, EventType: value.EventType, CaseID: value.CaseID,
		OccurredAt: s.now().UTC(), PreviousDigest: s.projection.LastDigest,
		State: state, Idempotency: value.Idempotency,
	}
	event.Digest, err = eventDigest(event)
	if err != nil {
		return Event{}, err
	}
	line, err := json.Marshal(event)
	if err != nil {
		return Event{}, err
	}
	if err := appendAndSync(s.journalPath, line); err != nil {
		return Event{}, err
	}
	s.projection.LastSequence = event.Sequence
	s.projection.LastDigest = event.Digest
	s.projection.Cases[event.CaseID] = append(json.RawMessage(nil), state...)
	if event.Idempotency != nil {
		s.projection.Idempotency[idempotencyMapKey(event.Idempotency.Scope, event.Idempotency.Key)] = cloneIdempotency(*event.Idempotency)
	}
	if err := writeProjectionAtomic(s.directory, s.projectionPath, s.projection); err != nil {
		return event, fmt.Errorf("事件已写入但投影保存失败: %w", err)
	}
	return event, nil
}

func appendAndSync(path string, line []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return fmt.Errorf("打开事件账本: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("追加事件账本: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("同步事件账本: %w", err)
	}
	return nil
}

func randomID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "evt_" + hex.EncodeToString(buffer), nil
}
