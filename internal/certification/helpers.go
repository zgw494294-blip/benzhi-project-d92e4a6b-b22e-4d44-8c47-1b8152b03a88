package certification

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/ledger"
)

func cloneCase(value *CertificationCase) *CertificationCase {
	payload, _ := json.Marshal(value)
	var result CertificationCase
	_ = json.Unmarshal(payload, &result)
	return &result
}

func newID(prefix string) (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buffer), nil
}

func requestDigest(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func validateIdempotencyKey(key string) error {
	key = strings.TrimSpace(key)
	if len(key) < 8 || len(key) > 128 {
		return domainError(CodeValidation, "Idempotency-Key 长度必须在 8 到 128 之间")
	}
	return nil
}

func (s *Service) replay(scope, key, digest string, target any) (bool, error) {
	record, ok := s.store.LookupIdempotency(scope, key)
	if !ok {
		return false, nil
	}
	if record.RequestDigest != digest {
		return false, domainError(CodeIdempotency, "幂等键已用于不同请求")
	}
	if err := json.Unmarshal(record.Response, target); err != nil {
		return false, fmt.Errorf("解码幂等响应: %w", err)
	}
	return true, nil
}

func (s *Service) persist(scope, key, digest, eventType string, value *CertificationCase, response any) error {
	if err := validateAggregate(value); err != nil {
		return domainError(CodePersistence, "认证聚合不变量失败: %v", err)
	}
	payload, err := json.Marshal(response)
	if err != nil {
		return err
	}
	record := ledger.IdempotencyRecord{
		Scope: scope, Key: key, RequestDigest: digest, StatusCode: 200,
		Response: payload, RecordedAt: s.now().UTC(),
	}
	if _, err := s.store.Commit(ledger.Commit{EventType: eventType, CaseID: value.CaseID, State: value, Idempotency: &record}); err != nil {
		return domainError(CodePersistence, "持久化认证事件失败: %v", err)
	}
	s.mu.Lock()
	s.cases[value.CaseID] = cloneCase(value)
	caseIDs := s.deviceCases[value.NormalizedCabinetCode]
	found := false
	for _, caseID := range caseIDs {
		if caseID == value.CaseID {
			found = true
			break
		}
	}
	if !found {
		s.deviceCases[value.NormalizedCabinetCode] = append(caseIDs, value.CaseID)
	}
	if value.Credential != nil {
		s.credentials[value.Credential.CredentialID] = value.CaseID
	}
	s.mu.Unlock()
	return nil
}

func requireVersion(value *CertificationCase, expected int64) error {
	if err := validateExpectedVersion(expected); err != nil {
		return err
	}
	if value.Version != expected {
		return domainError(CodeStaleVersion, "版本已变化：期望 %d，当前 %d", expected, value.Version)
	}
	return nil
}
