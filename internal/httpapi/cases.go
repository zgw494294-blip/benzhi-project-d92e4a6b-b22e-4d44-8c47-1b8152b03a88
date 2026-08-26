package httpapi

import (
	"net/http"
	"strings"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/certification"
)

func (a *API) CreateCaseHandler(writer http.ResponseWriter, request *http.Request) {
	if !requireRole(writer, request, "certification_engineer") {
		return
	}
	var command certification.CreateCaseCommand
	if err := decodeJSON(writer, request, &command); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	command.CabinetCode = certification.NormalizeCabinetCode(command.CabinetCode)
	command.IdempotencyKey = idempotencyKey(request)
	value, err := a.service.CreateCase(command)
	if err != nil {
		writeDomainError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, value)
}

func (a *API) GetCaseHandler(writer http.ResponseWriter, request *http.Request, caseID string) {
	value, err := a.service.GetCase(caseID)
	if err != nil {
		writeDomainError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, value)
}

func (a *API) ListCasesHandler(writer http.ResponseWriter, request *http.Request) {
	filter := certification.CaseFilter{CabinetCode: request.URL.Query().Get("cabinetCode"), Status: strings.TrimSpace(request.URL.Query().Get("status"))}
	values, err := a.service.QueryCases(filter)
	if err != nil {
		writeDomainError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": values, "count": len(values)})
}
