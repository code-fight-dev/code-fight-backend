package submissions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/matches"
)

type Task struct {
	ID       uuid.UUID
	IsActive bool
}

type TaskRepository interface {
	GetByID(ctx context.Context, taskID uuid.UUID) (*Task, error)
	IsLanguageAllowed(ctx context.Context, taskID uuid.UUID, language, languageVersion string) (bool, error)
}

type MatchRepository interface {
	GetMatch(ctx context.Context, matchID uuid.UUID) (matches.Match, error)
}

type MatchResultApplier interface {
	ApplySubmissionResult(ctx context.Context, submissionID uuid.UUID) error
}

type Service struct {
	repo           Repository
	taskRepo       TaskRepository
	matchRepo      MatchRepository
	judgeClient    judge.Client
	matchApplier   MatchResultApplier
	callbackURL    string
	callbackToken  string
	maxSourceBytes int
}

func NewService(
	repo Repository,
	taskRepo TaskRepository,
	matchRepo MatchRepository,
	judgeClient judge.Client,
	matchApplier MatchResultApplier,
	callbackURL string,
	callbackToken string,
	maxSourceBytes int,
) *Service {
	return &Service{
		repo:           repo,
		taskRepo:       taskRepo,
		matchRepo:      matchRepo,
		judgeClient:    judgeClient,
		matchApplier:   matchApplier,
		callbackURL:    callbackURL,
		callbackToken:  callbackToken,
		maxSourceBytes: maxSourceBytes,
	}
}

func (s *Service) CreateTaskSubmission(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	input CreateSubmissionRequest,
) (*Submission, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, ErrTaskNotFound
	}
	if !task.IsActive {
		return nil, ErrTaskInactive
	}

	if err := s.validateInput(input); err != nil {
		return nil, err
	}

	allowed, err := s.taskRepo.IsLanguageAllowed(ctx, taskID, input.Language, input.LanguageVersion)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrTaskLanguageNotAllowed
	}

	supported, err := s.isJudgeLanguageSupported(ctx, input.Language, input.LanguageVersion)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJudgeSubmitFailed, err)
	}
	if !supported {
		return nil, ErrTaskLanguageNotAllowed
	}

	now := time.Now().UTC()
	sub := &Submission{
		ID:              uuid.NewString(),
		MatchID:         nil,
		TaskID:          taskID.String(),
		UserID:          userID.String(),
		Language:        strings.TrimSpace(input.Language),
		LanguageVersion: strings.TrimSpace(input.LanguageVersion),
		SourceCode:      input.SourceCode,
		Status:          StatusQueued,
		PassedTests:     0,
		TotalTests:      0,
		Score:           0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}

	if err := s.sendToJudge(ctx, sub); err != nil {
		return nil, err
	}

	full, _, err := s.repo.GetByID(ctx, sub.ID)
	return full, err
}

func (s *Service) CreateMatchSubmission(
	ctx context.Context,
	userID uuid.UUID,
	matchID uuid.UUID,
	input CreateSubmissionRequest,
) (*Submission, error) {
	match, err := s.matchRepo.GetMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}

	if match.ID == uuid.Nil {
		return nil, ErrMatchNotFound
	}

	if match.Status != matches.MatchStatusRunning {
		return nil, ErrMatchNotStarted
	}

	if match.TaskID == nil || *match.TaskID == uuid.Nil {
		return nil, ErrMatchTaskNotAssigned
	}

	if userID != match.Player1ID && userID != match.Player2ID {
		return nil, ErrUserNotMatchParticipant
	}

	if err := s.validateInput(input); err != nil {
		return nil, err
	}

	allowed, err := s.taskRepo.IsLanguageAllowed(ctx, *match.TaskID, input.Language, input.LanguageVersion)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrTaskLanguageNotAllowed
	}

	supported, err := s.isJudgeLanguageSupported(ctx, input.Language, input.LanguageVersion)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJudgeSubmitFailed, err)
	}
	if !supported {
		return nil, ErrTaskLanguageNotAllowed
	}

	now := time.Now().UTC()
	matchIDStr := match.ID.String()
	sub := &Submission{
		ID:              uuid.NewString(),
		MatchID:         &matchIDStr,
		TaskID:          match.TaskID.String(),
		UserID:          userID.String(),
		Language:        strings.TrimSpace(input.Language),
		LanguageVersion: strings.TrimSpace(input.LanguageVersion),
		SourceCode:      input.SourceCode,
		Status:          StatusQueued,
		PassedTests:     0,
		TotalTests:      0,
		Score:           0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}

	if err := s.sendToJudge(ctx, sub); err != nil {
		return nil, err
	}

	full, _, err := s.repo.GetByID(ctx, sub.ID)
	return full, err
}

func (s *Service) GetByID(ctx context.Context, submissionID string) (*Submission, []SubmissionTestResult, error) {
	return s.repo.GetByID(ctx, submissionID)
}

func (s *Service) GetByIDForUser(ctx context.Context, submissionID string, userID uuid.UUID) (*Submission, []SubmissionTestResult, error) {
	sub, tests, err := s.repo.GetByID(ctx, submissionID)
	if err != nil {
		return nil, nil, err
	}

	if sub.UserID == userID.String() {
		return sub, tests, nil
	}

	if sub.MatchID == nil {
		return nil, nil, ErrSubmissionNotFound
	}

	matchID, err := uuid.Parse(*sub.MatchID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse match id: %w", err)
	}

	match, err := s.matchRepo.GetMatch(ctx, matchID)
	if err != nil {
		return nil, nil, err
	}

	if userID != match.Player1ID && userID != match.Player2ID {
		return nil, nil, ErrSubmissionNotFound
	}

	return sub, tests, nil
}

func (s *Service) HandleJudgeCallback(ctx context.Context, payload judge.CallbackPayload) error {
	sub, err := s.repo.GetByJudgeSubmissionID(ctx, payload.JudgeSubmissionID)
	if err != nil {
		return err
	}

	if err := s.validateJudgeCallback(ctx, sub, payload); err != nil {
		return err
	}

	if sub.Status == StatusFinished || sub.Status == StatusFailed {
		return nil
	}

	if payload.JudgeStatus == JudgeStatusRunning {
		return s.repo.MarkRunning(ctx, sub.ID, payload.JudgeStatus, payload.StartedAt)
	}

	finalStatus := StatusFinished
	if payload.JudgeStatus == JudgeStatusFailed || payload.Verdict == nil {
		finalStatus = StatusFailed
	}

	tests := make([]SubmissionTestResult, 0, len(payload.Tests))
	now := time.Now().UTC()

	for _, t := range payload.Tests {
		tests = append(tests, SubmissionTestResult{
			ID:             uuid.NewString(),
			SubmissionID:   sub.ID,
			TestCaseID:     t.TestCaseID,
			Status:         t.Status,
			TimeMs:         t.TimeMs,
			MemoryKb:       t.MemoryKb,
			ExitCode:       t.ExitCode,
			CheckerMessage: t.CheckerMessage,
			CreatedAt:      now,
		})
	}

	err = s.repo.Complete(ctx, sub.ID, CompleteInput{
		Status:          finalStatus,
		JudgeStatus:     payload.JudgeStatus,
		Verdict:         payload.Verdict,
		ErrorMessage:    payload.ErrorMessage,
		PassedTests:     payload.PassedTests,
		TotalTests:      payload.TotalTests,
		CompileTimeMs:   payload.CompileTimeMs,
		RunTimeMs:       payload.RunTimeMs,
		PeakMemoryKb:    payload.PeakMemoryKb,
		Score:           payload.Score,
		ExitCode:        payload.ExitCode,
		CompileLog:      payload.CompileLog,
		StdoutTruncated: payload.StdoutTruncated,
		StderrTruncated: payload.StderrTruncated,
		StartedAt:       payload.StartedAt,
		FinishedAt:      payload.FinishedAt,
	}, tests)
	if err != nil {
		return err
	}

	if sub.MatchID != nil && s.matchApplier != nil {
		submissionID, err := uuid.Parse(sub.ID)
		if err != nil {
			return fmt.Errorf("parse submission id: %w", err)
		}

		if err := s.matchApplier.ApplySubmissionResult(ctx, submissionID); err != nil {
			return fmt.Errorf("apply submission result to match: %w", err)
		}
	}

	return nil
}

func (s *Service) validateJudgeCallback(ctx context.Context, sub *Submission, payload judge.CallbackPayload) error {
	if payload.SubmissionID != sub.ID {
		return ErrJudgeCallbackUnauthorized
	}

	switch payload.JudgeStatus {
	case JudgeStatusRunning:
		return nil
	case JudgeStatusFinished, JudgeStatusFailed:
	default:
		return ErrJudgeCallbackUnauthorized
	}

	if payload.JudgeStatus == JudgeStatusFinished {
		if payload.Verdict == nil || !isAllowedVerdict(*payload.Verdict) {
			return ErrJudgeCallbackUnauthorized
		}
	} else if payload.Verdict != nil && !isAllowedVerdict(*payload.Verdict) {
		return ErrJudgeCallbackUnauthorized
	}

	if payload.PassedTests < 0 || payload.TotalTests < 0 || payload.Score < 0 {
		return ErrJudgeCallbackUnauthorized
	}
	if payload.PassedTests > payload.TotalTests {
		return ErrJudgeCallbackUnauthorized
	}

	allowedTests, err := s.repo.ListJudgeTaskTestCases(ctx, sub.TaskID)
	if err != nil {
		return err
	}

	allowed := make(map[string]struct{}, len(allowedTests))
	for _, test := range allowedTests {
		allowed[test.ID] = struct{}{}
	}

	for _, test := range payload.Tests {
		if _, ok := allowed[test.TestCaseID]; !ok {
			return ErrJudgeCallbackUnauthorized
		}
		if !isAllowedTestStatus(test.Status) {
			return ErrJudgeCallbackUnauthorized
		}
	}

	return nil
}

func isAllowedVerdict(verdict string) bool {
	switch verdict {
	case VerdictAccepted,
		VerdictWrongAnswer,
		VerdictTimeLimitExceeded,
		VerdictMemoryLimitExceeded,
		VerdictRuntimeError,
		VerdictCompileError,
		VerdictPresentationError,
		VerdictSystemError:
		return true
	default:
		return false
	}
}

func isAllowedTestStatus(status string) bool {
	switch status {
	case VerdictAccepted,
		VerdictWrongAnswer,
		VerdictTimeLimitExceeded,
		VerdictMemoryLimitExceeded,
		VerdictRuntimeError,
		VerdictSystemError,
		"skipped":
		return true
	default:
		return false
	}
}

func (s *Service) validateInput(input CreateSubmissionRequest) error {
	input.Language = strings.TrimSpace(input.Language)
	input.LanguageVersion = strings.TrimSpace(input.LanguageVersion)

	if input.Language == "" || input.LanguageVersion == "" {
		return ErrTaskLanguageNotAllowed
	}
	if strings.TrimSpace(input.SourceCode) == "" {
		return ErrInvalidSourceCode
	}
	if s.maxSourceBytes > 0 && len(input.SourceCode) > s.maxSourceBytes {
		return ErrInvalidSourceCode
	}

	return nil
}

func (s *Service) sendToJudge(ctx context.Context, sub *Submission) error {
	limits, err := s.repo.GetJudgeTaskLimits(ctx, sub.TaskID)
	if err != nil {
		return fmt.Errorf("get judge task limits: %w", err)
	}

	taskTests, err := s.repo.ListJudgeTaskTestCases(ctx, sub.TaskID)
	if err != nil {
		return fmt.Errorf("list judge task test cases: %w", err)
	}

	if len(taskTests) == 0 {
		return fmt.Errorf("task has no active test cases")
	}

	timeMS := limits.TimeLimitMS
	if timeMS <= 0 {
		timeMS = 5000
	}

	memoryMB := limits.MemoryLimitKB / 1024
	if memoryMB <= 0 {
		memoryMB = 256
	}

	judgeTests := make([]judge.SubmitTest, 0, len(taskTests))

	for _, test := range taskTests {
		points := test.Weight
		if points <= 0 {
			points = 1
		}

		testID := test.OrderNo
		if testID <= 0 {
			testID = len(judgeTests) + 1
		}

		judgeTests = append(judgeTests, judge.SubmitTest{
			ID:             test.ID,
			Input:          test.InputBlobRef,
			ExpectedOutput: test.ExpectedBlobRef,
			Points:         points,
		})
	}

	resp, err := s.judgeClient.Submit(ctx, judge.SubmitRequest{
		SubmissionID:    sub.ID,
		TaskID:          sub.TaskID,
		Language:        sub.Language,
		LanguageVersion: sub.LanguageVersion,
		SourceCode:      sub.SourceCode,
		CallbackURL:     s.callbackURL,
		CallbackToken:   s.callbackToken,

		Limits: judge.SubmitLimits{
			TimeMS:   timeMS,
			MemoryMB: memoryMB,
			OutputKB: 64,
		},
		Tests: judgeTests,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJudgeSubmitFailed, err)
	}

	if err := s.repo.MarkSentToJudge(ctx, sub.ID, resp.JudgeSubmissionID, resp.Status); err != nil {
		return err
	}

	return nil
}

func (s *Service) isJudgeLanguageSupported(ctx context.Context, language, version string) (bool, error) {
	supported, err := s.judgeClient.SupportedLanguages(ctx)
	if err != nil {
		return false, err
	}

	language = strings.TrimSpace(language)
	version = strings.TrimSpace(version)

	for _, item := range supported {
		if strings.TrimSpace(item.Language) == language && strings.TrimSpace(item.Version) == version {
			return true, nil
		}
	}

	return false, nil
}
