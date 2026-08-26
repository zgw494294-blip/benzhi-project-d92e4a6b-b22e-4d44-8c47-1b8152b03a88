package certification

import (
	"time"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/assessment"
)

type CreateCaseCommand struct {
	CabinetCode    string `json:"cabinetCode"`
	Location       string `json:"location"`
	CabinetClass   string `json:"cabinetClass"`
	BaselineStatus string `json:"baselineStatus"`
	IdempotencyKey string `json:"-"`
}

type LockPlanCommand struct {
	ExpectedVersion   int64                    `json:"expectedVersion"`
	StandardCode      string                   `json:"standardCode"`
	Revision          string                   `json:"revision"`
	VelocityRange     assessment.VelocityRange `json:"velocityRange"`
	ParticleLimit     int64                    `json:"particleLimit"`
	IntegrityRequired bool                     `json:"integrityRequired"`
	PointOrder        []string                 `json:"pointOrder"`
	IdempotencyKey    string                   `json:"-"`
	Preflight         bool                     `json:"preflight,omitempty"`
}

type SubmitMeasurementCommand struct {
	ExpectedVersion        int64     `json:"expectedVersion"`
	PointID                string    `json:"pointID"`
	Velocity               float64   `json:"velocity"`
	ParticleCount          int64     `json:"particleCount"`
	IntegrityPassed        bool      `json:"integrityPassed"`
	EvidenceDigest         string    `json:"evidenceDigest"`
	MeasuredBy             string    `json:"measuredBy"`
	MeasuredAt             time.Time `json:"measuredAt"`
	Assignee               string    `json:"assignee"`
	DueAt                  time.Time `json:"dueAt"`
	IdempotencyKey         string    `json:"-"`
	CorrectsMeasurementID  string    `json:"correctsMeasurementID,omitempty"`
	CorrectedMeasurementID string    `json:"correctedMeasurementID,omitempty"`
	CorrectionReason       string    `json:"correctionReason,omitempty"`
}

type RemediateCommand struct {
	ExpectedVersion  int64     `json:"expectedVersion"`
	DeviationID      string    `json:"deviationID"`
	RemediationNote  string    `json:"remediationNote"`
	EvidenceDigest   string    `json:"evidenceDigest"`
	Actor            string    `json:"actor"`
	IdempotencyKey   string    `json:"-"`
	Action           string    `json:"action,omitempty"`
	NewAssignee      string    `json:"newAssignee,omitempty"`
	NewDueAt         time.Time `json:"newDueAt,omitempty"`
	AdjustmentReason string    `json:"adjustmentReason,omitempty"`
	ExtensionReason  string    `json:"extensionReason,omitempty"`
	Operator         string    `json:"operator,omitempty"`
	Assignee         string    `json:"assignee,omitempty"`
	DueAt            time.Time `json:"dueAt,omitempty"`
}

type RetestCommand struct {
	ExpectedVersion int64     `json:"expectedVersion"`
	DeviationID     string    `json:"deviationID"`
	Velocity        float64   `json:"velocity"`
	ParticleCount   int64     `json:"particleCount"`
	IntegrityPassed bool      `json:"integrityPassed"`
	EvidenceDigest  string    `json:"evidenceDigest"`
	MeasuredBy      string    `json:"measuredBy"`
	MeasuredAt      time.Time `json:"measuredAt"`
	IdempotencyKey  string    `json:"-"`
}

type ReviewCommand struct {
	ExpectedVersion int64         `json:"expectedVersion"`
	Decision        string        `json:"decision"`
	Reviewer        string        `json:"reviewer"`
	Comment         string        `json:"comment"`
	PointID         string        `json:"pointID"`
	Assignee        string        `json:"assignee"`
	DueAt           time.Time     `json:"dueAt"`
	IdempotencyKey  string        `json:"-"`
	ChecklistDigest string        `json:"checklistDigest,omitempty"`
	Issues          []ReviewIssue `json:"issues,omitempty"`
}

type CaseFilter struct {
	CabinetCode string
	Status      string
}

type PlanPreflightReport struct {
	CaseID               string                      `json:"caseID"`
	CaseVersion          int64                       `json:"caseVersion"`
	Valid                bool                        `json:"valid"`
	NormalizedPlan       assessment.Plan             `json:"normalizedPlan"`
	NormalizedPointOrder []string                    `json:"normalizedPointOrder"`
	PointCount           int                         `json:"pointCount"`
	ThresholdSummary     assessment.ThresholdSummary `json:"thresholdSummary"`
	PlanDigest           string                      `json:"planDigest,omitempty"`
	Issues               []assessment.PlanIssue      `json:"issues"`
}

type IssueCredentialCommand struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	IssuedBy        string `json:"issuedBy"`
	IdempotencyKey  string `json:"-"`
}
