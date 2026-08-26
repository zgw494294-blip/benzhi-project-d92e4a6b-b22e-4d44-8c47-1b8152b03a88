package credential

import "time"

type FrozenSnapshot struct {
	CaseID         string            `json:"caseID"`
	CaseVersion    int64             `json:"caseVersion"`
	CabinetCode    string            `json:"cabinetCode"`
	Location       string            `json:"location"`
	CabinetClass   string            `json:"cabinetClass"`
	BaselineStatus string            `json:"baselineStatus"`
	PlanDigest     string            `json:"planDigest"`
	LatestOutcomes map[string]string `json:"latestOutcomes"`
	FrozenAt       time.Time         `json:"frozenAt"`
}

type ReleaseCredential struct {
	CredentialID     string    `json:"credentialID"`
	CaseID           string    `json:"caseID"`
	CaseVersion      int64     `json:"caseVersion"`
	SnapshotDigest   string    `json:"snapshotDigest"`
	IssuedAt         time.Time `json:"issuedAt"`
	IssuedBy         string    `json:"issuedBy"`
	VerificationCode string    `json:"verificationCode"`
}

type Verification struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason,omitempty"`
	CaseID string `json:"caseID,omitempty"`
	Digest string `json:"snapshotDigest,omitempty"`
}
