package httpapi

import (
	"net/http"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/certification"
)

func (a *API) ReviewHandler(writer http.ResponseWriter, request *http.Request, caseID string) {
	if !requireRole(writer, request, "quality_reviewer") {
		return
	}
	var command certification.ReviewCommand
	if err := decodeJSON(writer, request, &command); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	command.IdempotencyKey = idempotencyKey(request)
	value, err := a.service.Review(caseID, command)
	if err != nil {
		writeDomainError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, value)
}
