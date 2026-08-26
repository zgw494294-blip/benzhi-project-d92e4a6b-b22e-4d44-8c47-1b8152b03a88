package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type digestableEvent struct {
	SchemaVersion  int                `json:"schemaVersion"`
	Sequence       uint64             `json:"sequence"`
	EventID        string             `json:"eventID"`
	EventType      string             `json:"eventType"`
	CaseID         string             `json:"caseID"`
	OccurredAt     string             `json:"occurredAt"`
	PreviousDigest string             `json:"previousDigest"`
	State          json.RawMessage    `json:"state"`
	Idempotency    *IdempotencyRecord `json:"idempotency,omitempty"`
}

func eventDigest(event Event) (string, error) {
	value := digestableEvent{
		SchemaVersion: event.SchemaVersion, Sequence: event.Sequence, EventID: event.EventID,
		EventType: event.EventType, CaseID: event.CaseID,
		OccurredAt:     event.OccurredAt.UTC().Format("2006-01-02T15:04:05.000000000Z"),
		PreviousDigest: event.PreviousDigest, State: event.State, Idempotency: event.Idempotency,
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func verifyEvent(event Event, sequence uint64, previous string) error {
	if event.SchemaVersion != SchemaVersion {
		return &IntegrityError{Sequence: event.Sequence, Reason: "schemaVersion 不受支持"}
	}
	if event.Sequence != sequence {
		return &IntegrityError{Sequence: event.Sequence, Reason: "事件序号不连续"}
	}
	if event.PreviousDigest != previous {
		return &IntegrityError{Sequence: event.Sequence, Reason: "前序摘要不匹配"}
	}
	digest, err := eventDigest(event)
	if err != nil {
		return err
	}
	if event.Digest != digest {
		return &IntegrityError{Sequence: event.Sequence, Reason: "事件摘要不匹配"}
	}
	if event.CaseID == "" || event.EventID == "" || event.EventType == "" || len(event.State) == 0 {
		return &IntegrityError{Sequence: event.Sequence, Reason: "事件必要字段缺失"}
	}
	return nil
}
