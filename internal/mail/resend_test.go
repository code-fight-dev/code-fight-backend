package mail

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestNewResendSender(t *testing.T) {
	sender, err := NewResendSender(" api-key ", " Code Fight <noreply@example.com> ")
	if err != nil {
		t.Fatalf("NewResendSender returned error: %v", err)
	}
	if sender.apiKey != "api-key" {
		t.Fatalf("api key = %q", sender.apiKey)
	}
	if sender.from != "\"Code Fight\" <noreply@example.com>" {
		t.Fatalf("from = %q", sender.from)
	}

	if _, err := NewResendSender("api-key", "bad address"); err == nil {
		t.Fatal("NewResendSender returned nil error for bad from")
	}
}

func TestResendSenderSendPasswordResetCode(t *testing.T) {
	transport := &captureTransport{
		statusCode: http.StatusAccepted,
		body:       "ok",
	}
	sender := &ResendSender{
		apiKey:     "api-key",
		from:       "noreply@example.com",
		httpClient: &http.Client{Transport: transport},
	}

	err := sender.SendPasswordResetCode(context.Background(), "user@example.com", "123456")
	if err != nil {
		t.Fatalf("SendPasswordResetCode returned error: %v", err)
	}
	if transport.method != http.MethodPost {
		t.Fatalf("method = %q", transport.method)
	}
	if transport.auth != "Bearer api-key" {
		t.Fatalf("auth = %q", transport.auth)
	}
	if !strings.Contains(transport.body, "123456") || !strings.Contains(transport.body, "user@example.com") {
		t.Fatalf("body = %s", transport.body)
	}
}

func TestResendSenderSendPasswordResetCodeErrors(t *testing.T) {
	unconfigured := &ResendSender{}
	if err := unconfigured.SendPasswordResetCode(context.Background(), "user@example.com", "123456"); err == nil {
		t.Fatal("SendPasswordResetCode returned nil error for unconfigured sender")
	}

	sender := &ResendSender{
		apiKey:     "api-key",
		from:       "noreply@example.com",
		httpClient: &http.Client{Transport: &captureTransport{statusCode: http.StatusBadGateway, body: "down"}},
	}
	if err := sender.SendPasswordResetCode(context.Background(), "user@example.com", "123456"); err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("SendPasswordResetCode error = %v, want 502", err)
	}
}

func TestLogSenderSendPasswordResetCode(t *testing.T) {
	if err := NewLogSender(nil).SendPasswordResetCode(context.Background(), "user@example.com", "123456"); err != nil {
		t.Fatalf("nil logger returned error: %v", err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	if err := NewLogSender(logger).SendPasswordResetCode(context.Background(), "user@example.com", "123456"); err != nil {
		t.Fatalf("logger returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "user@example.com") || !strings.Contains(buf.String(), "123456") {
		t.Fatalf("log output = %s", buf.String())
	}
}

type captureTransport struct {
	statusCode int
	body       string
	method     string
	auth       string
	requestURL string
}

func (t *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	t.method = req.Method
	t.auth = req.Header.Get("Authorization")
	t.requestURL = req.URL.String()
	t.body = string(raw)
	return &http.Response{
		StatusCode: t.statusCode,
		Body:       io.NopCloser(strings.NewReader(t.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}
