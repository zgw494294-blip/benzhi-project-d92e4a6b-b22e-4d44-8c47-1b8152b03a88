package assessment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

func ValidatePlan(plan Plan) error {
	result := AssessPlan(plan)
	if !result.Valid {
		return &PlanValidationError{Issues: result.Issues}
	}
	return nil
}

type PlanValidationError struct {
	Issues []PlanIssue
}

func (e *PlanValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "认证方案无效"
	}
	return e.Issues[0].Message
}

func AssessPlan(plan Plan) PlanAssessment {
	normalized := plan
	normalized.StandardCode = strings.TrimSpace(plan.StandardCode)
	normalized.Revision = strings.TrimSpace(plan.Revision)
	normalized.PointOrder = make([]string, len(plan.PointOrder))
	for index, point := range plan.PointOrder {
		normalized.PointOrder[index] = strings.TrimSpace(point)
	}
	result := PlanAssessment{
		NormalizedPlan: normalized,
		PointCount:     len(normalized.PointOrder),
		ThresholdSummary: ThresholdSummary{
			VelocityMin: normalized.VelocityRange.Min, VelocityMax: normalized.VelocityRange.Max,
			ParticleLimit: normalized.ParticleLimit, IntegrityRequired: normalized.IntegrityRequired,
		},
		Issues: make([]PlanIssue, 0),
	}
	add := func(path, code, message string) {
		result.Issues = append(result.Issues, PlanIssue{FieldPath: path, Code: code, Message: message})
	}
	if normalized.StandardCode == "" {
		add("standardCode", "required", "认证规范不能为空")
	}
	if normalized.Revision == "" {
		add("revision", "required", "规范修订号不能为空")
	}
	minInvalid := math.IsNaN(normalized.VelocityRange.Min) || math.IsInf(normalized.VelocityRange.Min, 0) || normalized.VelocityRange.Min <= 0
	maxInvalid := math.IsNaN(normalized.VelocityRange.Max) || math.IsInf(normalized.VelocityRange.Max, 0) || normalized.VelocityRange.Max <= 0
	if minInvalid {
		add("velocityRange.min", "invalid_number", "风速下限必须为正有限数")
	}
	if maxInvalid {
		add("velocityRange.max", "invalid_number", "风速上限必须为正有限数")
	}
	if !minInvalid && !maxInvalid && normalized.VelocityRange.Min > normalized.VelocityRange.Max {
		add("velocityRange", "inverted_range", "风速下限不能高于上限")
	}
	if normalized.ParticleLimit <= 0 {
		add("particleLimit", "non_positive", "粒子计数上限必须为正整数")
	}
	if len(normalized.PointOrder) == 0 {
		add("pointOrder", "required", "至少需要一个测点")
	}
	if len(normalized.PointOrder) > 200 {
		add("pointOrder", "too_many_items", "测点数量不能超过 200")
	}
	seen := make(map[string]int, len(normalized.PointOrder))
	for index, point := range normalized.PointOrder {
		path := "pointOrder[" + strconv.Itoa(index) + "]"
		if point == "" {
			add(path, "required", "测点标识不能为空")
			continue
		}
		if len([]rune(point)) > 80 {
			add(path, "too_long", "测点标识不能超过 80 个字符")
		}
		if first, ok := seen[point]; ok {
			add(path, "duplicate", "测点与 pointOrder["+strconv.Itoa(first)+"] 规范化后重复")
		} else {
			seen[point] = index
		}
	}
	result.Valid = len(result.Issues) == 0
	if result.Valid {
		payload, _ := json.Marshal(normalized)
		sum := sha256.Sum256(payload)
		result.PlanDigest = hex.EncodeToString(sum[:])
	}
	return result
}

func ExpectedPoint(order []string, completed int) (string, bool) {
	if completed < 0 || completed >= len(order) {
		return "", false
	}
	return order[completed], true
}

func ContainsPoint(order []string, pointID string) bool {
	for _, point := range order {
		if point == pointID {
			return true
		}
	}
	return false
}
