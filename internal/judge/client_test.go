package judge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPClientSubmit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/submissions" {
			t.Fatalf("path = %s, want /submissions", r.URL.Path)
		}
		if r.Header.Get("X-Judge-Token") != "api-token" {
			t.Fatalf("X-Judge-Token = %q", r.Header.Get("X-Judge-Token"))
		}

		var req SubmitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.SubmissionID != "submission-1" {
			t.Fatalf("submission id = %q", req.SubmissionID)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"judge_submission_id":"judge-1"}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL+"/", time.Second, " api-token ")
	resp, err := client.Submit(context.Background(), validSubmitRequest())
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}
	if resp.JudgeSubmissionID != "judge-1" {
		t.Fatalf("judge submission id = %q", resp.JudgeSubmissionID)
	}
	if resp.Status != StatusQueued {
		t.Fatalf("status = %q, want queued default", resp.Status)
	}
}

func TestHTTPClientSubmitReturnsServerErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, time.Second, "")
	_, err := client.Submit(context.Background(), validSubmitRequest())
	if err == nil || !strings.Contains(err.Error(), "unexpected status 400") {
		t.Fatalf("Submit error = %v, want status error", err)
	}
}

func TestHTTPClientSubmitValidatesRequestAndResponse(t *testing.T) {
	client := NewHTTPClient("http://example.test", time.Second, "")
	if _, err := client.Submit(context.Background(), SubmitRequest{}); err == nil {
		t.Fatal("Submit returned nil error for invalid request")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"queued"}`))
	}))
	defer server.Close()

	client = NewHTTPClient(server.URL, time.Second, "")
	if _, err := client.Submit(context.Background(), validSubmitRequest()); err == nil || !strings.Contains(err.Error(), "empty judge_submission_id") {
		t.Fatalf("Submit error = %v, want empty judge_submission_id", err)
	}
}

func TestHTTPClientSupportedLanguages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/languages" {
			t.Fatalf("path = %s, want /languages", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"language":"go","version":"1.24"}]`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, time.Second, "")
	languages, err := client.SupportedLanguages(context.Background())
	if err != nil {
		t.Fatalf("SupportedLanguages returned error: %v", err)
	}
	if len(languages) != 1 || languages[0].Language != "go" {
		t.Fatalf("languages = %#v", languages)
	}
}

func TestHTTPClientSupportedLanguagesReturnsDecodeAndStatusErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{name: "status", status: http.StatusServiceUnavailable, body: "down", want: "unexpected status 503"},
		{name: "decode", status: http.StatusOK, body: "{", want: "decode response"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewHTTPClient(server.URL, time.Second, "")
			_, err := client.SupportedLanguages(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("SupportedLanguages error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateSubmitRequest(t *testing.T) {
	tests := []struct {
		name string
		edit func(*SubmitRequest)
		want string
	}{
		{name: "submission id", edit: func(req *SubmitRequest) { req.SubmissionID = "" }, want: "submission_id"},
		{name: "task id", edit: func(req *SubmitRequest) { req.TaskID = "" }, want: "task_id"},
		{name: "language", edit: func(req *SubmitRequest) { req.Language = "" }, want: "language"},
		{name: "language version", edit: func(req *SubmitRequest) { req.LanguageVersion = "" }, want: "language_version"},
		{name: "source", edit: func(req *SubmitRequest) { req.SourceCode = "" }, want: "source_code"},
		{name: "callback url", edit: func(req *SubmitRequest) { req.CallbackURL = "" }, want: "callback_url"},
		{name: "callback token", edit: func(req *SubmitRequest) { req.CallbackToken = "" }, want: "callback_token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validSubmitRequest()
			tt.edit(&req)
			err := validateSubmitRequest(req)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateSubmitRequest error = %v, want %q", err, tt.want)
			}
		})
	}
}

func validSubmitRequest() SubmitRequest {
	return SubmitRequest{
		SubmissionID:    "submission-1",
		TaskID:          "task-1",
		Language:        "go",
		LanguageVersion: "1.24",
		SourceCode:      "package main\nfunc main() {}\n",
		CallbackURL:     "http://backend/internal/judge/callback",
		CallbackToken:   "callback-token",
	}
}
