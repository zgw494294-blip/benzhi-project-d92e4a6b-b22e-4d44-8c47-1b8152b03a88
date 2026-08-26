package ledger

import (
	"encoding/json"
	"time"
)

const SchemaVersion = 1

type Event struct {
	SchemaVersion  int                `json:"schemaVersion"`
	Sequence       uint64             `json:"sequence"`
	EventID        string             `json:"eventID"`
	EventType      string             `json:"eventType"`
	CaseID         string             `json:"caseID"`
	OccurredAt     time.Time          `json:"occurredAt"`
	PreviousDigest string             `json:"previousDigest"`
	State          json.RawMessage    `json:"state"`
	Idempotency    *IdempotencyRecord `json:"idempotency,omitempty"`
	Digest         string             `json:"digest"`
}

type IdempotencyRecord struct {
	Scope         string          `json:"scope"`
	Key           string          `json:"key"`
	RequestDigest string          `json:"requestDigest"`
	StatusCode    int             `json:"statusCode"`
	Response      json.RawMessage `json:"response"`
	RecordedAt    time.Time       `json:"recordedAt"`
}

type Projection struct {
	SchemaVersion int                          `json:"schemaVersion"`
	LastSequence  uint64                       `json:"lastSequence"`
	LastDigest    string                       `json:"lastDigest"`
	Cases         map[string]json.RawMessage   `json:"cases"`
	Idempotency   map[string]IdempotencyRecord `json:"idempotency"`
}

type Commit struct {
	EventType   string
	CaseID      string
	State       any
	Idempotency *IdempotencyRecord
}

type Recovery struct {
	LastSequence uint64
	LastDigest   string
	Cases        map[string]json.RawMessage
	Idempotency  map[string]IdempotencyRecord
}
