package certification

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/assessment"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/credential"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/ledger"
)

func extensionService(t *testing.T, now func() time.Time) *Service {
	t.Helper()
	store, recovery, err := ledger.Open(t.TempDir(), now)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(store, recovery, credential.NewIssuer(now), now)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func createExtensionCase(t *testing.T, service *Service, key, cabinet string) *CertificationCase {
	t.Helper()
	value, err := service.CreateCase(CreateCaseCommand{CabinetCode: cabinet, Location: "一层", CabinetClass: "II", BaselineStatus: "qualified", IdempotencyKey: key})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func lockExtensionPlan(t *testing.T, service *Service, value *CertificationCase, points []string, key string) *CertificationCase {
	t.Helper()
	locked, err := service.LockPlan(value.CaseID, LockPlanCommand{
		ExpectedVersion: value.Version, StandardCode: "规范 A", Revision: "2026", VelocityRange: assessment.VelocityRange{Min: 0.3, Max: 0.6},
		ParticleLimit: 100, IntegrityRequired: true, PointOrder: points, IdempotencyKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	return locked
}

func passingMeasurement(t *testing.T, service *Service, value *CertificationCase, point, key string, now time.Time) *CertificationCase {
	t.Helper()
	measured, err := service.SubmitMeasurement(value.CaseID, SubmitMeasurementCommand{
		ExpectedVersion: value.Version, PointID: point, Velocity: 0.4, ParticleCount: 10, IntegrityPassed: true,
		EvidenceDigest: "sha256:passing-reading", MeasuredBy: "工程师", MeasuredAt: now, IdempotencyKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	return measured
}

func TestDeviceConflictRecoveryAndRecertification(t *testing.T) {
	now := time.Now().UTC()
	service := extensionService(t, func() time.Time { return now })
	first := createExtensionCase(t, service, "device-create-001", " cab-  01 ")
	if first.CabinetCode != "CAB- 01" {
		t.Fatalf("编号未规范化: %q", first.CabinetCode)
	}
	var wait sync.WaitGroup
	errorsFound := make(chan error, 2)
	for _, key := range []string{"device-create-002", "device-create-003"} {
		wait.Add(1)
		go func(key string) {
			defer wait.Done()
			_, err := service.CreateCase(CreateCaseCommand{CabinetCode: "CAB- 01", Location: "二层", CabinetClass: "II", BaselineStatus: "qualified", IdempotencyKey: key})
			errorsFound <- err
		}(key)
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if ErrorCodeOf(err) != CodeConflict {
			t.Fatalf("同设备在途案例未冲突: %v", err)
		}
	}
	locked := lockExtensionPlan(t, service, first, []string{"P1"}, "device-plan-001")
	measured := passingMeasurement(t, service, locked, "P1", "device-measure-001", now)
	detail, err := service.GetCase(first.CaseID)
	if err != nil || detail.ReviewReadiness == nil {
		t.Fatalf("缺少复核清单: %v", err)
	}
	frozen, err := service.Review(first.CaseID, ReviewCommand{ExpectedVersion: measured.Version, Decision: "approve", Reviewer: "质量员", Comment: "确认资料完整", ChecklistDigest: detail.ReviewReadiness.Digest, IdempotencyKey: "device-review-001"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.IssueCredential(first.CaseID, IssueCredentialCommand{ExpectedVersion: frozen.Version, IssuedBy: "质量员", IdempotencyKey: "device-issue-001"})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	second := createExtensionCase(t, service, "device-create-004", "cab- 01")
	if second.PreviousCaseID != first.CaseID || second.Status != StatusDraft {
		t.Fatalf("复认证关联错误: %+v", second)
	}
	listed, err := service.QueryCases(CaseFilter{CabinetCode: " CAB- 01 "})
	if err != nil || len(listed) != 2 || listed[0].CaseID != first.CaseID {
		t.Fatalf("设备筛选或排序错误: %+v %v", listed, err)
	}
	if _, err := service.QueryCases(CaseFilter{Status: "unknown"}); ErrorCodeOf(err) != CodeValidation {
		t.Fatalf("未知状态未拒绝: %v", err)
	}
}

func TestRecoveryRejectsDuplicateInFlightDevice(t *testing.T) {
	now := time.Now().UTC()
	store, _, err := ledger.Open(t.TempDir(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	makePayload := func(caseID string) json.RawMessage {
		payload, err := json.Marshal(CertificationCase{CaseID: caseID, CabinetCode: "CAB-X", NormalizedCabinetCode: "CAB-X", Location: "一层", CabinetClass: "II", BaselineStatus: "qualified", Status: StatusDraft, Version: 1, CreatedAt: now, Measurements: []Measurement{}, Deviations: []Deviation{}, ReviewHistory: []ReviewDecision{}})
		if err != nil {
			t.Fatal(err)
		}
		return payload
	}
	recovery := ledger.Recovery{Cases: map[string]json.RawMessage{"case_a": makePayload("case_a"), "case_b": makePayload("case_b")}}
	if _, err := NewService(store, recovery, credential.NewIssuer(func() time.Time { return now }), func() time.Time { return now }); err == nil {
		t.Fatal("恢复时未拒绝同设备多个在途案例")
	}
}

func TestPlanPreflightUsesFormalNormalization(t *testing.T) {
	now := time.Now().UTC()
	service := extensionService(t, func() time.Time { return now })
	value := createExtensionCase(t, service, "preflight-create-001", "CAB-P")
	invalid, err := service.PreflightPlan(value.CaseID, LockPlanCommand{StandardCode: "S", Revision: "R", VelocityRange: assessment.VelocityRange{Min: 0.6, Max: 0.3}, ParticleLimit: 0, PointOrder: []string{"P1", " P1 "}})
	if err != nil {
		t.Fatal(err)
	}
	if invalid.Valid || len(invalid.Issues) != 3 {
		t.Fatalf("预检未一次返回三个问题: %+v", invalid.Issues)
	}
	unchanged, _ := service.GetCase(value.CaseID)
	if unchanged.Version != 1 || unchanged.Status != StatusDraft {
		t.Fatalf("预检改变了案例: %+v", unchanged)
	}
	command := LockPlanCommand{ExpectedVersion: 1, StandardCode: " S ", Revision: " R ", VelocityRange: assessment.VelocityRange{Min: 0.3, Max: 0.6}, ParticleLimit: 10, PointOrder: []string{" P1 ", "P2"}}
	report, err := service.PreflightPlan(value.CaseID, command)
	if err != nil || !report.Valid {
		t.Fatalf("合法预检失败: %+v %v", report, err)
	}
	command.IdempotencyKey = "preflight-lock-001"
	locked, err := service.LockPlan(value.CaseID, command)
	if err != nil {
		t.Fatal(err)
	}
	if locked.Plan.Digest != report.PlanDigest || locked.Plan.PointOrder[0] != "P1" {
		t.Fatalf("预检与锁定结果不一致: %+v %+v", report, locked.Plan)
	}
}

func TestCorrectLatestInitialMeasurement(t *testing.T) {
	now := time.Now().UTC()
	service := extensionService(t, func() time.Time { return now })
	value := lockExtensionPlan(t, service, createExtensionCase(t, service, "correct-create-001", "CAB-C"), []string{"P1", "P2"}, "correct-plan-001")
	value = passingMeasurement(t, service, value, "P1", "correct-measure-001", now)
	originalID := value.Measurements[0].MeasurementID
	corrected, err := service.SubmitMeasurement(value.CaseID, SubmitMeasurementCommand{
		ExpectedVersion: value.Version, PointID: "P1", Velocity: 0.1, ParticleCount: 10, IntegrityPassed: true,
		EvidenceDigest: "sha256:corrected-reading", MeasuredBy: "工程师", MeasuredAt: now, Assignee: "责任人", DueAt: now.Add(24 * time.Hour),
		CorrectsMeasurementID: originalID, CorrectionReason: "人工抄录错误，重新核对", IdempotencyKey: "correct-measure-002",
	})
	if err != nil {
		t.Fatal(err)
	}
	if corrected.Status != StatusRemediation || len(corrected.Measurements) != 2 || len(corrected.Deviations) != 1 {
		t.Fatalf("纠错未生成完整审计链: %+v", corrected)
	}
	if corrected.Measurements[0].SupersededByMeasurementID != corrected.Measurements[1].MeasurementID || corrected.Measurements[1].CorrectsMeasurementID != originalID || corrected.Measurements[1].Attempt != 2 {
		t.Fatalf("纠错关系错误: %+v", corrected.Measurements)
	}
	replayed, err := service.SubmitMeasurement(value.CaseID, SubmitMeasurementCommand{
		ExpectedVersion: value.Version, PointID: "P1", Velocity: 0.1, ParticleCount: 10, IntegrityPassed: true,
		EvidenceDigest: "sha256:corrected-reading", MeasuredBy: "工程师", MeasuredAt: now, Assignee: "责任人", DueAt: now.Add(24 * time.Hour),
		CorrectsMeasurementID: originalID, CorrectionReason: "人工抄录错误，重新核对", IdempotencyKey: "correct-measure-002",
	})
	if err != nil || len(replayed.Measurements) != 2 {
		t.Fatalf("纠错幂等重放失败: %v", err)
	}
}

func TestDeviationAdjustmentDueStatusAndBatchReviewReturn(t *testing.T) {
	current := time.Now().UTC()
	service := extensionService(t, func() time.Time { return current })
	value := lockExtensionPlan(t, service, createExtensionCase(t, service, "adjust-create-001", "CAB-D"), []string{"P1"}, "adjust-plan-001")
	failed, err := service.SubmitMeasurement(value.CaseID, SubmitMeasurementCommand{ExpectedVersion: value.Version, PointID: "P1", Velocity: 0.1, ParticleCount: 10, IntegrityPassed: true, EvidenceDigest: "sha256:failed-reading", MeasuredBy: "工程师", MeasuredAt: current, Assignee: "甲", DueAt: current.Add(time.Hour), IdempotencyKey: "adjust-measure-001"})
	if err != nil {
		t.Fatal(err)
	}
	current = current.Add(2 * time.Hour)
	detail, err := service.GetCase(value.CaseID)
	if err != nil || detail.Deviations[0].DueStatus != "overdue" || detail.Deviations[0].OverdueDurationSeconds <= 0 || detail.Version != failed.Version {
		t.Fatalf("逾期派生状态错误: %+v %v", detail, err)
	}
	adjusted, err := service.AdjustDeviation(value.CaseID, RemediateCommand{ExpectedVersion: failed.Version, DeviationID: failed.Deviations[0].DeviationID, Action: "adjust_assignment", NewAssignee: "乙", NewDueAt: current.Add(24 * time.Hour), AdjustmentReason: "现场人员安排发生变化", ExtensionReason: "等待替代人员完成复测", Actor: "工程师", IdempotencyKey: "adjust-deviation-001"})
	if err != nil {
		t.Fatal(err)
	}
	if adjusted.Version != failed.Version+1 || len(adjusted.Deviations[0].AssignmentHistory) != 1 || adjusted.Deviations[0].Status != "open" {
		t.Fatalf("责任调整错误: %+v", adjusted.Deviations[0])
	}

	reviewService := extensionService(t, func() time.Time { return current })
	reviewCase := lockExtensionPlan(t, reviewService, createExtensionCase(t, reviewService, "batch-create-001", "CAB-R"), []string{"P1", "P2"}, "batch-plan-001")
	reviewCase = passingMeasurement(t, reviewService, reviewCase, "P1", "batch-measure-001", current)
	reviewCase = passingMeasurement(t, reviewService, reviewCase, "P2", "batch-measure-002", current)
	before, _ := reviewService.GetCase(reviewCase.CaseID)
	returned, err := reviewService.Review(reviewCase.CaseID, ReviewCommand{ExpectedVersion: reviewCase.Version, Decision: "return", Reviewer: "质量员", Comment: "两个测点需要补充确认", ChecklistDigest: before.ReviewReadiness.Digest, Issues: []ReviewIssue{{PointID: "P1", Reason: "证据照片需要重新拍摄", Assignee: "甲", DueAt: current.Add(24 * time.Hour)}, {PointID: "P2", Reason: "测点位置标识需要核对", Assignee: "乙", DueAt: current.Add(48 * time.Hour)}}, IdempotencyKey: "batch-review-001"})
	if err != nil {
		t.Fatal(err)
	}
	if returned.Status != StatusRemediation || returned.Version != reviewCase.Version+1 || len(returned.Deviations) != 2 || len(returned.ReviewHistory) != 1 || len(returned.ReviewHistory[0].Issues) != 2 {
		t.Fatalf("批量退回未原子生效: %+v", returned)
	}
}
