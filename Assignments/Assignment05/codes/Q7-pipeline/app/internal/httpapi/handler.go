// Package httpapi exposes textstats over HTTP.
package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"example.com/ci-demo/internal/textstats"
)

const defaultTopN = 5

// Version is stamped at build time with -ldflags.
var Version = "dev"

type analyzeRequest struct {
	Text string `json:"text"`
	TopN int    `json:"top_n"`
}

type analyzeResponse struct {
	RequestID string            `json:"request_id"`
	Summary   textstats.Summary `json:"summary"`
}

// NewRouter wires every route of the service.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /v1/analyze", analyze)
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": Version})
}

func analyze(w http.ResponseWriter, r *http.Request) {
	var req analyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "field 'text' is required"})
		return
	}

	topN := req.TopN
	if topN == 0 {
		topN = defaultTopN
	}

	writeJSON(w, http.StatusOK, analyzeResponse{
		RequestID: uuid.NewString(),
		Summary:   textstats.Analyze(req.Text, topN),
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	payload, err := json.Marshal(body)
	if err != nil {
		http.Error(w, `{"error":"encoding failure"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}
