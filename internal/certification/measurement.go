package certification

import (
	"strings"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/assessment"
)

func (s *Service) SubmitMeasurement(caseID string, command SubmitMeasurementCommand) (*CertificationCase, error) {
	if command.CorrectsMeasurementID == "" {
		command.CorrectsMeasurementID = strings.TrimSpace(command.CorrectedMeasurementID)
	}
	command.CorrectedMeasurementID = ""
	if err := validateIdempotencyKey(command.IdempotencyKey); err != nil {
		return nil, err
	}
	digest, err := requestDigest(command)
	if err != nil {
		return nil, err
	}
	scope := "measurement:" + caseID
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
	if strings.TrimSpace(command.CorrectsMeasurementID) != "" {
		return s.correctMeasurement(scope, digest, value, command)
	}
	if value.Status != StatusAwaitingTest && value.Status != StatusTesting {
		return nil, domainError(CodeInvalidState, "当前状态不允许提交常规测量")
	}
	reading := assessment.Reading{
		PointID: command.PointID, Velocity: command.Velocity, ParticleCount: command.ParticleCount,
		IntegrityPassed: command.IntegrityPassed, EvidenceDigest: command.EvidenceDigest,
		MeasuredBy: command.MeasuredBy, MeasuredAt: command.MeasuredAt,
	}
	if err := validateEvidence(command.EvidenceDigest); err != nil {
		return nil, err
	}
	if err := validatePerson("测量人员", command.MeasuredBy); err != nil {
		return nil, err
	}
	if err := assessment.ValidateReading(reading, s.now()); err != nil {
		return nil, domainError(CodeValidation, "%v", err)
	}
	completedInitial := 0
	for _, recorded := range value.Measurements {
		if !recorded.Retest && recorded.SupersededByMeasurementID == "" {
			completedInitial++
		}
	}
	expected, ok := assessment.ExpectedPoint(value.Plan.PointOrder, completedInitial)
	if !ok {
		return nil, domainError(CodeConflict, "全部测点已经完成")
	}
	if command.PointID != expected {
		return nil, domainError(CodeConflict, "必须先提交测点 %s", expected)
	}
	result := assessment.Evaluate(toAssessmentPlan(value.Plan), reading)
	measurementID, err := newID("meas_")
	if err != nil {
		return nil, err
	}
	measurement := Measurement{
		MeasurementID: measurementID, CaseID: caseID, PointID: command.PointID, Attempt: 1,
		Velocity: command.Velocity, ParticleCount: command.ParticleCount, IntegrityPassed: command.IntegrityPassed,
		EvidenceDigest: strings.TrimSpace(command.EvidenceDigest), MeasuredBy: strings.TrimSpace(command.MeasuredBy),
		MeasuredAt: command.MeasuredAt.UTC(), Outcome: result.Outcome, ReasonCodes: result.ReasonCodes,
	}
	value.Measurements = append(value.Measurements, measurement)
	eventType := "measurement.recorded"
	if result.Outcome == "failed" {
		if err := validatePerson("偏差责任人", command.Assignee); err != nil {
			return nil, err
		}
		if err := validateDueDate(command.DueAt, s.now()); err != nil {
			return nil, err
		}
		deviationID, err := newID("dev_")
		if err != nil {
			return nil, err
		}
		value.Deviations = append(value.Deviations, Deviation{
			DeviationID: deviationID, CaseID: caseID, PointID: command.PointID,
			ReasonCode: strings.Join(result.ReasonCodes, ","), Assignee: strings.TrimSpace(command.Assignee),
			DueAt: command.DueAt.UTC(), Status: "open", OpenedAt: s.now().UTC(),
		})
		if err := transition(value, StatusRemediation); err != nil {
			return nil, err
		}
		eventType = "measurement.failed_and_deviation_opened"
	} else if completedInitial+1 == len(value.Plan.PointOrder) {
		if err := transition(value, StatusAwaitingReview); err != nil {
			return nil, err
		}
	} else {
		if err := transition(value, StatusTesting); err != nil {
			return nil, err
		}
	}
	value.Version++
	if err := s.persist(scope, command.IdempotencyKey, digest, eventType, value, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Service) correctMeasurement(scope, digest string, value *CertificationCase, command SubmitMeasurementCommand) (*CertificationCase, error) {
	if value.Status != StatusTesting && value.Status != StatusAwaitingReview {
		return nil, domainError(CodeInvalidState, "当前状态已跨越实测纠错边界")
	}
	if len(value.ReviewHistory) != 0 {
		return nil, domainError(CodeConflict, "案例已经开始复核，不能更正实测")
	}
	if len(value.Deviations) != 0 {
		return nil, domainError(CodeConflict, "案例已经进入过整改闭环，不能更正初测")
	}
	var target *Measurement
	var lastInitial *Measurement
	for index := range value.Measurements {
		measurement := &value.Measurements[index]
		if measurement.MeasurementID == command.CorrectsMeasurementID {
			target = measurement
		}
		if !measurement.Retest && measurement.SupersededByMeasurementID == "" {
			lastInitial = measurement
		}
	}
	if target == nil {
		return nil, domainError(CodeNotFound, "被更正测量 %s 不存在", command.CorrectsMeasurementID)
	}
	if target.Retest {
		return nil, domainError(CodeConflict, "测量 %s 是复测记录，不能通过初测纠错入口更正", target.MeasurementID)
	}
	if target.SupersededByMeasurementID != "" {
		return nil, domainError(CodeConflict, "测量 %s 已被 %s 替代", target.MeasurementID, target.SupersededByMeasurementID)
	}
	if lastInitial == nil || lastInitial.MeasurementID != target.MeasurementID {
		return nil, domainError(CodeConflict, "测量 %s 不是当前末次初测", target.MeasurementID)
	}
	if strings.TrimSpace(command.PointID) != target.PointID {
		return nil, domainError(CodeValidation, "纠错测点必须与原测量 %s 一致", target.PointID)
	}
	if err := validateChineseReason("更正原因", command.CorrectionReason); err != nil {
		return nil, err
	}
	reading := assessment.Reading{
		PointID: target.PointID, Velocity: command.Velocity, ParticleCount: command.ParticleCount,
		IntegrityPassed: command.IntegrityPassed, EvidenceDigest: command.EvidenceDigest,
		MeasuredBy: command.MeasuredBy, MeasuredAt: command.MeasuredAt,
	}
	if err := validateEvidence(command.EvidenceDigest); err != nil {
		return nil, err
	}
	if err := validatePerson("测量人员", command.MeasuredBy); err != nil {
		return nil, err
	}
	if err := assessment.ValidateReading(reading, s.now()); err != nil {
		return nil, domainError(CodeValidation, "%v", err)
	}
	result := assessment.Evaluate(toAssessmentPlan(value.Plan), reading)
	if result.Outcome == "failed" {
		if err := validatePerson("偏差责任人", command.Assignee); err != nil {
			return nil, err
		}
		if err := validateDueDate(command.DueAt, s.now()); err != nil {
			return nil, err
		}
	}
	measurementID, err := newID("meas_")
	if err != nil {
		return nil, err
	}
	attempt := 1
	for _, measurement := range value.Measurements {
		if measurement.PointID == target.PointID && measurement.Attempt >= attempt {
			attempt = measurement.Attempt + 1
		}
	}
	target.SupersededByMeasurementID = measurementID
	value.Measurements = append(value.Measurements, Measurement{
		MeasurementID: measurementID, CaseID: value.CaseID, PointID: target.PointID, Attempt: attempt,
		Velocity: command.Velocity, ParticleCount: command.ParticleCount, IntegrityPassed: command.IntegrityPassed,
		EvidenceDigest: strings.TrimSpace(command.EvidenceDigest), MeasuredBy: strings.TrimSpace(command.MeasuredBy), MeasuredAt: command.MeasuredAt.UTC(),
		Outcome: result.Outcome, ReasonCodes: result.ReasonCodes, CorrectsMeasurementID: target.MeasurementID,
		CorrectionReason: strings.TrimSpace(command.CorrectionReason),
	})
	eventType := "measurement.corrected"
	if result.Outcome == "failed" {
		deviationID, err := newID("dev_")
		if err != nil {
			return nil, err
		}
		now := s.now().UTC()
		value.Deviations = append(value.Deviations, Deviation{
			DeviationID: deviationID, CaseID: value.CaseID, PointID: target.PointID,
			ReasonCode: strings.Join(result.ReasonCodes, ","), Assignee: strings.TrimSpace(command.Assignee), DueAt: command.DueAt.UTC(),
			Status: "open", OpenedAt: now, AssignmentHistory: []DeviationAssignmentChange{},
		})
		if err := transition(value, StatusRemediation); err != nil {
			return nil, err
		}
		eventType = "measurement.corrected_and_deviation_opened"
	}
	value.Version++
	if err := s.persist(scope, command.IdempotencyKey, digest, eventType, value, value); err != nil {
		return nil, err
	}
	return value, nil
}

func toAssessmentPlan(plan *CertificationPlan) assessment.Plan {
	return assessment.Plan{
		StandardCode: plan.StandardCode, Revision: plan.Revision, VelocityRange: plan.VelocityRange,
		ParticleLimit: plan.ParticleLimit, IntegrityRequired: plan.IntegrityRequired,
		PointOrder: append([]string(nil), plan.PointOrder...),
	}
}
