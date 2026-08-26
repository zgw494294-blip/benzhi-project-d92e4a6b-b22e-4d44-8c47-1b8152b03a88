package certification

import "fmt"

type ErrorCode string

const (
	CodeValidation   ErrorCode = "validation_error"
	CodeNotFound     ErrorCode = "not_found"
	CodeConflict     ErrorCode = "conflict"
	CodeStaleVersion ErrorCode = "stale_version"
	CodeInvalidState ErrorCode = "invalid_state"
	CodeIdempotency  ErrorCode = "idempotency_conflict"
	CodePersistence  ErrorCode = "persistence_error"
)

type DomainError struct {
	Code    ErrorCode
	Message string
	Details any
}

func deviceConflictError(value *CertificationCase) error {
	return &DomainError{
		Code:    CodeConflict,
		Message: fmt.Sprintf("工作台 %s 已存在在途案例 %s（状态 %s，版本 %d）", value.NormalizedCabinetCode, value.CaseID, value.Status, value.Version),
		Details: map[string]any{"caseID": value.CaseID, "status": value.Status, "version": value.Version},
	}
}

func ErrorDetailsOf(err error) any {
	if value, ok := err.(*DomainError); ok {
		return value.Details
	}
	return nil
}

func (e *DomainError) Error() string { return e.Message }

func domainError(code ErrorCode, format string, args ...any) error {
	return &DomainError{Code: code, Message: fmt.Sprintf(format, args...)}
}

func ErrorCodeOf(err error) ErrorCode {
	if value, ok := err.(*DomainError); ok {
		return value.Code
	}
	return CodePersistence
}
