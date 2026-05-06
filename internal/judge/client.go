package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client interface {
	Submit(ctx context.Context, req SubmitRequest) (*SubmitResponse, error)
	SupportedLanguages(ctx context.Context) ([]SupportedLanguage, error)
}

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *HTTPClient) Submit(ctx context.Context, req SubmitRequest) (*SubmitResponse, error) {
	if err := validateSubmitRequest(req); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("judge submit: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/submissions",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("judge submit: build request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("judge submit: send request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("judge submit: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("judge submit: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(rawBody)))
	}

	var out SubmitResponse
	if err := json.Unmarshal(rawBody, &out); err != nil {
		return nil, fmt.Errorf("judge submit: decode response: %w", err)
	}

	if out.JudgeSubmissionID == "" {
		return nil, fmt.Errorf("judge submit: empty judge_submission_id in response")
	}
	if out.Status == "" {
		out.Status = StatusQueued
	}

	return &out, nil
}

func (c *HTTPClient) SupportedLanguages(ctx context.Context) ([]SupportedLanguage, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/languages", nil)
	if err != nil {
		return nil, fmt.Errorf("judge languages: build request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("judge languages: send request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("judge languages: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("judge languages: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(rawBody)))
	}

	var out []SupportedLanguage
	if err := json.Unmarshal(rawBody, &out); err != nil {
		return nil, fmt.Errorf("judge languages: decode response: %w", err)
	}

	return out, nil
}

func validateSubmitRequest(req SubmitRequest) error {
	switch {
	case strings.TrimSpace(req.SubmissionID) == "":
		return fmt.Errorf("judge submit: submission_id is required")
	case strings.TrimSpace(req.TaskID) == "":
		return fmt.Errorf("judge submit: task_id is required")
	case strings.TrimSpace(req.Language) == "":
		return fmt.Errorf("judge submit: language is required")
	case strings.TrimSpace(req.LanguageVersion) == "":
		return fmt.Errorf("judge submit: language_version is required")
	case strings.TrimSpace(req.SourceCode) == "":
		return fmt.Errorf("judge submit: source_code is required")
	case strings.TrimSpace(req.CallbackURL) == "":
		return fmt.Errorf("judge submit: callback_url is required")
	case strings.TrimSpace(req.CallbackToken) == "":
		return fmt.Errorf("judge submit: callback_token is required")
	default:
		return nil
	}
}
