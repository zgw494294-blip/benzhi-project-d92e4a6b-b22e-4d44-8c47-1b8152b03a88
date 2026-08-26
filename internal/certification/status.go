package certification

import (
	"fmt"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/assessment"
)

var allowedTransitions = map[string]map[string]bool{
	StatusDraft: {
		StatusAwaitingTest: true,
	},
	StatusAwaitingTest: {
		StatusTesting: true, StatusRemediation: true, StatusAwaitingReview: true,
	},
	StatusTesting: {
		StatusTesting: true, StatusRemediation: true, StatusAwaitingReview: true,
	},
	StatusRemediation: {
		StatusRemediation: true, StatusTesting: true, StatusAwaitingReview: true,
	},
	StatusAwaitingReview: {
		StatusRemediation: true, StatusFrozen: true,
	},
	StatusFrozen: {
		StatusReleased: true,
	},
	StatusReleased: {},
}

func transition(value *CertificationCase, next string) error {
	if value == nil {
		return domainError(CodePersistence, "不能迁移空案例")
	}
	if !allowedTransitions[value.Status][next] {
		return domainError(CodeInvalidState, "不允许从 %s 迁移到 %s", value.Status, next)
	}
	value.Status = next
	return nil
}

func validateStatusSemantics(value *CertificationCase) error {
	if value.Status == StatusDraft {
		return nil
	}
	openDeviations := 0
	states := make([]assessment.DeviationState, 0, len(value.Deviations))
	for _, deviation := range value.Deviations {
		states = append(states, assessment.DeviationState{PointID: deviation.PointID, Status: deviation.Status})
		if deviation.Status != "closed" {
			openDeviations++
		}
	}
	initialCount := 0
	for _, measurement := range value.Measurements {
		if !measurement.Retest && measurement.SupersededByMeasurementID == "" {
			initialCount++
		}
	}
	switch value.Status {
	case StatusAwaitingTest:
		if initialCount != 0 || openDeviations != 0 {
			return fmt.Errorf("待测试案例不能已有测量或未闭环偏差")
		}
	case StatusTesting:
		if initialCount == 0 || initialCount >= len(value.Plan.PointOrder) || openDeviations != 0 {
			return fmt.Errorf("测试中案例的测点进度或偏差状态不一致")
		}
	case StatusRemediation:
		if openDeviations == 0 {
			return fmt.Errorf("整改中案例必须存在未闭环偏差")
		}
	case StatusAwaitingReview, StatusFrozen, StatusReleased:
		if err := assessment.ReviewReadiness(value.Plan.PointOrder, latestOutcomes(value), states); err != nil {
			return fmt.Errorf("案例状态 %s 不满足全点合格和偏差闭环条件: %w", value.Status, err)
		}
	}
	return nil
}
