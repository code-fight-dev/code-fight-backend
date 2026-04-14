package judge

import (
	"context"
	"fmt"
)

type Service struct {
	client   Client
	verifier *Verifier
}

func NewService(client Client, verifier *Verifier) *Service {
	return &Service{
		client:   client,
		verifier: verifier,
	}
}

func (s *Service) Submit(ctx context.Context, req SubmitRequest) (*SubmitResponse, error) {
	if s.client == nil {
		return nil, fmt.Errorf("judge service: client is not configured")
	}

	resp, err := s.client.Submit(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("judge service: %w", err)
	}

	return resp, nil
}

func (s *Service) VerifyCallbackToken(token string) error {
	if s.verifier == nil {
		return fmt.Errorf("judge service: verifier is not configured")
	}

	return s.verifier.VerifyToken(token)
}