package certification

import (
	"strings"
	"time"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/assessment"
)

func IsAssignmentAdjustmentAction(action string) bool {
	switch strings.TrimSpace(action) {
	case "adjust_assignment", "adjust_responsibility", "assignment_adjustment", "reassign", "adjust":
		return true
	default:
		return false
	}
}

func (s *Service) AdjustDeviation(caseID string, command RemediateCommand) (*CertificationCase, error) {
	if strings.TrimSpace(command.NewAssignee) == "" {
		command.NewAssignee = command.Assignee
	}
	if command.NewDueAt.IsZero() {
		command.NewDueAt = command.DueAt
	}
	command.Assignee = ""
	command.DueAt = time.Time{}
	if err := validateIdempotencyKey(command.IdempotencyKey); err != nil {
		return nil, err
	}
	digest, err := requestDigest(command)
	if err != nil {
		return nil, err
	}
	scope := "adjust-deviation:" + caseID
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
	deviation := findDeviation(value, command.DeviationID)
	if deviation == nil {
		return nil, domainError(CodeNotFound, "偏差 %s 不存在", command.DeviationID)
	}
	if deviation.Status != "open" && deviation.Status != "ready_for_retest" {
		return nil, domainError(CodeInvalidState, "已关闭偏差不能调整责任或期限")
	}
	if !IsAssignmentAdjustmentAction(command.Action) {
		return nil, domainError(CodeValidation, "action 必须为 adjust_assignment")
	}
	if err := validatePerson("新责任人", command.NewAssignee); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	if err := validateDueDate(command.NewDueAt, now); err != nil {
		return nil, err
	}
	if err := validateChineseReason("调整原因", command.AdjustmentReason); err != nil {
		return nil, err
	}
	operator := strings.TrimSpace(command.Operator)
	if operator == "" {
		operator = strings.TrimSpace(command.Actor)
	}
	if err := validatePerson("操作人", operator); err != nil {
		return nil, err
	}
	if command.NewDueAt.After(deviation.DueAt) {
		if err := validateChineseReason("延长期限原因", command.ExtensionReason); err != nil {
			return nil, err
		}
	}
	newAssignee := strings.TrimSpace(command.NewAssignee)
	newDueAt := command.NewDueAt.UTC()
	if newAssignee == deviation.Assignee && newDueAt.Equal(deviation.DueAt) {
		return nil, domainError(CodeValidation, "责任人或期限至少需要变更一项")
	}
	deviation.AssignmentHistory = append(deviation.AssignmentHistory, DeviationAssignmentChange{
		PreviousAssignee: deviation.Assignee, PreviousDueAt: deviation.DueAt.UTC(), NewAssignee: newAssignee,
		NewDueAt: newDueAt, Operator: operator, Reason: strings.TrimSpace(command.AdjustmentReason),
		ExtensionReason: strings.TrimSpace(command.ExtensionReason), OccurredAt: now,
	})
	deviation.Assignee = newAssignee
	deviation.DueAt = newDueAt
	value.Version++
	if err := s.persist(scope, command.IdempotencyKey, digest, "deviation.assignment_adjusted", value, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Service) Remediate(caseID string, command RemediateCommand) (*CertificationCase, error) {
	if err := validateIdempotencyKey(command.IdempotencyKey); err != nil {
		return nil, err
	}
	if strings.TrimSpace(command.Action) != "" && strings.TrimSpace(command.Action) != "remediate" {
		return nil, domainError(CodeValidation, "未知偏差整改 action")
	}
	digest, err := requestDigest(command)
	if err != nil {
		return nil, err
	}
	scope := "remediate:" + caseID
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
	if value.Status != StatusRemediation {
		return nil, domainError(CodeInvalidState, "当前状态不允许登记整改")
	}
	deviation := findDeviation(value, command.DeviationID)
	if deviation == nil {
		return nil, domainError(CodeNotFound, "偏差 %s 不存在", command.DeviationID)
	}
	if deviation.Status != "open" {
		return nil, domainError(CodeConflict, "偏差当前不能登记整改")
	}
	if strings.TrimSpace(command.RemediationNote) == "" || strings.TrimSpace(command.EvidenceDigest) == "" || strings.TrimSpace(command.Actor) == "" {
		return nil, domainError(CodeValidation, "整改说明、证据摘要和操作人不能为空")
	}
	if err := validateText("整改说明", command.RemediationNote, 2, 1000); err != nil {
		return nil, err
	}
	if err := validateEvidence(command.EvidenceDigest); err != nil {
		return nil, err
	}
	if err := validatePerson("整改操作人", command.Actor); err != nil {
		return nil, err
	}
	deviation.RemediationNote = strings.TrimSpace(command.RemediationNote)
	deviation.EvidenceDigest = strings.TrimSpace(command.EvidenceDigest)
	deviation.Status = "ready_for_retest"
	value.Version++
	if err := s.persist(scope, command.IdempotencyKey, digest, "deviation.remediated", value, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Service) Retest(caseID string, command RetestCommand) (*CertificationCase, error) {
	if err := validateIdempotencyKey(command.IdempotencyKey); err != nil {
		return nil, err
	}
	digest, err := requestDigest(command)
	if err != nil {
		return nil, err
	}
	scope := "retest:" + caseID
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
	deviation := findDeviation(value, command.DeviationID)
	if deviation == nil {
		return nil, domainError(CodeNotFound, "偏差 %s 不存在", command.DeviationID)
	}
	if value.Status != StatusRemediation || deviation.Status != "ready_for_retest" {
		return nil, domainError(CodeInvalidState, "偏差尚未完成整改或当前状态不可复测")
	}
	reading := assessment.Reading{
		PointID: deviation.PointID, Velocity: command.Velocity, ParticleCount: command.ParticleCount,
		IntegrityPassed: command.IntegrityPassed, EvidenceDigest: command.EvidenceDigest,
		MeasuredBy: command.MeasuredBy, MeasuredAt: command.MeasuredAt,
	}
	if err := validateEvidence(command.EvidenceDigest); err != nil {
		return nil, err
	}
	if err := validatePerson("复测人员", command.MeasuredBy); err != nil {
		return nil, err
	}
	if err := assessment.ValidateReading(reading, s.now()); err != nil {
		return nil, domainError(CodeValidation, "%v", err)
	}
	result := assessment.Evaluate(toAssessmentPlan(value.Plan), reading)
	measurementID, err := newID("meas_")
	if err != nil {
		return nil, err
	}
	attempt := 1
	for _, item := range value.Measurements {
		if item.PointID == deviation.PointID {
			attempt++
		}
	}
	value.Measurements = append(value.Measurements, Measurement{
		MeasurementID: measurementID, CaseID: caseID, PointID: deviation.PointID, Attempt: attempt,
		Velocity: command.Velocity, ParticleCount: command.ParticleCount, IntegrityPassed: command.IntegrityPassed,
		EvidenceDigest: strings.TrimSpace(command.EvidenceDigest), MeasuredBy: strings.TrimSpace(command.MeasuredBy),
		MeasuredAt: command.MeasuredAt.UTC(), Outcome: result.Outcome, ReasonCodes: result.ReasonCodes, Retest: true,
	})
	if result.Outcome == "passed" {
		now := s.now().UTC()
		deviation.Status = "closed"
		deviation.ClosedAt = &now
		if allDeviationsClosed(value) {
			if initialPointsCompleted(value) {
				if err := transition(value, StatusAwaitingReview); err != nil {
					return nil, err
				}
			} else {
				if err := transition(value, StatusTesting); err != nil {
					return nil, err
				}
			}
		}
	} else {
		deviation.Status = "open"
		deviation.ReasonCode = strings.Join(result.ReasonCodes, ",")
		deviation.RemediationNote = ""
		deviation.EvidenceDigest = ""
	}
	value.Version++
	if err := s.persist(scope, command.IdempotencyKey, digest, "deviation.retested", value, value); err != nil {
		return nil, err
	}
	return value, nil
}

func findDeviation(value *CertificationCase, id string) *Deviation {
	for index := range value.Deviations {
		if value.Deviations[index].DeviationID == id {
			return &value.Deviations[index]
		}
	}
	return nil
}

func allDeviationsClosed(value *CertificationCase) bool {
	for _, item := range value.Deviations {
		if item.Status != "closed" {
			return false
		}
	}
	return true
}

func initialPointsCompleted(value *CertificationCase) bool {
	seen := make(map[string]bool)
	for _, item := range value.Measurements {
		if !item.Retest && item.SupersededByMeasurementID == "" {
			seen[item.PointID] = true
		}
	}
	return len(seen) == len(value.Plan.PointOrder)
}
