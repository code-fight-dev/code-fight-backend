package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const googleRecaptchaDefaultVerifyURL = "https://www.google.com/recaptcha/api/siteverify"

type CaptchaVerifier interface {
	Verify(ctx context.Context, token string, remoteIP string) error
}

type GoogleRecaptchaVerifier struct {
	secret         string
	verifyURL      string
	httpClient     *http.Client
	minScore       float64
	expectedAction string
}

func NewGoogleRecaptchaVerifier(secret string, verifyURL string, timeout time.Duration, minScore float64, expectedAction string) *GoogleRecaptchaVerifier {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if strings.TrimSpace(verifyURL) == "" {
		verifyURL = googleRecaptchaDefaultVerifyURL
	}

	return &GoogleRecaptchaVerifier{
		secret:    strings.TrimSpace(secret),
		verifyURL: strings.TrimSpace(verifyURL),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		minScore:       minScore,
		expectedAction: strings.TrimSpace(expectedAction),
	}
}

func (v *GoogleRecaptchaVerifier) Verify(ctx context.Context, token string, remoteIP string) error {
	if v == nil || v.secret == "" {
		return nil
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return ErrCaptchaInvalid
	}

	values := url.Values{}
	values.Set("secret", v.secret)
	values.Set("response", token)
	if remoteIP = strings.TrimSpace(remoteIP); remoteIP != "" {
		values.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.verifyURL, strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("%w: create request: %v", ErrCaptchaUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCaptchaUnavailable, err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("%w: status=%d body=%s", ErrCaptchaUnavailable, response.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Success    bool     `json:"success"`
		Score      float64  `json:"score"`
		Action     string   `json:"action"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("%w: decode response: %v", ErrCaptchaUnavailable, err)
	}

	if !result.Success {
		return ErrCaptchaInvalid
	}
	if v.minScore > 0 && result.Score < v.minScore {
		return ErrCaptchaInvalid
	}
	if v.expectedAction != "" && result.Action != v.expectedAction {
		return ErrCaptchaInvalid
	}

	return nil
}
