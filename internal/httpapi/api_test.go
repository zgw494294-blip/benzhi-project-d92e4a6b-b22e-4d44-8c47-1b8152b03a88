package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/certification"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/credential"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/ledger"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	store, recovery, err := ledger.Open(t.TempDir(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	service, err := certification.NewService(store, recovery, credential.NewIssuer(time.Now), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return New(service).Handler()
}

func TestHealthAndWriteGuards(t *testing.T) {
	handler := testHandler(t)
	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("健康检查返回 %d", health.Code)
	}
	body := []byte(`{"cabinetCode":"CAB","location":"L","cabinetClass":"II","baselineStatus":"qualified"}`)
	missingRole := httptest.NewRequest(http.MethodPost, "/api/v1/certification-cases", bytes.NewReader(body))
	missingRole.Header.Set("Content-Type", "application/json")
	missingRole.Header.Set("Idempotency-Key", "http-key-001")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, missingRole)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("缺少角色返回 %d", recorder.Code)
	}
	wrongType := httptest.NewRequest(http.MethodPost, "/api/v1/certification-cases", bytes.NewReader(body))
	wrongType.Header.Set("Content-Type", "text/plain")
	wrongType.Header.Set("X-Actor-Role", "certification_engineer")
	wrongType.Header.Set("Idempotency-Key", "http-key-002")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, wrongType)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("错误媒体类型返回 %d", recorder.Code)
	}
}
