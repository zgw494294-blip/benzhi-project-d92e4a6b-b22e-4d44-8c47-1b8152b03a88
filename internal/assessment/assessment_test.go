package assessment

import (
	"testing"
	"time"
)

func TestPlanValidationAndDeterministicEvaluation(t *testing.T) {
	plan := Plan{
		StandardCode: "JG/T 292", Revision: "2024", VelocityRange: VelocityRange{Min: 0.3, Max: 0.6},
		ParticleLimit: 100, IntegrityRequired: true, PointOrder: []string{"P1", "P2"},
	}
	if err := ValidatePlan(plan); err != nil {
		t.Fatalf("有效方案被拒绝: %v", err)
	}
	first, err := PlanDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PlanDigest(plan)
	if err != nil || first != second {
		t.Fatalf("方案摘要不稳定: %q %q %v", first, second, err)
	}
	result := Evaluate(plan, Reading{Velocity: 0.2, ParticleCount: 101, IntegrityPassed: false})
	if result.Outcome != "failed" || len(result.ReasonCodes) != 3 {
		t.Fatalf("不合格原因不完整: %+v", result)
	}
	result = Evaluate(plan, Reading{Velocity: 0.45, ParticleCount: 50, IntegrityPassed: true})
	if result.Outcome != "passed" {
		t.Fatalf("合格读数被错误判定: %+v", result)
	}
}

func TestOrderAndReviewReadiness(t *testing.T) {
	if point, ok := ExpectedPoint([]string{"A", "B"}, 1); !ok || point != "B" {
		t.Fatalf("执行顺序错误: %q %v", point, ok)
	}
	if err := ReviewReadiness([]string{"A"}, map[string]string{"A": "passed"}, []DeviationState{{PointID: "A", Status: "open"}}); err == nil {
		t.Fatal("未闭环偏差不应允许复核")
	}
	reading := Reading{PointID: "A", Velocity: 0.4, EvidenceDigest: "sha256:evidence", MeasuredBy: "工程师", MeasuredAt: time.Now()}
	if err := ValidateReading(reading, time.Now()); err != nil {
		t.Fatalf("有效读数被拒绝: %v", err)
	}
}
