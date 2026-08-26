package certification

import (
	"strings"
)

func (s *Service) CreateCase(command CreateCaseCommand) (*CertificationCase, error) {
	command.CabinetCode = NormalizeCabinetCode(command.CabinetCode)
	if err := validateIdempotencyKey(command.IdempotencyKey); err != nil {
		return nil, err
	}
	if err := validateCaseFields(command); err != nil {
		return nil, err
	}
	if command.BaselineStatus != "qualified" && command.BaselineStatus != "conditional" {
		return nil, domainError(CodeValidation, "baselineStatus 必须为 qualified 或 conditional")
	}
	digest, err := requestDigest(command)
	if err != nil {
		return nil, err
	}
	scope := "create-case"
	unlockIdempotency := s.locks.Lock("create-idempotency:" + command.IdempotencyKey)
	defer unlockIdempotency()
	unlockDevice := s.locks.Lock("device:" + command.CabinetCode)
	defer unlockDevice()
	var replay CertificationCase
	if ok, err := s.replay(scope, command.IdempotencyKey, digest, &replay); ok || err != nil {
		return &replay, err
	}
	s.mu.RLock()
	deviceCaseIDs := append([]string(nil), s.deviceCases[command.CabinetCode]...)
	var inFlight *CertificationCase
	var previous *CertificationCase
	for _, existingID := range deviceCaseIDs {
		existing := s.cases[existingID]
		if existing == nil {
			continue
		}
		if isInFlightStatus(existing.Status) {
			inFlight = cloneCase(existing)
			break
		}
		if existing.Status == StatusReleased && (previous == nil || previous.CreatedAt.Before(existing.CreatedAt) || previous.CreatedAt.Equal(existing.CreatedAt) && previous.CaseID < existing.CaseID) {
			previous = cloneCase(existing)
		}
	}
	s.mu.RUnlock()
	if inFlight != nil {
		return nil, deviceConflictError(inFlight)
	}
	caseID, err := newID("case_")
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	value := &CertificationCase{
		CaseID: caseID, CabinetCode: command.CabinetCode, NormalizedCabinetCode: command.CabinetCode, Location: strings.TrimSpace(command.Location),
		CabinetClass: strings.TrimSpace(command.CabinetClass), BaselineStatus: command.BaselineStatus,
		Status: StatusDraft, Version: 1, CreatedAt: now,
		Measurements: []Measurement{}, Deviations: []Deviation{}, ReviewHistory: []ReviewDecision{},
	}
	if previous != nil {
		if previous.NormalizedCabinetCode != command.CabinetCode || previous.Status != StatusReleased {
			return nil, domainError(CodePersistence, "前序案例设备或状态无效")
		}
		value.PreviousCaseID = previous.CaseID
	}
	if err := s.persist(scope, command.IdempotencyKey, digest, "case.created", value, value); err != nil {
		return nil, err
	}
	return cloneCase(value), nil
}
