package httpapi

import (
	"net/http"
	"strings"
)

func (a *API) routes() {
	a.mux.HandleFunc("GET /healthz", a.HealthHandler)
	a.mux.HandleFunc("GET /api/v1/certification-cases", a.ListCasesHandler)
	a.mux.HandleFunc("POST /api/v1/certification-cases", a.CreateCaseHandler)
	a.mux.HandleFunc("GET /api/v1/certification-cases/", a.CaseSubresourceHandler)
	a.mux.HandleFunc("POST /api/v1/certification-cases/", a.CaseSubresourceHandler)
	a.mux.HandleFunc("GET /api/v1/credentials/", a.VerifyCredentialHandler)
}

func (a *API) CaseSubresourceHandler(writer http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/api/v1/certification-cases/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeAPIError(writer, http.StatusNotFound, "not_found", "路由不存在")
		return
	}
	caseID := parts[0]
	if request.Method == http.MethodGet && len(parts) == 1 {
		a.GetCaseHandler(writer, request, caseID)
		return
	}
	if request.Method != http.MethodPost || len(parts) < 2 {
		writeAPIError(writer, http.StatusNotFound, "not_found", "路由不存在")
		return
	}
	switch strings.Join(parts[1:], "/") {
	case "plan/lock":
		a.LockPlanHandler(writer, request, caseID)
	case "measurements":
		a.SubmitMeasurementHandler(writer, request, caseID)
	case "deviations/remediate":
		a.RemediateHandler(writer, request, caseID)
	case "retests":
		a.RetestHandler(writer, request, caseID)
	case "review":
		a.ReviewHandler(writer, request, caseID)
	case "credentials":
		a.IssueCredentialHandler(writer, request, caseID)
	default:
		writeAPIError(writer, http.StatusNotFound, "not_found", "路由不存在")
	}
}
