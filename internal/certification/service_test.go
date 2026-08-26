package certification

import (
	"testing"
	"time"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/assessment"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/credential"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/ledger"
)

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	store, recovery, err := ledger.Open(t.TempDir(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(store, recovery, credential.NewIssuer(time.Now), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateCase(CreateCaseCommand{CabinetCode: "CAB-1", Location: "L1", CabinetClass: "II", BaselineStatus: "qualified", IdempotencyKey: "create-key-001"})
	if err != nil {
		t.Fatal(err)
	}
	return service, created.CaseID
}

func TestFullWorkflowWithDeviationAndCredential(t *testing.T) {
	service, caseID := newTestService(t)
	value, err := service.LockPlan(caseID, LockPlanCommand{
		ExpectedVersion: 1, StandardCode: "JG/T 292", Revision: "2024", VelocityRange: assessment.VelocityRange{Min: 0.3, Max: 0.6},
		ParticleLimit: 100, IntegrityRequired: true, PointOrder: []string{"P1", "P2"}, IdempotencyKey: "plan-key-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	failed := SubmitMeasurementCommand{
		ExpectedVersion: value.Version, PointID: "P1", Velocity: 0.2, ParticleCount: 10, IntegrityPassed: true,
		EvidenceDigest: "sha256:failed-reading", MeasuredBy: "工程师", MeasuredAt: time.Now(), Assignee: "责任人", DueAt: time.Now().Add(time.Hour), IdempotencyKey: "measure-key-001",
	}
	value, err = service.SubmitMeasurement(caseID, failed)
	if err != nil {
		t.Fatal(err)
	}
	if value.Status != StatusRemediation || len(value.Deviations) != 1 {
		t.Fatalf("未进入整改: %+v", value)
	}
	deviationID := value.Deviations[0].DeviationID
	value, err = service.Remediate(caseID, RemediateCommand{ExpectedVersion: value.Version, DeviationID: deviationID, RemediationNote: "校准风机", EvidenceDigest: "sha256:remediation", Actor: "工程师", IdempotencyKey: "remediate-key-001"})
	if err != nil {
		t.Fatal(err)
	}
	value, err = service.Retest(caseID, RetestCommand{ExpectedVersion: value.Version, DeviationID: deviationID, Velocity: 0.4, ParticleCount: 10, IntegrityPassed: true, EvidenceDigest: "sha256:retest-reading", MeasuredBy: "工程师", MeasuredAt: time.Now(), IdempotencyKey: "retest-key-001"})
	if err != nil {
		t.Fatal(err)
	}
	value, err = service.SubmitMeasurement(caseID, SubmitMeasurementCommand{ExpectedVersion: value.Version, PointID: "P2", Velocity: 0.4, ParticleCount: 10, IntegrityPassed: true, EvidenceDigest: "sha256:second-reading", MeasuredBy: "工程师", MeasuredAt: time.Now(), IdempotencyKey: "measure-key-002"})
	if err != nil {
		t.Fatal(err)
	}
	value, err = service.Review(caseID, ReviewCommand{ExpectedVersion: value.Version, Decision: "approve", Reviewer: "质量员", Comment: "资料完整，同意冻结", IdempotencyKey: "review-key-001"})
	if err != nil || value.Status != StatusFrozen {
		t.Fatalf("冻结失败: %+v %v", value, err)
	}
	issued, err := service.IssueCredential(caseID, IssueCredentialCommand{ExpectedVersion: value.Version, IssuedBy: "质量员", IdempotencyKey: "issue-key-001"})
	if err != nil {
		t.Fatal(err)
	}
	verified, err := service.VerifyCredential(issued.CredentialID, issued.VerificationCode)
	if err != nil || !verified.Valid {
		t.Fatalf("凭据校验失败: %+v %v", verified, err)
	}
}

func TestVersionOrderAndIdempotencyProtection(t *testing.T) {
	service, caseID := newTestService(t)
	command := LockPlanCommand{ExpectedVersion: 1, StandardCode: "S", Revision: "R", VelocityRange: assessment.VelocityRange{Min: 0.3, Max: 0.6}, ParticleLimit: 1, PointOrder: []string{"P1", "P2"}, IdempotencyKey: "same-key-001"}
	first, err := service.LockPlan(caseID, command)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := service.LockPlan(caseID, command)
	if err != nil || replayed.Version != first.Version {
		t.Fatalf("幂等重放失败: %+v %v", replayed, err)
	}
	command.Revision = "different"
	if _, err := service.LockPlan(caseID, command); ErrorCodeOf(err) != CodeIdempotency {
		t.Fatalf("幂等键冲突未识别: %v", err)
	}
	_, err = service.SubmitMeasurement(caseID, SubmitMeasurementCommand{ExpectedVersion: first.Version - 1, PointID: "P1", EvidenceDigest: "sha256:evidence", MeasuredBy: "工程师", MeasuredAt: time.Now(), IdempotencyKey: "stale-key-001"})
	if ErrorCodeOf(err) != CodeStaleVersion {
		t.Fatalf("陈旧版本未拒绝: %v", err)
	}
	_, err = service.SubmitMeasurement(caseID, SubmitMeasurementCommand{ExpectedVersion: first.Version, PointID: "P2", Velocity: 0.4, EvidenceDigest: "sha256:evidence", MeasuredBy: "工程师", MeasuredAt: time.Now(), IdempotencyKey: "order-key-001"})
	if ErrorCodeOf(err) != CodeConflict {
		t.Fatalf("越序测点未拒绝: %v", err)
	}
}
