package httpapi

import (
	"net/http"
	"strings"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/certification"
)

func (a *API) IssueCredentialHandler(writer http.ResponseWriter, request *http.Request, caseID string) {
	if !requireRole(writer, request, "quality_reviewer") {
		return
	}
	var command certification.IssueCredentialCommand
	if err := decodeJSON(writer, request, &command); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	command.IdempotencyKey = idempotencyKey(request)
	value, err := a.service.IssueCredential(caseID, command)
	if err != nil {
		writeDomainError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, value)
}

func (a *API) VerifyCredentialHandler(writer http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/api/v1/credentials/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "verify" {
		writeAPIError(writer, http.StatusNotFound, "not_found", "路由不存在")
		return
	}
	code := request.URL.Query().Get("code")
	if code == "" {
		writeAPIError(writer, http.StatusBadRequest, "validation_error", "code 查询参数不能为空")
		return
	}
	value, err := a.service.VerifyCredential(parts[0], code)
	if err != nil {
		writeDomainError(writer, err)
		return
	}
	status := http.StatusOK
	if !value.Valid {
		status = http.StatusUnprocessableEntity
	}
	writeJSON(writer, status, value)
}
