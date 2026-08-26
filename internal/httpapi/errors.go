package httpapi

import (
	"net/http"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/certification"
)

type errorResponse struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"requestID,omitempty"`
		Details   any    `json:"details,omitempty"`
	} `json:"error"`
}

func writeAPIError(writer http.ResponseWriter, status int, code, message string) {
	response := errorResponse{}
	response.Error.Code = code
	response.Error.Message = message
	writeJSON(writer, status, response)
}

func writeDomainError(writer http.ResponseWriter, err error) {
	code := certification.ErrorCodeOf(err)
	status := http.StatusInternalServerError
	switch code {
	case certification.CodeValidation:
		status = http.StatusBadRequest
	case certification.CodeNotFound:
		status = http.StatusNotFound
	case certification.CodeStaleVersion:
		status = http.StatusPreconditionFailed
	case certification.CodeConflict, certification.CodeInvalidState, certification.CodeIdempotency:
		status = http.StatusConflict
	}
	response := errorResponse{}
	response.Error.Code = string(code)
	response.Error.Message = err.Error()
	response.Error.Details = certification.ErrorDetailsOf(err)
	writeJSON(writer, status, response)
}

func requireRole(writer http.ResponseWriter, request *http.Request, allowed ...string) bool {
	role := request.Header.Get("X-Actor-Role")
	for _, value := range allowed {
		if role == value {
			return true
		}
	}
	writeAPIError(writer, http.StatusForbidden, "forbidden", "X-Actor-Role 无权执行此操作")
	return false
}
