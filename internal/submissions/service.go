package submissions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/matches"
)

const customCodeRunTestID = "custom"

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

type Notifier interface {
	Publish(userID uuid.UUID, eventType string, payload any)
	PublishMany(userIDs []uuid.UUID, eventType string, payload any)
}

type Metrics interface {
	RecordSubmissionCreated(scope string, language string, status string)
	RecordSubmissionFinished(scope string, language string, status string, verdict string)
	RecordCodeRunCreated(language string, status string)
	RecordCodeRunFinished(language string, status string, verdict string)
}

type Service struct {
	repo           Repository
	taskRepo       TaskRepository
	matchRepo      MatchRepository
	judgeClient    judge.Client
	matchApplier   MatchResultApplier
	notifier       Notifier
	metrics        Metrics
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
	metrics Metrics,
) *Service {
	return &Service{
		repo:           repo,
		taskRepo:       taskRepo,
		matchRepo:      matchRepo,
		judgeClient:    judgeClient,
		matchApplier:   matchApplier,
		metrics:        metrics,
		callbackURL:    callbackURL,
		callbackToken:  callbackToken,
		maxSourceBytes: maxSourceBytes,
	}
}

func (s *Service) SetNotifier(notifier Notifier) {
	s.notifier = notifier
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

	if err := s.validateJudgeLanguage(ctx, input.Language, input.LanguageVersion); err != nil {
		return nil, err
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

	if err := s.persistSubmission(ctx, sub); err != nil {
		return nil, err
	}

	full, tests, err := s.repo.GetByID(ctx, sub.ID)
	if err == nil {
		s.publishSubmission(ctx, "submission.created", full, tests)
	}
	s.recordSubmissionCreated("task", sub.Language, "success")
	return full, err
}

func (s *Service) CreateTaskCodeRun(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	input CreateCodeRunRequest,
) (*CodeRun, error) {
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

	if err := s.validateCodeRunInput(input); err != nil {
		return nil, err
	}

	allowed, err := s.taskRepo.IsLanguageAllowed(ctx, taskID, input.Language, input.LanguageVersion)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrTaskLanguageNotAllowed
	}

	if err := s.validateJudgeLanguage(ctx, input.Language, input.LanguageVersion); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	run := &CodeRun{
		ID:              uuid.NewString(),
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

	if err := s.persistCodeRun(ctx, run, input); err != nil {
		return nil, err
	}

	full, tests, err := s.repo.GetCodeRunByID(ctx, run.ID)
	if err == nil {
		s.publishCodeRun("code_run.created", full, tests)
	}
	s.recordCodeRunCreated(run.Language, "success")
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

	if err := s.validateJudgeLanguage(ctx, input.Language, input.LanguageVersion); err != nil {
		return nil, err
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

	if err := s.persistSubmission(ctx, sub); err != nil {
		return nil, err
	}

	full, tests, err := s.repo.GetByID(ctx, sub.ID)
	if err == nil {
		s.publishSubmission(ctx, "submission.created", full, tests)
	}
	s.recordSubmissionCreated("match", sub.Language, "success")
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

func (s *Service) GetCodeRunByIDForUser(ctx context.Context, codeRunID string, userID uuid.UUID) (*CodeRun, []CodeRunTestResult, error) {
	run, tests, err := s.repo.GetCodeRunByID(ctx, codeRunID)
	if err != nil {
		return nil, nil, err
	}

	if run.UserID != userID.String() {
		return nil, nil, ErrSubmissionNotFound
	}

	return run, tests, nil
}

func (s *Service) HandleJudgeCallback(ctx context.Context, payload judge.CallbackPayload) error {
	sub, err := s.repo.GetByJudgeSubmissionID(ctx, payload.JudgeSubmissionID)
	// Judge can report progress before the acknowledgement reaches Backend.
	if errors.Is(err, ErrSubmissionNotFound) && uuid.Validate(payload.SubmissionID) == nil {
		sub, _, err = s.repo.GetByID(ctx, payload.SubmissionID)
	}
	if err != nil {
		if errors.Is(err, ErrSubmissionNotFound) {
			return s.HandleCodeRunJudgeCallback(ctx, payload)
		}
		return err
	}

	if err := s.validateJudgeCallback(ctx, sub, payload); err != nil {
		return err
	}
	if sub.JudgeSubmissionID == nil {
		if err := s.repo.MarkSentToJudge(ctx, sub.ID, payload.JudgeSubmissionID, JudgeStatusQueued); err != nil {
			return err
		}
	}

	if staleJudgeCallback(sub.Status, sub.StartedAt, payload.StartedAt) {
		// A terminal write can succeed before match processing fails. Replaying
		// that result can apply the match outcome without writing the verdict again.
		if payload.JudgeStatus != JudgeStatusRunning &&
			(sub.Status == StatusFinished || sub.Status == StatusFailed) &&
			sameJudgeAttempt(sub.StartedAt, payload.StartedAt) {
			return s.applySubmissionMatchResult(ctx, sub)
		}
		return nil
	}

	if payload.JudgeStatus == JudgeStatusRunning {
		if err := s.repo.MarkRunning(ctx, sub.ID, payload.JudgeStatus, payload.StartedAt); err != nil {
			if errors.Is(err, ErrStaleJudgeCallback) {
				return nil
			}
			return err
		}
		full, tests, err := s.repo.GetByID(ctx, sub.ID)
		if err == nil {
			s.publishSubmission(ctx, "submission.running", full, tests)
		}
		return nil
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
		if errors.Is(err, ErrStaleJudgeCallback) {
			if sub.MatchID == nil || s.matchApplier == nil {
				return nil
			}
			stored, _, err := s.repo.GetByID(ctx, sub.ID)
			if err != nil {
				return err
			}
			if (stored.Status == StatusFinished || stored.Status == StatusFailed) &&
				sameJudgeAttempt(stored.StartedAt, payload.StartedAt) {
				return s.applySubmissionMatchResult(ctx, stored)
			}
			return nil
		}
		return err
	}
	full, fullTests, err := s.repo.GetByID(ctx, sub.ID)
	if err == nil {
		eventType := "submission.finished"
		if finalStatus == StatusFailed {
			eventType = "submission.failed"
		}
		s.publishSubmission(ctx, eventType, full, fullTests)
	}
	s.recordSubmissionFinished(submissionScope(sub), sub.Language, finalStatus, nullableVerdict(payload.Verdict))

	return s.applySubmissionMatchResult(ctx, sub)
}

func (s *Service) applySubmissionMatchResult(ctx context.Context, sub *Submission) error {
	if sub.MatchID == nil || s.matchApplier == nil {
		return nil
	}
	submissionID, err := uuid.Parse(sub.ID)
	if err != nil {
		return fmt.Errorf("parse submission id: %w", err)
	}
	if err := s.matchApplier.ApplySubmissionResult(ctx, submissionID); err != nil {
		return fmt.Errorf("apply submission result to match: %w", err)
	}

	return nil
}

func (s *Service) HandleCodeRunJudgeCallback(ctx context.Context, payload judge.CallbackPayload) error {
	run, err := s.repo.GetCodeRunByJudgeSubmissionID(ctx, payload.JudgeSubmissionID)
	if errors.Is(err, ErrSubmissionNotFound) && uuid.Validate(payload.SubmissionID) == nil {
		run, _, err = s.repo.GetCodeRunByID(ctx, payload.SubmissionID)
	}
	if err != nil {
		return err
	}

	if err := s.validateCodeRunJudgeCallback(ctx, run, payload); err != nil {
		return err
	}
	if run.JudgeSubmissionID == nil {
		if err := s.repo.MarkCodeRunSentToJudge(ctx, run.ID, payload.JudgeSubmissionID, JudgeStatusQueued); err != nil {
			return err
		}
	}

	if staleJudgeCallback(run.Status, run.StartedAt, payload.StartedAt) {
		return nil
	}

	if payload.JudgeStatus == JudgeStatusRunning {
		if err := s.repo.MarkCodeRunRunning(ctx, run.ID, payload.JudgeStatus, payload.StartedAt); err != nil {
			if errors.Is(err, ErrStaleJudgeCallback) {
				return nil
			}
			return err
		}
		full, tests, err := s.repo.GetCodeRunByID(ctx, run.ID)
		if err == nil {
			s.publishCodeRun("code_run.running", full, tests)
		}
		return nil
	}

	finalStatus := StatusFinished
	if payload.JudgeStatus == JudgeStatusFailed || payload.Verdict == nil {
		finalStatus = StatusFailed
	}

	tests := make([]CodeRunTestResult, 0, len(payload.Tests))
	now := time.Now().UTC()

	for _, t := range payload.Tests {
		testCaseID, testName, isCustom := codeRunResultTestIdentity(t.TestCaseID)
		tests = append(tests, CodeRunTestResult{
			ID:             uuid.NewString(),
			CodeRunID:      run.ID,
			TestCaseID:     testCaseID,
			TestName:       testName,
			IsCustom:       isCustom,
			Status:         t.Status,
			TimeMs:         t.TimeMs,
			MemoryKb:       t.MemoryKb,
			ExitCode:       t.ExitCode,
			CheckerMessage: t.CheckerMessage,
			CreatedAt:      now,
		})
	}

	if err := s.repo.CompleteCodeRun(ctx, run.ID, CompleteInput{
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
	}, tests); err != nil {
		if errors.Is(err, ErrStaleJudgeCallback) {
			return nil
		}
		return err
	}
	full, fullTests, err := s.repo.GetCodeRunByID(ctx, run.ID)
	if err == nil {
		eventType := "code_run.finished"
		if finalStatus == StatusFailed {
			eventType = "code_run.failed"
		}
		s.publishCodeRun(eventType, full, fullTests)
	}
	s.recordCodeRunFinished(run.Language, finalStatus, nullableVerdict(payload.Verdict))
	return nil
}

func (s *Service) recordSubmissionCreated(scope string, language string, status string) {
	if s.metrics != nil {
		s.metrics.RecordSubmissionCreated(scope, strings.TrimSpace(language), status)
	}
}

func (s *Service) recordSubmissionFinished(scope string, language string, status string, verdict string) {
	if s.metrics != nil {
		s.metrics.RecordSubmissionFinished(scope, strings.TrimSpace(language), status, verdict)
	}
}

func (s *Service) recordCodeRunCreated(language string, status string) {
	if s.metrics != nil {
		s.metrics.RecordCodeRunCreated(strings.TrimSpace(language), status)
	}
}

func (s *Service) recordCodeRunFinished(language string, status string, verdict string) {
	if s.metrics != nil {
		s.metrics.RecordCodeRunFinished(strings.TrimSpace(language), status, verdict)
	}
}

func submissionScope(sub *Submission) string {
	if sub != nil && sub.MatchID != nil {
		return "match"
	}

	return "task"
}

func nullableVerdict(verdict *string) string {
	if verdict == nil || strings.TrimSpace(*verdict) == "" {
		return "none"
	}

	return strings.TrimSpace(*verdict)
}

func (s *Service) validateJudgeCallback(ctx context.Context, sub *Submission, payload judge.CallbackPayload) error {
	if payload.SubmissionID != sub.ID || payload.JudgeSubmissionID == "" ||
		(sub.JudgeSubmissionID != nil && *sub.JudgeSubmissionID != payload.JudgeSubmissionID) {
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

func (s *Service) validateCodeRunJudgeCallback(ctx context.Context, run *CodeRun, payload judge.CallbackPayload) error {
	if payload.SubmissionID != run.ID || payload.JudgeSubmissionID == "" ||
		(run.JudgeSubmissionID != nil && *run.JudgeSubmissionID != payload.JudgeSubmissionID) {
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

	allowedTests, err := s.repo.ListJudgeTaskSampleTestCases(ctx, run.TaskID)
	if err != nil {
		return err
	}

	allowed := make(map[string]struct{}, len(allowedTests))
	for _, test := range allowedTests {
		allowed[test.ID] = struct{}{}
	}

	for _, test := range payload.Tests {
		if !isAllowedTestStatus(test.Status) {
			return ErrJudgeCallbackUnauthorized
		}
		if isCustomCodeRunTestID(test.TestCaseID) {
			continue
		}
		if _, ok := allowed[test.TestCaseID]; !ok {
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
		VerdictOutputLimitExceeded,
		VerdictOutputDirectoryLimitExceeded,
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
		VerdictOutputLimitExceeded,
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

func (s *Service) validateCodeRunInput(input CreateCodeRunRequest) error {
	if err := s.validateInput(CreateSubmissionRequest{
		Language:        input.Language,
		LanguageVersion: input.LanguageVersion,
		SourceCode:      input.SourceCode,
	}); err != nil {
		return err
	}

	if input.CustomTest == nil {
		return nil
	}

	if s.maxSourceBytes > 0 {
		size := len(input.CustomTest.Input)
		if input.CustomTest.ExpectedOutput != nil {
			size += len(*input.CustomTest.ExpectedOutput)
		}
		if size > s.maxSourceBytes {
			return ErrInvalidSourceCode
		}
	}

	return nil
}

func (s *Service) submissionDispatch(ctx context.Context, sub *Submission) (judge.SubmitRequest, error) {
	limits, err := s.repo.GetJudgeTaskLimits(ctx, sub.TaskID)
	if err != nil {
		return judge.SubmitRequest{}, fmt.Errorf("get judge task limits: %w", err)
	}

	taskTests, err := s.repo.ListJudgeTaskTestCases(ctx, sub.TaskID)
	if err != nil {
		return judge.SubmitRequest{}, fmt.Errorf("list judge task test cases: %w", err)
	}

	if len(taskTests) == 0 {
		return judge.SubmitRequest{}, fmt.Errorf("task has no active test cases")
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

	return judge.SubmitRequest{
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
	}, nil
}

func (s *Service) sendToJudge(ctx context.Context, sub *Submission) error {
	req, err := s.submissionDispatch(ctx, sub)
	if err != nil {
		return err
	}
	if err := s.repo.Create(ctx, sub); err != nil {
		return err
	}
	resp, err := s.judgeClient.Submit(ctx, req)
	if err != nil {
		if rejected, ok := judgeRejection(err); ok {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			err = errors.Join(err, s.repo.Complete(cleanupCtx, sub.ID, rejected, nil))
		}
		return fmt.Errorf("%w: %w", ErrJudgeSubmitFailed, err)
	}

	if err := s.repo.MarkSentToJudge(ctx, sub.ID, resp.JudgeSubmissionID, resp.Status); err != nil {
		return err
	}

	return nil
}

func (s *Service) codeRunDispatch(ctx context.Context, run *CodeRun, input CreateCodeRunRequest) (judge.SubmitRequest, error) {
	limits, err := s.repo.GetJudgeTaskLimits(ctx, run.TaskID)
	if err != nil {
		return judge.SubmitRequest{}, fmt.Errorf("get judge task limits: %w", err)
	}

	taskTests, err := s.repo.ListJudgeTaskSampleTestCases(ctx, run.TaskID)
	if err != nil {
		return judge.SubmitRequest{}, fmt.Errorf("list judge task sample test cases: %w", err)
	}

	timeMS := limits.TimeLimitMS
	if timeMS <= 0 {
		timeMS = 5000
	}

	memoryMB := limits.MemoryLimitKB / 1024
	if memoryMB <= 0 {
		memoryMB = 256
	}

	judgeTests := make([]judge.SubmitTest, 0, len(taskTests)+1)

	for _, test := range taskTests {
		points := test.Weight
		if points <= 0 {
			points = 1
		}

		judgeTests = append(judgeTests, judge.SubmitTest{
			ID:             test.ID,
			Input:          test.InputBlobRef,
			ExpectedOutput: test.ExpectedBlobRef,
			Points:         points,
		})
	}

	if input.CustomTest != nil {
		expectedOutput := ""
		skipOutputCheck := true
		if input.CustomTest.ExpectedOutput != nil {
			expectedOutput = *input.CustomTest.ExpectedOutput
			skipOutputCheck = false
		}

		judgeTests = append(judgeTests, judge.SubmitTest{
			ID:              customCodeRunTestID,
			Input:           input.CustomTest.Input,
			ExpectedOutput:  expectedOutput,
			Points:          0,
			SkipOutputCheck: skipOutputCheck,
		})
	}

	if len(judgeTests) == 0 {
		return judge.SubmitRequest{}, fmt.Errorf("task has no active sample test cases")
	}

	return judge.SubmitRequest{
		SubmissionID:    run.ID,
		TaskID:          run.TaskID,
		Language:        run.Language,
		LanguageVersion: run.LanguageVersion,
		SourceCode:      run.SourceCode,
		CallbackURL:     s.callbackURL,
		CallbackToken:   s.callbackToken,

		Limits: judge.SubmitLimits{
			TimeMS:   timeMS,
			MemoryMB: memoryMB,
			OutputKB: 64,
		},
		Tests: judgeTests,
	}, nil
}

func (s *Service) sendCodeRunToJudge(ctx context.Context, run *CodeRun, input CreateCodeRunRequest) error {
	req, err := s.codeRunDispatch(ctx, run, input)
	if err != nil {
		return err
	}
	if err := s.repo.CreateCodeRun(ctx, run); err != nil {
		return err
	}
	resp, err := s.judgeClient.Submit(ctx, req)
	if err != nil {
		if rejected, ok := judgeRejection(err); ok {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			err = errors.Join(err, s.repo.CompleteCodeRun(cleanupCtx, run.ID, rejected, nil))
		}
		return fmt.Errorf("%w: %w", ErrJudgeSubmitFailed, err)
	}

	if err := s.repo.MarkCodeRunSentToJudge(ctx, run.ID, resp.JudgeSubmissionID, resp.Status); err != nil {
		return err
	}

	return nil
}

func judgeRejection(err error) (CompleteInput, bool) {
	var rejection *judge.SubmitHTTPError
	if !errors.As(err, &rejection) || rejection.Status != http.StatusBadRequest ||
		(rejection.Code != "validation_error" && rejection.Code != "invalid_json") {
		return CompleteInput{}, false
	}
	message := "Judge rejected this execution request"
	finishedAt := time.Now().UTC()
	return CompleteInput{
		Status: StatusFailed, JudgeStatus: JudgeStatusFailed, ErrorMessage: &message, FinishedAt: &finishedAt,
	}, true
}

func isCustomCodeRunTestID(testID string) bool {
	return strings.TrimSpace(testID) == customCodeRunTestID
}

func codeRunResultTestIdentity(testID string) (*string, *string, bool) {
	if isCustomCodeRunTestID(testID) {
		name := "Custom test"
		return nil, &name, true
	}

	cleaned := strings.TrimSpace(testID)
	if cleaned == "" {
		return nil, nil, false
	}

	return &cleaned, nil, false
}

func (s *Service) validateJudgeLanguage(ctx context.Context, language, version string) error {
	// For queued requests, the task allow-list is checked at admission and
	// Judge validates its runtime when the dispatcher delivers the request.
	if _, durable := s.repo.(dispatchRepository); durable {
		return nil
	}
	supported, err := s.judgeClient.SupportedLanguages(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJudgeSubmitFailed, err)
	}

	language = strings.TrimSpace(language)
	version = strings.TrimSpace(version)

	for _, item := range supported {
		if strings.TrimSpace(item.Language) == language && strings.TrimSpace(item.Version) == version {
			return nil
		}
	}

	return ErrTaskLanguageNotAllowed
}

func (s *Service) publishSubmission(ctx context.Context, eventType string, sub *Submission, tests []SubmissionTestResult) {
	if s.notifier == nil || sub == nil {
		return
	}

	payload := ToSubmissionResponse(sub, tests)
	userID, err := uuid.Parse(sub.UserID)
	if err != nil {
		return
	}

	if sub.MatchID == nil {
		s.notifier.Publish(userID, eventType, payload)
		return
	}

	matchID, err := uuid.Parse(*sub.MatchID)
	if err != nil {
		s.notifier.Publish(userID, eventType, payload)
		return
	}

	match, err := s.matchRepo.GetMatch(ctx, matchID)
	if err != nil {
		s.notifier.Publish(userID, eventType, payload)
		return
	}

	s.notifier.PublishMany([]uuid.UUID{match.Player1ID, match.Player2ID}, eventType, payload)
}

func (s *Service) publishCodeRun(eventType string, run *CodeRun, tests []CodeRunTestResult) {
	if s.notifier == nil || run == nil {
		return
	}

	userID, err := uuid.Parse(run.UserID)
	if err != nil {
		return
	}

	s.notifier.Publish(userID, eventType, ToCodeRunResponse(run, tests))
}

func sameJudgeAttempt(current, incoming *time.Time) bool {
	if current == nil || incoming == nil {
		return current == nil && incoming == nil
	}
	// PostgreSQL stores timestamps at microsecond precision.
	return current.Truncate(time.Microsecond).Equal(incoming.Truncate(time.Microsecond))
}

func staleJudgeCallback(status string, current, incoming *time.Time) bool {
	if current != nil && (incoming == nil || incoming.Before(*current)) {
		return true
	}
	return (status == StatusFinished || status == StatusFailed) &&
		(current == nil || incoming == nil || !incoming.After(*current))
}
