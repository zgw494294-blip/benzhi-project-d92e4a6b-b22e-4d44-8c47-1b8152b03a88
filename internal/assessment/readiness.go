package assessment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

func ReviewReadiness(pointOrder []string, latestOutcomes map[string]string, deviations []DeviationState) error {
	for _, point := range pointOrder {
		outcome, ok := latestOutcomes[point]
		if !ok {
			return fmt.Errorf("测点 %s 尚未完成", point)
		}
		if outcome != "passed" {
			return fmt.Errorf("测点 %s 当前结论不是合格", point)
		}
	}
	for _, deviation := range deviations {
		if deviation.Status != "closed" {
			return fmt.Errorf("测点 %s 的偏差尚未闭环", deviation.PointID)
		}
	}
	return nil
}

func BuildReviewChecklist(caseVersion int64, pointOrder []string, measurements []ReadinessMeasurement, deviations []DeviationState) ReviewChecklist {
	latest := make(map[string]ReadinessMeasurement, len(pointOrder))
	for _, measurement := range measurements {
		latest[measurement.PointID] = measurement
	}
	deviationStatus := make(map[string]string, len(pointOrder))
	for _, deviation := range deviations {
		current := deviationStatus[deviation.PointID]
		if deviation.Status != "closed" {
			deviationStatus[deviation.PointID] = "open"
		} else if current == "" {
			deviationStatus[deviation.PointID] = "closed"
		}
	}
	checklist := ReviewChecklist{CaseVersion: caseVersion, Items: make([]ReviewChecklistItem, 0, len(pointOrder)), Blockers: []string{}}
	for _, point := range pointOrder {
		measurement := latest[point]
		status := deviationStatus[point]
		if status == "" {
			status = "none"
		}
		item := ReviewChecklistItem{
			PointID: point, LatestMeasurementID: measurement.MeasurementID, Outcome: measurement.Outcome,
			EvidenceDigest: measurement.EvidenceDigest, DeviationClosureStatus: status, BlockingReasons: []string{},
		}
		if measurement.MeasurementID == "" {
			item.BlockingReasons = append(item.BlockingReasons, "测点尚无有效测量")
		} else if measurement.Outcome != "passed" {
			item.BlockingReasons = append(item.BlockingReasons, "测点最新有效结论不是合格")
		}
		if status == "open" {
			item.BlockingReasons = append(item.BlockingReasons, "测点关联偏差尚未闭环")
		}
		for _, reason := range item.BlockingReasons {
			checklist.Blockers = append(checklist.Blockers, point+": "+reason)
		}
		checklist.Items = append(checklist.Items, item)
	}
	payload, _ := json.Marshal(struct {
		CaseVersion int64                 `json:"caseVersion"`
		Items       []ReviewChecklistItem `json:"items"`
	}{CaseVersion: checklist.CaseVersion, Items: checklist.Items})
	sum := sha256.Sum256(payload)
	checklist.Digest = hex.EncodeToString(sum[:])
	return checklist
}

func DeviationDueStatus(dueAt, now time.Time) (string, int64) {
	dueAt = dueAt.UTC()
	now = now.UTC()
	if dueAt.Before(now) {
		return "overdue", int64(now.Sub(dueAt) / time.Second)
	}
	if dueAt.Sub(now) <= 72*time.Hour {
		return "due_soon", 0
	}
	return "normal", 0
}

func OverallOutcome(pointOrder []string, latestOutcomes map[string]string) string {
	if len(pointOrder) == 0 {
		return "incomplete"
	}
	for _, point := range pointOrder {
		outcome, ok := latestOutcomes[point]
		if !ok {
			return "incomplete"
		}
		if outcome != "passed" {
			return "failed"
		}
	}
	return "passed"
}
