package judge

import "time"

const (
	StatusPending  = "pending"
	StatusQueued   = "queued"
	StatusRunning  = "running"
	StatusFinished = "finished"
	StatusFailed   = "failed"
)

const (
	VerdictAccepted            = "accepted"
	VerdictWrongAnswer         = "wrong_answer"
	VerdictTimeLimitExceeded   = "time_limit_exceeded"
	VerdictMemoryLimitExceeded = "memory_limit_exceeded"
	VerdictRuntimeError        = "runtime_error"
	VerdictCompileError        = "compile_error"
	VerdictPresentationError   = "presentation_error"
	VerdictSystemError         = "system_error"
)

type SubmitRequest struct {
	SubmissionID    string `json:"submission_id"`
	TaskID          string `json:"task_id"`
	Language        string `json:"language"`
	LanguageVersion string `json:"language_version"`
	SourceCode      string `json:"source_code"`
	CallbackURL     string `json:"callback_url"`
	CallbackToken   string `json:"callback_token"`

	Limits SubmitLimits `json:"limits"`
	Tests  []SubmitTest `json:"tests"`
}

type SubmitLimits struct {
	TimeMS   int `json:"time_ms"`
	MemoryMB int `json:"memory_mb"`
	OutputKB int `json:"output_kb"`
}

type SubmitTest struct {
	ID             string `json:"id"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	Points         int    `json:"points"`
}

type SubmitResponse struct {
	JudgeSubmissionID string `json:"judge_submission_id"`
	Status            string `json:"status"`
}

type SupportedLanguage struct {
	Language string `json:"language"`
	Version  string `json:"version"`
}

type CallbackPayload struct {
	JudgeSubmissionID string               `json:"judge_submission_id"`
	SubmissionID      string               `json:"submission_id"`
	JudgeStatus       string               `json:"judge_status"`
	Verdict           *string              `json:"verdict"`
	ErrorMessage      *string              `json:"error_message"`
	PassedTests       int                  `json:"passed_tests"`
	TotalTests        int                  `json:"total_tests"`
	CompileTimeMs     *int                 `json:"compile_time_ms"`
	RunTimeMs         *int                 `json:"run_time_ms"`
	PeakMemoryKb      *int                 `json:"peak_memory_kb"`
	Score             int                  `json:"score"`
	ExitCode          *int                 `json:"exit_code"`
	CompileLog        *string              `json:"compile_log"`
	StdoutTruncated   *string              `json:"stdout_truncated"`
	StderrTruncated   *string              `json:"stderr_truncated"`
	StartedAt         *time.Time           `json:"started_at"`
	FinishedAt        *time.Time           `json:"finished_at"`
	Tests             []CallbackTestResult `json:"tests"`
}

type CallbackTestResult struct {
	TestCaseID     string  `json:"test_case_id"`
	Status         string  `json:"status"`
	TimeMs         *int    `json:"time_ms"`
	MemoryKb       *int    `json:"memory_kb"`
	ExitCode       *int    `json:"exit_code"`
	CheckerMessage *string `json:"checker_message"`
}
