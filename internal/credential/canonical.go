package credential

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

type canonicalOutcome struct {
	PointID string `json:"pointID"`
	Outcome string `json:"outcome"`
}

type canonicalSnapshot struct {
	CaseID         string             `json:"caseID"`
	CaseVersion    int64              `json:"caseVersion"`
	CabinetCode    string             `json:"cabinetCode"`
	Location       string             `json:"location"`
	CabinetClass   string             `json:"cabinetClass"`
	BaselineStatus string             `json:"baselineStatus"`
	PlanDigest     string             `json:"planDigest"`
	Outcomes       []canonicalOutcome `json:"outcomes"`
	FrozenAt       string             `json:"frozenAt"`
}

func SnapshotDigest(snapshot FrozenSnapshot) (string, error) {
	if snapshot.CaseID == "" || snapshot.CaseVersion <= 0 || snapshot.FrozenAt.IsZero() {
		return "", errors.New("冻结快照字段不完整")
	}
	keys := make([]string, 0, len(snapshot.LatestOutcomes))
	for key := range snapshot.LatestOutcomes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	outcomes := make([]canonicalOutcome, 0, len(keys))
	for _, key := range keys {
		outcomes = append(outcomes, canonicalOutcome{PointID: key, Outcome: snapshot.LatestOutcomes[key]})
	}
	value := canonicalSnapshot{
		CaseID: snapshot.CaseID, CaseVersion: snapshot.CaseVersion, CabinetCode: snapshot.CabinetCode,
		Location: snapshot.Location, CabinetClass: snapshot.CabinetClass, BaselineStatus: snapshot.BaselineStatus,
		PlanDigest: snapshot.PlanDigest, Outcomes: outcomes, FrozenAt: snapshot.FrozenAt.UTC().Format(timeLayout),
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

const timeLayout = "2006-01-02T15:04:05.000000000Z"
