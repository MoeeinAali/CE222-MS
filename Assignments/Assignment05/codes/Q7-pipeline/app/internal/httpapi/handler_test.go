package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/ci-demo/internal/httpapi"
)

func do(t *testing.T, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	httpapi.NewRouter().ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	rec := do(t, http.MethodGet, "/health", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %q, want ok", body["status"])
	}
}

func TestAnalyzeHappyPath(t *testing.T) {
	rec := do(t, http.MethodPost, "/v1/analyze", `{"text":"go go gopher","top_n":2}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body struct {
		RequestID string `json:"request_id"`
		Summary   struct {
			Words  int `json:"words"`
			Unique int `json:"unique"`
			Top    []struct {
				Word  string `json:"word"`
				Count int    `json:"count"`
			} `json:"top"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if body.RequestID == "" {
		t.Error("request_id must not be empty")
	}
	if body.Summary.Words != 3 || body.Summary.Unique != 2 {
		t.Errorf("summary = %+v", body.Summary)
	}
	if body.Summary.Top[0].Word != "go" || body.Summary.Top[0].Count != 2 {
		t.Errorf("top[0] = %+v", body.Summary.Top[0])
	}
}

func TestAnalyzeRejectsEmptyText(t *testing.T) {
	if rec := do(t, http.MethodPost, "/v1/analyze", `{"text":""}`); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestAnalyzeRejectsBrokenJSON(t *testing.T) {
	if rec := do(t, http.MethodPost, "/v1/analyze", `{`); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	if rec := do(t, http.MethodGet, "/v1/analyze", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
