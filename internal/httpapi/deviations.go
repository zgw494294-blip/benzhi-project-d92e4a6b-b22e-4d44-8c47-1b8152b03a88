package httpapi

import (
	"net/http"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/certification"
)

func (a *API) RemediateHandler(writer http.ResponseWriter, request *http.Request, caseID string) {
	if !requireRole(writer, request, "certification_engineer") {
		return
	}
	var command certification.RemediateCommand
	if err := decodeJSON(writer, request, &command); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	command.IdempotencyKey = idempotencyKey(request)
	var value *certification.CertificationCase
	var err error
	if certification.IsAssignmentAdjustmentAction(command.Action) {
		value, err = a.service.AdjustDeviation(caseID, command)
	} else {
		value, err = a.service.Remediate(caseID, command)
	}
	if err != nil {
		writeDomainError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, value)
}

func (a *API) RetestHandler(writer http.ResponseWriter, request *http.Request, caseID string) {
	if !requireRole(writer, request, "certification_engineer") {
		return
	}
	var command certification.RetestCommand
	if err := decodeJSON(writer, request, &command); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	command.IdempotencyKey = idempotencyKey(request)
	value, err := a.service.Retest(caseID, command)
	if err != nil {
		writeDomainError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, value)
}
