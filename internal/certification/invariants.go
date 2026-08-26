package certification

import (
	"fmt"
	"strings"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/assessment"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/credential"
)

func validateAggregate(value *CertificationCase) error {
	if value == nil || value.CaseID == "" || value.Version <= 0 || value.CreatedAt.IsZero() {
		return fmt.Errorf("案例基础投影不完整")
	}
	if value.NormalizedCabinetCode == "" || value.CabinetCode != value.NormalizedCabinetCode || NormalizeCabinetCode(value.CabinetCode) != value.NormalizedCabinetCode {
		return fmt.Errorf("案例工作台规范化编号无效")
	}
	validStatus := map[string]bool{
		StatusDraft: true, StatusAwaitingTest: true, StatusTesting: true,
		StatusRemediation: true, StatusAwaitingReview: true, StatusFrozen: true, StatusReleased: true,
	}
	if !validStatus[value.Status] {
		return fmt.Errorf("案例状态 %s 无效", value.Status)
	}
	if value.Status == StatusDraft {
		if value.Plan != nil || len(value.Measurements) != 0 || len(value.Deviations) != 0 {
			return fmt.Errorf("草稿案例不能包含锁定方案、测量或偏差")
		}
		return nil
	}
	if value.Plan == nil || value.Plan.CaseID != value.CaseID || value.Plan.PlanID == "" || value.Plan.LockedAt.IsZero() {
		return fmt.Errorf("非草稿案例必须包含有效锁定方案")
	}
	plan := toAssessmentPlan(value.Plan)
	digest, err := assessment.PlanDigest(plan)
	if err != nil {
		return fmt.Errorf("锁定方案无效: %w", err)
	}
	if digest != value.Plan.Digest {
		return fmt.Errorf("锁定方案摘要不一致")
	}
	attempts := make(map[string]int)
	initialRoots := make(map[string]bool)
	activeInitials := make(map[string]bool)
	measurementsByID := make(map[string]Measurement)
	for _, measurement := range value.Measurements {
		if measurement.CaseID != value.CaseID || measurement.MeasurementID == "" || measurementsByID[measurement.MeasurementID].MeasurementID != "" {
			return fmt.Errorf("测量记录归属或标识无效")
		}
		if !assessment.ContainsPoint(value.Plan.PointOrder, measurement.PointID) {
			return fmt.Errorf("测量记录包含方案外测点 %s", measurement.PointID)
		}
		attempts[measurement.PointID]++
		if measurement.Attempt != attempts[measurement.PointID] {
			return fmt.Errorf("测点 %s 的测量次数不连续", measurement.PointID)
		}
		if measurement.Outcome != "passed" && measurement.Outcome != "failed" {
			return fmt.Errorf("测点 %s 的测量结论无效", measurement.PointID)
		}
		if measurement.Retest {
			if measurement.CorrectsMeasurementID != "" || measurement.SupersededByMeasurementID != "" || measurement.CorrectionReason != "" {
				return fmt.Errorf("复测记录不能带有初测纠错关系")
			}
		} else if measurement.CorrectsMeasurementID == "" {
			if initialRoots[measurement.PointID] {
				return fmt.Errorf("测点 %s 存在多个初测根记录", measurement.PointID)
			}
			initialRoots[measurement.PointID] = true
		} else {
			original, ok := measurementsByID[measurement.CorrectsMeasurementID]
			if !ok || original.Retest || original.PointID != measurement.PointID || original.SupersededByMeasurementID != measurement.MeasurementID {
				return fmt.Errorf("测量 %s 的纠错引用无效", measurement.MeasurementID)
			}
			if strings.TrimSpace(measurement.CorrectionReason) == "" {
				return fmt.Errorf("测量 %s 缺少更正原因", measurement.MeasurementID)
			}
		}
		if !measurement.Retest && measurement.SupersededByMeasurementID == "" {
			if activeInitials[measurement.PointID] {
				return fmt.Errorf("测点 %s 存在多个有效初测", measurement.PointID)
			}
			activeInitials[measurement.PointID] = true
		}
		measurementsByID[measurement.MeasurementID] = measurement
	}
	for _, measurement := range value.Measurements {
		if measurement.SupersededByMeasurementID == "" {
			continue
		}
		replacement, ok := measurementsByID[measurement.SupersededByMeasurementID]
		if !ok || replacement.CorrectsMeasurementID != measurement.MeasurementID {
			return fmt.Errorf("测量 %s 的替代引用无效", measurement.MeasurementID)
		}
	}
	deviationIDs := make(map[string]bool)
	for _, deviation := range value.Deviations {
		if deviation.CaseID != value.CaseID || deviation.DeviationID == "" || deviationIDs[deviation.DeviationID] {
			return fmt.Errorf("偏差记录归属、标识或唯一性无效")
		}
		deviationIDs[deviation.DeviationID] = true
		if !assessment.ContainsPoint(value.Plan.PointOrder, deviation.PointID) {
			return fmt.Errorf("偏差关联方案外测点 %s", deviation.PointID)
		}
		if deviation.Status != "open" && deviation.Status != "ready_for_retest" && deviation.Status != "closed" {
			return fmt.Errorf("偏差 %s 状态无效", deviation.DeviationID)
		}
		if deviation.Status == "closed" && deviation.ClosedAt == nil {
			return fmt.Errorf("已关闭偏差 %s 缺少关闭时间", deviation.DeviationID)
		}
		for _, change := range deviation.AssignmentHistory {
			if strings.TrimSpace(change.PreviousAssignee) == "" || strings.TrimSpace(change.NewAssignee) == "" || strings.TrimSpace(change.Operator) == "" || strings.TrimSpace(change.Reason) == "" || change.PreviousDueAt.IsZero() || change.NewDueAt.IsZero() || change.OccurredAt.IsZero() {
				return fmt.Errorf("偏差 %s 的责任调整历史不完整", deviation.DeviationID)
			}
		}
	}
	if value.Status == StatusFrozen || value.Status == StatusReleased {
		if value.FrozenAt == nil || value.FrozenSnapshot == nil {
			return fmt.Errorf("冻结或放行案例缺少冻结信息")
		}
		if value.FrozenSnapshot.CaseID != value.CaseID || value.FrozenSnapshot.CaseVersion <= 0 {
			return fmt.Errorf("冻结快照案例信息无效")
		}
		if _, err := credential.SnapshotDigest(*value.FrozenSnapshot); err != nil {
			return fmt.Errorf("冻结快照无效: %w", err)
		}
	}
	if value.Status == StatusReleased && value.Credential == nil {
		return fmt.Errorf("已放行案例缺少启用凭据")
	}
	if value.Status != StatusReleased && value.Credential != nil {
		return fmt.Errorf("未放行案例不能包含启用凭据")
	}
	return validateStatusSemantics(value)
}
