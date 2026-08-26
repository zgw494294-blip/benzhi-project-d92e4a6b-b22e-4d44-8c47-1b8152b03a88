package certification

import (
	"strings"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/credential"
)

func (s *Service) IssueCredential(caseID string, command IssueCredentialCommand) (*credential.ReleaseCredential, error) {
	if err := validateIdempotencyKey(command.IdempotencyKey); err != nil {
		return nil, err
	}
	digest, err := requestDigest(command)
	if err != nil {
		return nil, err
	}
	scope := "issue-credential:" + caseID
	unlock := s.locks.Lock(caseID)
	defer unlock()
	var replay credential.ReleaseCredential
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
	if value.Status == StatusReleased && value.Credential != nil {
		return nil, domainError(CodeConflict, "案例已经签发启用凭据")
	}
	if value.Status != StatusFrozen || value.FrozenSnapshot == nil {
		return nil, domainError(CodeInvalidState, "只有已冻结且合格的案例可以签发凭据")
	}
	if strings.TrimSpace(command.IssuedBy) == "" {
		return nil, domainError(CodeValidation, "签发人不能为空")
	}
	if err := validatePerson("签发人", command.IssuedBy); err != nil {
		return nil, err
	}
	issued, err := s.issuer.Issue(*value.FrozenSnapshot, command.IssuedBy)
	if err != nil {
		return nil, domainError(CodeValidation, "%v", err)
	}
	value.Credential = &issued
	if err := transition(value, StatusReleased); err != nil {
		return nil, err
	}
	value.Version++
	if err := s.persist(scope, command.IdempotencyKey, digest, "credential.issued", value, &issued); err != nil {
		return nil, err
	}
	return &issued, nil
}

func (s *Service) VerifyCredential(credentialID, code string) (credential.Verification, error) {
	s.mu.RLock()
	caseID, ok := s.credentials[credentialID]
	value := s.cases[caseID]
	if value != nil {
		value = cloneCase(value)
	}
	s.mu.RUnlock()
	if !ok || value == nil || value.Credential == nil || value.FrozenSnapshot == nil {
		return credential.Verification{}, domainError(CodeNotFound, "启用凭据不存在")
	}
	return credential.Verify(*value.Credential, *value.FrozenSnapshot, code), nil
}
