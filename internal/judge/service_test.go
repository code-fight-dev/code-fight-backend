package judge

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestServiceSubmitDelegatesToClient(t *testing.T) {
	client := &fakeClient{
		submitResponse: &SubmitResponse{
			JudgeSubmissionID: "judge-1",
			Status:            StatusQueued,
		},
	}
	service := NewService(client, NewVerifier("callback-token"))

	resp, err := service.Submit(context.Background(), SubmitRequest{
		SubmissionID: "submission-1",
		TaskID:       "task-1",
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}
	if resp.JudgeSubmissionID != "judge-1" {
		t.Fatalf("judge submission id = %q, want judge-1", resp.JudgeSubmissionID)
	}
	if len(client.submits) != 1 {
		t.Fatalf("client submits = %d, want 1", len(client.submits))
	}
	if client.submits[0].SubmissionID != "submission-1" {
		t.Fatalf("submission id = %q, want submission-1", client.submits[0].SubmissionID)
	}
}

func TestServiceSubmitWrapsClientError(t *testing.T) {
	clientErr := errors.New("network down")
	service := NewService(&fakeClient{submitErr: clientErr}, nil)

	_, err := service.Submit(context.Background(), SubmitRequest{})
	if !errors.Is(err, clientErr) {
		t.Fatalf("Submit error = %v, want wrapped client error", err)
	}
	if !strings.Contains(err.Error(), "judge service") {
		t.Fatalf("Submit error = %q, want service context", err.Error())
	}
}

func TestServiceSubmitRequiresClient(t *testing.T) {
	service := NewService(nil, nil)

	_, err := service.Submit(context.Background(), SubmitRequest{})
	if err == nil {
		t.Fatal("Submit returned nil error, want configuration error")
	}
}

func TestServiceSupportedLanguagesDelegatesToClient(t *testing.T) {
	service := NewService(&fakeClient{
		supportedLanguages: []SupportedLanguage{
			{Language: "go", Version: "1.24"},
			{Language: "python", Version: "3.12"},
		},
	}, nil)

	languages, err := service.SupportedLanguages(context.Background())
	if err != nil {
		t.Fatalf("SupportedLanguages returned error: %v", err)
	}
	if len(languages) != 2 {
		t.Fatalf("languages = %d, want 2", len(languages))
	}
	if languages[0].Language != "go" {
		t.Fatalf("first language = %q, want go", languages[0].Language)
	}
}

func TestServiceSupportedLanguagesErrors(t *testing.T) {
	clientErr := errors.New("languages unavailable")
	service := NewService(&fakeClient{supportedErr: clientErr}, nil)

	_, err := service.SupportedLanguages(context.Background())
	if !errors.Is(err, clientErr) {
		t.Fatalf("SupportedLanguages error = %v, want wrapped client error", err)
	}
	if !strings.Contains(err.Error(), "judge service") {
		t.Fatalf("SupportedLanguages error = %q, want service context", err.Error())
	}

	service = NewService(nil, nil)
	if _, err := service.SupportedLanguages(context.Background()); err == nil {
		t.Fatal("SupportedLanguages returned nil error, want configuration error")
	}
}

func TestServiceVerifyCallbackToken(t *testing.T) {
	service := NewService(nil, NewVerifier(" callback-token "))

	if err := service.VerifyCallbackToken("callback-token"); err != nil {
		t.Fatalf("VerifyCallbackToken returned error: %v", err)
	}
	if err := service.VerifyCallbackToken("wrong-token"); !errors.Is(err, ErrUnauthorizedCallback) {
		t.Fatalf("VerifyCallbackToken error = %v, want ErrUnauthorizedCallback", err)
	}
}

func TestServiceVerifyCallbackTokenRequiresVerifier(t *testing.T) {
	service := NewService(nil, nil)

	err := service.VerifyCallbackToken("callback-token")
	if err == nil {
		t.Fatal("VerifyCallbackToken returned nil error, want configuration error")
	}
}

type fakeClient struct {
	submits            []SubmitRequest
	submitResponse     *SubmitResponse
	submitErr          error
	supportedLanguages []SupportedLanguage
	supportedErr       error
}

func (c *fakeClient) Submit(_ context.Context, req SubmitRequest) (*SubmitResponse, error) {
	c.submits = append(c.submits, req)
	if c.submitErr != nil {
		return nil, c.submitErr
	}
	return c.submitResponse, nil
}

func (c *fakeClient) SupportedLanguages(context.Context) ([]SupportedLanguage, error) {
	if c.supportedErr != nil {
		return nil, c.supportedErr
	}
	return c.supportedLanguages, nil
}
