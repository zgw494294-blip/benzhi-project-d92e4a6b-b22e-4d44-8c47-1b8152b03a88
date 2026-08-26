package ledger

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCommitRecoverAndIdempotency(t *testing.T) {
	directory := t.TempDir()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	store, recovery, err := Open(directory, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if recovery.LastSequence != 0 {
		t.Fatalf("新账本序号应为零: %d", recovery.LastSequence)
	}
	record := &IdempotencyRecord{Scope: "scope", Key: "request-1", RequestDigest: "digest", StatusCode: 200, Response: []byte(`{"ok":true}`), RecordedAt: now}
	event, err := store.Commit(Commit{EventType: "case.created", CaseID: "case-1", State: map[string]any{"caseID": "case-1", "version": 1}, Idempotency: record})
	if err != nil {
		t.Fatal(err)
	}
	if event.Sequence != 1 || event.Digest == "" {
		t.Fatalf("事件链字段无效: %+v", event)
	}
	_, recovered, err := Open(directory, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.LastSequence != 1 || len(recovered.Cases) != 1 || len(recovered.Idempotency) != 1 {
		t.Fatalf("恢复结果不完整: %+v", recovered)
	}
}

func TestTamperedJournalIsRejected(t *testing.T) {
	directory := t.TempDir()
	store, _, err := Open(directory, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Commit(Commit{EventType: "case.created", CaseID: "case-1", State: map[string]any{"version": 1}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "events.jsonl")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(payload, []byte("case.created"), []byte("case.changed"), 1)
	if bytes.Equal(payload, tampered) {
		t.Fatal("测试未能修改账本")
	}
	if err := os.WriteFile(path, tampered, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Open(directory, time.Now); err == nil {
		t.Fatal("被篡改账本不应恢复成功")
	}
}
