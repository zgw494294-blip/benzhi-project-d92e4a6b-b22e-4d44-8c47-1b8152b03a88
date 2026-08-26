package assessment

import (
	"errors"
	"math"
	"strings"
	"time"
)

func ValidateReading(reading Reading, now time.Time) error {
	if strings.TrimSpace(reading.PointID) == "" {
		return errors.New("测点标识不能为空")
	}
	if math.IsNaN(reading.Velocity) || math.IsInf(reading.Velocity, 0) || reading.Velocity < 0 {
		return errors.New("风速读数无效")
	}
	if reading.ParticleCount < 0 {
		return errors.New("粒子计数不能为负数")
	}
	if strings.TrimSpace(reading.EvidenceDigest) == "" {
		return errors.New("证据摘要不能为空")
	}
	if strings.TrimSpace(reading.MeasuredBy) == "" {
		return errors.New("测量人员不能为空")
	}
	if reading.MeasuredAt.IsZero() {
		return errors.New("测量时间不能为空")
	}
	if reading.MeasuredAt.After(now.Add(5 * time.Minute)) {
		return errors.New("测量时间不能明显晚于当前时间")
	}
	return nil
}

func Evaluate(plan Plan, reading Reading) Result {
	reasons := make([]string, 0, 3)
	if reading.Velocity < plan.VelocityRange.Min || reading.Velocity > plan.VelocityRange.Max {
		reasons = append(reasons, "VELOCITY_OUT_OF_RANGE")
	}
	if reading.ParticleCount > plan.ParticleLimit {
		reasons = append(reasons, "PARTICLE_LIMIT_EXCEEDED")
	}
	if plan.IntegrityRequired && !reading.IntegrityPassed {
		reasons = append(reasons, "INTEGRITY_FAILED")
	}
	if len(reasons) > 0 {
		return Result{Outcome: "failed", ReasonCodes: reasons}
	}
	return Result{Outcome: "passed"}
}
