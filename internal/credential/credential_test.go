package credential

import (
	"testing"
	"time"
)

func TestIssueAndVerifyFrozenSnapshot(t *testing.T) {
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	snapshot := FrozenSnapshot{
		CaseID: "case-1", CaseVersion: 7, CabinetCode: "CAB-1", Location: "L1", CabinetClass: "II",
		BaselineStatus: "qualified", PlanDigest: "plan-digest", LatestOutcomes: map[string]string{"P2": "passed", "P1": "passed"}, FrozenAt: now,
	}
	digest, err := SnapshotDigest(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	reordered := snapshot
	reordered.LatestOutcomes = map[string]string{"P1": "passed", "P2": "passed"}
	second, err := SnapshotDigest(reordered)
	if err != nil || digest != second {
		t.Fatalf("映射顺序影响规范化摘要: %q %q", digest, second)
	}
	issued, err := NewIssuer(func() time.Time { return now.Add(time.Minute) }).Issue(snapshot, "复核员")
	if err != nil {
		t.Fatal(err)
	}
	if result := Verify(issued, snapshot, issued.VerificationCode); !result.Valid {
		t.Fatalf("有效凭据校验失败: %+v", result)
	}
	tampered := snapshot
	tampered.Location = "L2"
	if result := Verify(issued, tampered, issued.VerificationCode); result.Valid {
		t.Fatal("篡改快照仍通过校验")
	}
	if result := Verify(issued, snapshot, "wrong-code"); result.Valid {
		t.Fatal("错误验证代码仍通过校验")
	}
}
