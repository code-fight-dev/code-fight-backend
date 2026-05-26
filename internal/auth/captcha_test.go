package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGoogleRecaptchaVerifierVerifySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("secret"); got != "secret" {
			t.Fatalf("expected secret, got %q", got)
		}
		if got := r.Form.Get("response"); got != "token" {
			t.Fatalf("expected response token, got %q", got)
		}
		if got := r.Form.Get("remoteip"); got != "203.0.113.10" {
			t.Fatalf("expected remote ip, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"score":0.9,"action":"password_reset"}`))
	}))
	defer server.Close()

	verifier := NewGoogleRecaptchaVerifier("secret", server.URL, time.Second, 0.5, "password_reset")
	if err := verifier.Verify(context.Background(), "token", "203.0.113.10"); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestGoogleRecaptchaVerifierVerifyInvalid(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		responseBody   string
		minScore       float64
		expectedAction string
	}{
		{
			name:         "missing token",
			token:        "",
			responseBody: `{"success":true}`,
		},
		{
			name:         "google rejected token",
			token:        "token",
			responseBody: `{"success":false}`,
		},
		{
			name:         "score too low",
			token:        "token",
			responseBody: `{"success":true,"score":0.2}`,
			minScore:     0.5,
		},
		{
			name:           "action mismatch",
			token:          "token",
			responseBody:   `{"success":true,"score":0.9,"action":"login"}`,
			expectedAction: "password_reset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.TrimSpace(tt.token) == "" {
					t.Fatal("server should not be called for missing token")
				}

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			verifier := NewGoogleRecaptchaVerifier("secret", server.URL, time.Second, tt.minScore, tt.expectedAction)
			err := verifier.Verify(context.Background(), tt.token, "")
			if !errors.Is(err, ErrCaptchaInvalid) {
				t.Fatalf("expected ErrCaptchaInvalid, got %v", err)
			}
		})
	}
}

func TestGoogleRecaptchaVerifierVerifyUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	verifier := NewGoogleRecaptchaVerifier("secret", server.URL, time.Second, 0, "")
	err := verifier.Verify(context.Background(), "token", "")
	if !errors.Is(err, ErrCaptchaUnavailable) {
		t.Fatalf("expected ErrCaptchaUnavailable, got %v", err)
	}
}
