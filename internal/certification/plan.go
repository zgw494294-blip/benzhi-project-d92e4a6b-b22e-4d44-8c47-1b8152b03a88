package certification

import (
	"strings"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/assessment"
)

func (s *Service) PreflightPlan(caseID string, command LockPlanCommand) (*PlanPreflightReport, error) {
	value, err := s.getCaseProjection(caseID)
	if err != nil {
		return nil, err
	}
	if value.Status != StatusDraft || value.Plan != nil {
		return nil, domainError(CodeInvalidState, "只有草稿案例可以执行方案预检")
	}
	result := assessPlanCommand(command)
	return &PlanPreflightReport{
		CaseID: caseID, CaseVersion: value.Version, Valid: result.Valid, NormalizedPlan: result.NormalizedPlan,
		NormalizedPointOrder: append([]string(nil), result.NormalizedPlan.PointOrder...), PointCount: result.PointCount,
		ThresholdSummary: result.ThresholdSummary, PlanDigest: result.PlanDigest, Issues: result.Issues,
	}, nil
}

func (s *Service) LockPlan(caseID string, command LockPlanCommand) (*CertificationCase, error) {
	if err := validateIdempotencyKey(command.IdempotencyKey); err != nil {
		return nil, err
	}
	digest, err := requestDigest(command)
	if err != nil {
		return nil, err
	}
	scope := "lock-plan:" + caseID
	unlock := s.locks.Lock(caseID)
	defer unlock()
	var replay CertificationCase
	if ok, err := s.replay(scope, command.IdempotencyKey, digest, &replay); ok || err != nil {
		return &replay, err
	}
	value, err := s.getCaseProjection(caseID)
	if err != nil {
		return nil, err
	}
	if err := requireVersion(value, command.ExpectedVersion); err != nil {
		return nil, err
	}
	if value.Status != StatusDraft || value.Plan != nil {
		return nil, domainError(CodeInvalidState, "只有草稿案例可以锁定方案")
	}
	result := assessPlanCommand(command)
	if !result.Valid {
		return nil, &DomainError{Code: CodeValidation, Message: "认证方案完整性校验失败", Details: map[string]any{"issues": result.Issues}}
	}
	planValue := result.NormalizedPlan
	planID, err := newID("plan_")
	if err != nil {
		return nil, err
	}
	value.Plan = &CertificationPlan{
		PlanID: planID, CaseID: caseID, StandardCode: planValue.StandardCode, Revision: planValue.Revision,
		VelocityRange: planValue.VelocityRange, ParticleLimit: planValue.ParticleLimit,
		IntegrityRequired: planValue.IntegrityRequired, PointOrder: append([]string(nil), planValue.PointOrder...),
		Digest: result.PlanDigest, LockedAt: s.now().UTC(),
	}
	if err := transition(value, StatusAwaitingTest); err != nil {
		return nil, err
	}
	value.Version++
	if err := s.persist(scope, command.IdempotencyKey, digest, "plan.locked", value, value); err != nil {
		return nil, err
	}
	return value, nil
}

func assessPlanCommand(command LockPlanCommand) assessment.PlanAssessment {
	return assessment.AssessPlan(assessment.Plan{
		StandardCode: strings.TrimSpace(command.StandardCode), Revision: strings.TrimSpace(command.Revision), VelocityRange: command.VelocityRange,
		ParticleLimit: command.ParticleLimit, IntegrityRequired: command.IntegrityRequired,
		PointOrder: append([]string(nil), command.PointOrder...),
	})
}
