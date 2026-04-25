package submissions

import "time"

type Submission struct {
	ID                string
	MatchID           *string
	TaskID            string
	UserID            string
	Language          string
	LanguageVersion   string
	SourceCode        string
	Status            string
	Verdict           *string
	JudgeSubmissionID *string
	JudgeStatus       *string
	ErrorMessage      *string
	PassedTests       int
	TotalTests        int
	CompileTimeMs     *int
	RunTimeMs         *int
	PeakMemoryKb      *int
	Score             int
	ExitCode          *int
	CompileLog        *string
	StdoutTruncated   *string
	StderrTruncated   *string
	CreatedAt         time.Time
	StartedAt         *time.Time
	FinishedAt        *time.Time
	UpdatedAt         time.Time
}

type SubmissionTestResult struct {
	ID             string
	SubmissionID   string
	TestCaseID     string
	Status         string
	TimeMs         *int
	MemoryKb       *int
	ExitCode       *int
	CheckerMessage *string
	CreatedAt      time.Time
}

const (
	StatusQueued      = "queued"
	StatusSentToJudge = "sent_to_judge"
	StatusRunning     = "running"
	StatusFinished    = "finished"
	StatusFailed      = "failed"
)

const (
	JudgeStatusPending  = "pending"
	JudgeStatusQueued   = "queued"
	JudgeStatusRunning  = "running"
	JudgeStatusFinished = "finished"
	JudgeStatusFailed   = "failed"
)

const (
	VerdictAccepted           = "accepted"
	VerdictWrongAnswer        = "wrong_answer"
	VerdictTimeLimitExceeded  = "time_limit_exceeded"
	VerdictMemoryLimitExceeded = "memory_limit_exceeded"
	VerdictRuntimeError       = "runtime_error"
	VerdictCompileError       = "compile_error"
	VerdictPresentationError  = "presentation_error"
	VerdictSystemError        = "system_error"
)