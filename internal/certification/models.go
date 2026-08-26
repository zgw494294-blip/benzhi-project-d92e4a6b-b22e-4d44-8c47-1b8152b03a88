package certification

import (
	"time"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/assessment"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/credential"
)

const (
	StatusDraft          = "draft"
	StatusAwaitingTest   = "awaiting_test"
	StatusTesting        = "testing"
	StatusRemediation    = "remediation"
	StatusAwaitingReview = "awaiting_review"
	StatusFrozen         = "frozen"
	StatusReleased       = "released"
)

type CertificationCase struct {
	CaseID                string                        `json:"caseID"`
	CabinetCode           string                        `json:"cabinetCode"`
	NormalizedCabinetCode string                        `json:"normalizedCabinetCode"`
	PreviousCaseID        string                        `json:"previousCaseID,omitempty"`
	Location              string                        `json:"location"`
	CabinetClass          string                        `json:"cabinetClass"`
	BaselineStatus        string                        `json:"baselineStatus"`
	Status                string                        `json:"status"`
	Version               int64                         `json:"version"`
	CreatedAt             time.Time                     `json:"createdAt"`
	FrozenAt              *time.Time                    `json:"frozenAt,omitempty"`
	Plan                  *CertificationPlan            `json:"plan,omitempty"`
	Measurements          []Measurement                 `json:"measurements"`
	Deviations            []Deviation                   `json:"deviations"`
	ReviewHistory         []ReviewDecision              `json:"reviewHistory"`
	Credential            *credential.ReleaseCredential `json:"credential,omitempty"`
	FrozenSnapshot        *credential.FrozenSnapshot    `json:"frozenSnapshot,omitempty"`
	ReviewReadiness       *assessment.ReviewChecklist   `json:"reviewReadiness,omitempty"`
}

type CertificationPlan struct {
	PlanID            string                   `json:"planID"`
	CaseID            string                   `json:"caseID"`
	StandardCode      string                   `json:"standardCode"`
	Revision          string                   `json:"revision"`
	VelocityRange     assessment.VelocityRange `json:"velocityRange"`
	ParticleLimit     int64                    `json:"particleLimit"`
	IntegrityRequired bool                     `json:"integrityRequired"`
	PointOrder        []string                 `json:"pointOrder"`
	Digest            string                   `json:"digest"`
	LockedAt          time.Time                `json:"lockedAt"`
}

type Measurement struct {
	MeasurementID             string    `json:"measurementID"`
	CaseID                    string    `json:"caseID"`
	PointID                   string    `json:"pointID"`
	Attempt                   int       `json:"attempt"`
	Velocity                  float64   `json:"velocity"`
	ParticleCount             int64     `json:"particleCount"`
	IntegrityPassed           bool      `json:"integrityPassed"`
	EvidenceDigest            string    `json:"evidenceDigest"`
	MeasuredBy                string    `json:"measuredBy"`
	MeasuredAt                time.Time `json:"measuredAt"`
	Outcome                   string    `json:"outcome"`
	ReasonCodes               []string  `json:"reasonCodes,omitempty"`
	Retest                    bool      `json:"retest"`
	CorrectsMeasurementID     string    `json:"correctsMeasurementID,omitempty"`
	SupersededByMeasurementID string    `json:"supersededByMeasurementID,omitempty"`
	CorrectionReason          string    `json:"correctionReason,omitempty"`
}

type Deviation struct {
	DeviationID            string                      `json:"deviationID"`
	CaseID                 string                      `json:"caseID"`
	PointID                string                      `json:"pointID"`
	ReasonCode             string                      `json:"reasonCode"`
	Assignee               string                      `json:"assignee"`
	DueAt                  time.Time                   `json:"dueAt"`
	RemediationNote        string                      `json:"remediationNote,omitempty"`
	EvidenceDigest         string                      `json:"evidenceDigest,omitempty"`
	Status                 string                      `json:"status"`
	OpenedAt               time.Time                   `json:"openedAt"`
	ClosedAt               *time.Time                  `json:"closedAt,omitempty"`
	Description            string                      `json:"description,omitempty"`
	AssignmentHistory      []DeviationAssignmentChange `json:"assignmentHistory"`
	DueStatus              string                      `json:"dueStatus,omitempty"`
	OverdueDurationSeconds int64                       `json:"overdueDurationSeconds,omitempty"`
}

type DeviationAssignmentChange struct {
	PreviousAssignee string    `json:"previousAssignee"`
	PreviousDueAt    time.Time `json:"previousDueAt"`
	NewAssignee      string    `json:"newAssignee"`
	NewDueAt         time.Time `json:"newDueAt"`
	Operator         string    `json:"operator"`
	Reason           string    `json:"reason"`
	ExtensionReason  string    `json:"extensionReason,omitempty"`
	OccurredAt       time.Time `json:"occurredAt"`
}

type ReviewIssue struct {
	PointID  string    `json:"pointID"`
	Reason   string    `json:"reason"`
	Assignee string    `json:"assignee"`
	DueAt    time.Time `json:"dueAt"`
}

type ReviewDecision struct {
	Decision  string        `json:"decision"`
	Reviewer  string        `json:"reviewer"`
	Comment   string        `json:"comment"`
	PointID   string        `json:"pointID,omitempty"`
	DecidedAt time.Time     `json:"decidedAt"`
	Issues    []ReviewIssue `json:"issues,omitempty"`
}
