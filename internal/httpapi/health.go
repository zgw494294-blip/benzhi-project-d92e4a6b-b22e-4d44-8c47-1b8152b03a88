package httpapi

import "net/http"

func (a *API) HealthHandler(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "洁净工作台现场认证与启用放行服务",
	})
}
