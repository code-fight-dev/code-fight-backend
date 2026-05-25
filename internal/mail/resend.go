package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	netmail "net/mail"
	"strings"
	"time"
)

const resendEmailSendEndpoint = "https://api.resend.com/emails"

type PasswordResetSender interface {
	SendPasswordResetCode(ctx context.Context, to string, code string) error
}

type ResendSender struct {
	apiKey     string
	from       string
	httpClient *http.Client
}

func NewResendSender(apiKey string, from string) (*ResendSender, error) {
	parsedFrom, err := netmail.ParseAddress(strings.TrimSpace(from))
	if err != nil {
		return nil, fmt.Errorf("parse EMAIL_FROM: %w", err)
	}

	return &ResendSender{
		apiKey: strings.TrimSpace(apiKey),
		from:   parsedFrom.String(),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (s *ResendSender) SendPasswordResetCode(ctx context.Context, to string, code string) error {
	if s.apiKey == "" || s.from == "" {
		return fmt.Errorf("resend sender is not configured")
	}

	payload := resendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: "Code Fight password reset code",
		Text:    "Your Code Fight password reset code is " + code + ". It expires in 10 minutes.",
		HTML:    "<p>Your Code Fight password reset code is <strong>" + code + "</strong>.</p><p>It expires in 10 minutes.</p>",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode resend payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendEmailSendEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "code-fight-backend/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send resend request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("resend returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

type resendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
	HTML    string   `json:"html"`
}

type LogSender struct {
	logger *slog.Logger
}

func NewLogSender(logger *slog.Logger) *LogSender {
	return &LogSender{logger: logger}
}

func (s *LogSender) SendPasswordResetCode(ctx context.Context, to string, code string) error {
	if s.logger != nil {
		s.logger.Info("password reset code generated", "email", to, "code", code)
	}

	return nil
}
