package assessment

import "time"

type VelocityRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type Plan struct {
	StandardCode      string        `json:"standardCode"`
	Revision          string        `json:"revision"`
	VelocityRange     VelocityRange `json:"velocityRange"`
	ParticleLimit     int64         `json:"particleLimit"`
	IntegrityRequired bool          `json:"integrityRequired"`
	PointOrder        []string      `json:"pointOrder"`
}

type Reading struct {
	PointID         string    `json:"pointID"`
	Velocity        float64   `json:"velocity"`
	ParticleCount   int64     `json:"particleCount"`
	IntegrityPassed bool      `json:"integrityPassed"`
	EvidenceDigest  string    `json:"evidenceDigest"`
	MeasuredBy      string    `json:"measuredBy"`
	MeasuredAt      time.Time `json:"measuredAt"`
}

type Result struct {
	Outcome     string   `json:"outcome"`
	ReasonCodes []string `json:"reasonCodes,omitempty"`
}

type DeviationState struct {
	PointID string
	Status  string
}

type PlanIssue struct {
	FieldPath string `json:"fieldPath"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

type ThresholdSummary struct {
	VelocityMin       float64 `json:"velocityMin"`
	VelocityMax       float64 `json:"velocityMax"`
	ParticleLimit     int64   `json:"particleLimit"`
	IntegrityRequired bool    `json:"integrityRequired"`
}

type PlanAssessment struct {
	Valid            bool             `json:"valid"`
	NormalizedPlan   Plan             `json:"normalizedPlan"`
	PointCount       int              `json:"pointCount"`
	ThresholdSummary ThresholdSummary `json:"thresholdSummary"`
	PlanDigest       string           `json:"planDigest,omitempty"`
	Issues           []PlanIssue      `json:"issues"`
}

type ReadinessMeasurement struct {
	MeasurementID  string
	PointID        string
	Outcome        string
	EvidenceDigest string
}

type ReviewChecklistItem struct {
	PointID                string   `json:"pointID"`
	LatestMeasurementID    string   `json:"latestMeasurementID,omitempty"`
	Outcome                string   `json:"outcome,omitempty"`
	EvidenceDigest         string   `json:"evidenceDigest,omitempty"`
	DeviationClosureStatus string   `json:"deviationClosureStatus"`
	BlockingReasons        []string `json:"blockingReasons"`
}

type ReviewChecklist struct {
	CaseVersion int64                 `json:"caseVersion"`
	Items       []ReviewChecklistItem `json:"items"`
	Blockers    []string              `json:"blockers"`
	Digest      string                `json:"digest"`
}
