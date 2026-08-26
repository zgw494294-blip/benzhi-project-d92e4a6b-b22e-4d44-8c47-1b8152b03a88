package httpapi

import (
	"net/http"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/certification"
)

type API struct {
	service *certification.Service
	mux     *http.ServeMux
}

func New(service *certification.Service) *API {
	api := &API{service: service, mux: http.NewServeMux()}
	api.routes()
	return api
}

func (a *API) Handler() http.Handler {
	return securityHeaders(a.recoverPanic(a.requestID(a.mux)))
}
