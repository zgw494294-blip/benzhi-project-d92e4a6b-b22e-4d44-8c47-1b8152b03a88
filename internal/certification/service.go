package certification

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/assessment"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/credential"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/ledger"
)

type Service struct {
	mu          sync.RWMutex
	cases       map[string]*CertificationCase
	credentials map[string]string
	deviceCases map[string][]string
	store       *ledger.Store
	issuer      *credential.Issuer
	locks       *ledger.KeyLocker
	now         func() time.Time
}

func NewService(store *ledger.Store, recovery ledger.Recovery, issuer *credential.Issuer, now func() time.Time) (*Service, error) {
	if store == nil || issuer == nil {
		return nil, fmt.Errorf("认证服务依赖不能为空")
	}
	if now == nil {
		now = time.Now
	}
	service := &Service{
		cases: make(map[string]*CertificationCase), credentials: make(map[string]string), deviceCases: make(map[string][]string),
		store: store, issuer: issuer, locks: ledger.NewKeyLocker(), now: now,
	}
	for caseID, payload := range recovery.Cases {
		var value CertificationCase
		if err := json.Unmarshal(payload, &value); err != nil {
			return nil, fmt.Errorf("恢复案例 %s: %w", caseID, err)
		}
		if value.CaseID != caseID || value.Version <= 0 {
			return nil, fmt.Errorf("恢复案例 %s 的投影标识或版本无效", caseID)
		}
		if value.NormalizedCabinetCode == "" {
			value.NormalizedCabinetCode = NormalizeCabinetCode(value.CabinetCode)
		}
		value.CabinetCode = value.NormalizedCabinetCode
		if err := validateAggregate(&value); err != nil {
			return nil, fmt.Errorf("恢复案例 %s 的聚合不变量失败: %w", caseID, err)
		}
		copy := cloneCase(&value)
		service.cases[caseID] = copy
		service.deviceCases[copy.NormalizedCabinetCode] = append(service.deviceCases[copy.NormalizedCabinetCode], caseID)
		if copy.Credential != nil {
			service.credentials[copy.Credential.CredentialID] = caseID
		}
	}
	for device, caseIDs := range service.deviceCases {
		active := make([]string, 0, 2)
		for _, caseID := range caseIDs {
			if isInFlightStatus(service.cases[caseID].Status) {
				active = append(active, caseID)
			}
		}
		if len(active) > 1 {
			sort.Strings(active)
			return nil, fmt.Errorf("恢复设备索引失败：工作台 %s 存在多个在途案例 %s", device, strings.Join(active, ", "))
		}
		for _, caseID := range caseIDs {
			value := service.cases[caseID]
			if value.PreviousCaseID == "" {
				continue
			}
			previous := service.cases[value.PreviousCaseID]
			if previous == nil || previous.CaseID == value.CaseID || previous.Status != StatusReleased || previous.NormalizedCabinetCode != device {
				return nil, fmt.Errorf("恢复案例 %s 失败：前序案例 %s 不属于同一工作台或尚未放行", caseID, value.PreviousCaseID)
			}
		}
	}
	return service, nil
}

func (s *Service) getCaseProjection(caseID string) (*CertificationCase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value := s.cases[caseID]
	if value == nil {
		return nil, domainError(CodeNotFound, "认证案例 %s 不存在", caseID)
	}
	return cloneCase(value), nil
}

func (s *Service) GetCase(caseID string) (*CertificationCase, error) {
	value, err := s.getCaseProjection(caseID)
	if err != nil {
		return nil, err
	}
	s.enrichCase(value)
	return value, nil
}

func (s *Service) ListCases() []*CertificationCase {
	values, _ := s.QueryCases(CaseFilter{})
	return values
}

func (s *Service) QueryCases(filter CaseFilter) ([]*CertificationCase, error) {
	device := ""
	if strings.TrimSpace(filter.CabinetCode) != "" {
		device = NormalizeCabinetCode(filter.CabinetCode)
	}
	if filter.Status != "" && !IsValidStatus(filter.Status) {
		return nil, domainError(CodeValidation, "未知案例状态 %s", filter.Status)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*CertificationCase, 0, len(s.cases))
	for _, value := range s.cases {
		if device != "" && value.NormalizedCabinetCode != device {
			continue
		}
		if filter.Status != "" && value.Status != filter.Status {
			continue
		}
		result = append(result, cloneCase(value))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].CaseID < result[j].CaseID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func NormalizeCabinetCode(value string) string {
	return strings.ToUpper(strings.Join(strings.Fields(value), " "))
}

func IsValidStatus(status string) bool {
	switch status {
	case StatusDraft, StatusAwaitingTest, StatusTesting, StatusRemediation, StatusAwaitingReview, StatusFrozen, StatusReleased:
		return true
	default:
		return false
	}
}

func isInFlightStatus(status string) bool { return status != StatusReleased && IsValidStatus(status) }

func (s *Service) enrichCase(value *CertificationCase) {
	now := s.now().UTC()
	for index := range value.Deviations {
		deviation := &value.Deviations[index]
		if deviation.Status == "closed" {
			continue
		}
		deviation.DueStatus, deviation.OverdueDurationSeconds = assessment.DeviationDueStatus(deviation.DueAt, now)
	}
	if value.Status == StatusAwaitingReview {
		checklist := buildReviewChecklist(value)
		value.ReviewReadiness = &checklist
	}
}

func buildReviewChecklist(value *CertificationCase) assessment.ReviewChecklist {
	measurements := make([]assessment.ReadinessMeasurement, 0, len(value.Measurements))
	for _, item := range value.Measurements {
		if item.SupersededByMeasurementID != "" {
			continue
		}
		measurements = append(measurements, assessment.ReadinessMeasurement{
			MeasurementID: item.MeasurementID, PointID: item.PointID, Outcome: item.Outcome, EvidenceDigest: item.EvidenceDigest,
		})
	}
	deviations := make([]assessment.DeviationState, 0, len(value.Deviations))
	for _, item := range value.Deviations {
		deviations = append(deviations, assessment.DeviationState{PointID: item.PointID, Status: item.Status})
	}
	return assessment.BuildReviewChecklist(value.Version, value.Plan.PointOrder, measurements, deviations)
}
