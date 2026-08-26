package certification

import (
	"strings"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/assessment"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/credential"
)

func (s *Service) Review(caseID string, command ReviewCommand) (*CertificationCase, error) {
	if err := validateIdempotencyKey(command.IdempotencyKey); err != nil {
		return nil, err
	}
	digest, err := requestDigest(command)
	if err != nil {
		return nil, err
	}
	scope := "review:" + caseID
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
	if value.Status != StatusAwaitingReview {
		return nil, domainError(CodeInvalidState, "案例尚未进入待复核状态")
	}
	if err := validatePerson("质量复核员", command.Reviewer); err != nil {
		return nil, err
	}
	if err := validateText("复核意见", command.Comment, 2, 1000); err != nil {
		return nil, err
	}
	if command.Decision != "approve" && command.Decision != "return" {
		return nil, domainError(CodeValidation, "decision 必须为 approve 或 return")
	}
	checklist := buildReviewChecklist(value)
	if strings.TrimSpace(command.ChecklistDigest) != "" && command.ChecklistDigest != checklist.Digest {
		return nil, &DomainError{Code: CodeConflict, Message: "复核就绪清单摘要已经陈旧", Details: map[string]any{"currentChecklistDigest": checklist.Digest}}
	}
	if len(checklist.Blockers) != 0 {
		return nil, &DomainError{Code: CodeConflict, Message: "案例存在复核阻断项", Details: map[string]any{"blockers": checklist.Blockers}}
	}
	if command.Decision == "return" {
		return s.returnReview(scope, digest, value, command)
	}
	latest := latestOutcomes(value)
	value.ReviewHistory = append(value.ReviewHistory, ReviewDecision{Decision: "approve", Reviewer: strings.TrimSpace(command.Reviewer), Comment: strings.TrimSpace(command.Comment), DecidedAt: s.now().UTC()})
	frozenAt := s.now().UTC()
	if err := transition(value, StatusFrozen); err != nil {
		return nil, err
	}
	value.FrozenAt = &frozenAt
	value.Version++
	value.FrozenSnapshot = &credential.FrozenSnapshot{
		CaseID: value.CaseID, CaseVersion: value.Version, CabinetCode: value.CabinetCode,
		Location: value.Location, CabinetClass: value.CabinetClass, BaselineStatus: value.BaselineStatus,
		PlanDigest: value.Plan.Digest, LatestOutcomes: latest, FrozenAt: frozenAt,
	}
	if err := s.persist(scope, command.IdempotencyKey, digest, "review.approved", value, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Service) returnReview(scope, digest string, value *CertificationCase, command ReviewCommand) (*CertificationCase, error) {
	issues := append([]ReviewIssue(nil), command.Issues...)
	if len(issues) == 0 && strings.TrimSpace(command.PointID) != "" {
		issues = append(issues, ReviewIssue{PointID: command.PointID, Reason: command.Comment, Assignee: command.Assignee, DueAt: command.DueAt})
	}
	if len(issues) == 0 {
		return nil, domainError(CodeValidation, "复核退回至少需要一个问题项")
	}
	now := s.now().UTC()
	seen := make(map[string]bool, len(issues))
	for index := range issues {
		issue := &issues[index]
		issue.PointID = strings.TrimSpace(issue.PointID)
		issue.Reason = strings.TrimSpace(issue.Reason)
		issue.Assignee = strings.TrimSpace(issue.Assignee)
		issue.DueAt = issue.DueAt.UTC()
		if !assessment.ContainsPoint(value.Plan.PointOrder, issue.PointID) {
			return nil, domainError(CodeValidation, "第 %d 个退回问题包含方案外测点 %s", index+1, issue.PointID)
		}
		if seen[issue.PointID] {
			return nil, domainError(CodeValidation, "退回问题包含重复测点 %s", issue.PointID)
		}
		seen[issue.PointID] = true
		if err := validateChineseReason("退回原因", issue.Reason); err != nil {
			return nil, err
		}
		if err := validatePerson("退回责任人", issue.Assignee); err != nil {
			return nil, err
		}
		if err := validateDueDate(issue.DueAt, now); err != nil {
			return nil, err
		}
	}
	deviations := make([]Deviation, 0, len(issues))
	for _, issue := range issues {
		deviationID, err := newID("dev_")
		if err != nil {
			return nil, err
		}
		deviations = append(deviations, Deviation{
			DeviationID: deviationID, CaseID: value.CaseID, PointID: issue.PointID, ReasonCode: "QUALITY_REVIEW_RETURNED",
			Description: issue.Reason, Assignee: issue.Assignee, DueAt: issue.DueAt, Status: "open", OpenedAt: now,
			AssignmentHistory: []DeviationAssignmentChange{},
		})
	}
	value.Deviations = append(value.Deviations, deviations...)
	value.ReviewHistory = append(value.ReviewHistory, ReviewDecision{Decision: "return", Reviewer: strings.TrimSpace(command.Reviewer), Comment: strings.TrimSpace(command.Comment), DecidedAt: now, Issues: issues})
	if err := transition(value, StatusRemediation); err != nil {
		return nil, err
	}
	value.Version++
	if err := s.persist(scope, command.IdempotencyKey, digest, "review.returned", value, value); err != nil {
		return nil, err
	}
	return value, nil
}

func latestOutcomes(value *CertificationCase) map[string]string {
	latest := make(map[string]string)
	for _, item := range value.Measurements {
		if item.SupersededByMeasurementID == "" {
			latest[item.PointID] = item.Outcome
		}
	}
	return latest
}
