package httpapi

import (
	"net/http"
	"strings"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/certification"
)

func (a *API) LockPlanHandler(writer http.ResponseWriter, request *http.Request, caseID string) {
	if !requireRole(writer, request, "certification_engineer") {
		return
	}
	var command certification.LockPlanCommand
	if err := decodeJSON(writer, request, &command); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	command.IdempotencyKey = idempotencyKey(request)
	if raw := strings.TrimSpace(request.URL.Query().Get("preflight")); raw != "" {
		if raw != "true" && raw != "false" {
			writeAPIError(writer, http.StatusBadRequest, "validation_error", "preflight 查询参数必须为 true 或 false")
			return
		}
		command.Preflight = raw == "true"
	}
	if command.Preflight {
		value, err := a.service.PreflightPlan(caseID, command)
		if err != nil {
			writeDomainError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, value)
		return
	}
	value, err := a.service.LockPlan(caseID, command)
	if err != nil {
		writeDomainError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, value)
}
