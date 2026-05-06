package tasks

import "encoding/json"

type Example struct {
	Title       string  `json:"title"`
	Input       string  `json:"input"`
	Output      string  `json:"output"`
	Explanation *string `json:"explanation,omitempty"`
}

type TestCase struct {
	Name           string `json:"name"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expectedOutput"`
	Locked         bool   `json:"locked,omitempty"`
}

type Challenge struct {
	ID                    string            `json:"id"`
	TaskID                string            `json:"taskId"`
	Slug                  string            `json:"slug"`
	Title                 string            `json:"title"`
	Difficulty            string            `json:"difficulty"`
	Summary               string            `json:"summary"`
	Description           []string          `json:"description"`
	Examples              []Example         `json:"examples"`
	Constraints           []string          `json:"constraints"`
	Notes                 []string          `json:"notes,omitempty"`
	Tags                  []string          `json:"tags"`
	Category              string            `json:"category"`
	Kind                  string            `json:"kind"`
	SupportedLanguages    []string          `json:"supportedLanguages"`
	LanguageVersions      map[string]string `json:"languageVersions"`
	StarterCodeByLanguage map[string]string `json:"starterCodeByLanguage"`
	AcceptanceRate        int               `json:"acceptanceRate"`
	EstimatedMinutes      int               `json:"estimatedMinutes"`
	Attempts              int               `json:"attempts"`
	Popularity            int               `json:"popularity"`
	AcceptedSubmissions    int             `json:"acceptedSubmissions"`
	AcceptedSubmissionRate int             `json:"acceptedSubmissionRate"`
	CreatedAt              string          `json:"createdAt"`
	Progress               string          `json:"progress"`
	TestCases              []TestCase      `json:"testCases"`
	FrontendPayload        json.RawMessage `json:"frontendPayload"`
	TotalSubmissions       int             `json:"totalSubmissions"`
	SubmissionHistory      []TaskSubmission `json:"submissionHistory,omitempty"`
}

type ChallengeListItem struct {
	ID                 string   `json:"id"`
	TaskID             string   `json:"taskId"`
	Slug               string   `json:"slug"`
	Title              string   `json:"title"`
	Difficulty         string   `json:"difficulty"`
	Summary            string   `json:"summary"`
	Tags               []string `json:"tags"`
	Category           string   `json:"category"`
	Kind               string   `json:"kind"`
	SupportedLanguages []string `json:"supportedLanguages"`
	AcceptanceRate     int      `json:"acceptanceRate"`
	EstimatedMinutes   int      `json:"estimatedMinutes"`
	Attempts           int      `json:"attempts"`
	Popularity         int      `json:"popularity"`
	CreatedAt          string   `json:"createdAt"`
	Progress           string   `json:"progress"`
}

type UserTaskStats struct {
	TotalTasks      int `json:"totalTasks"`
	SolvedTasks     int `json:"solvedTasks"`
	InProgressTasks int `json:"inProgressTasks"`
}

type TaskSubmission struct {
	ID                string  `json:"id"`
	MatchID           *string `json:"matchId,omitempty"`
	TaskID            string  `json:"taskId"`
	UserID            string  `json:"userId"`
	Language          string  `json:"language"`
	LanguageVersion   string  `json:"languageVersion"`
	SourceCode        string  `json:"sourceCode"`
	Status            string  `json:"status"`
	Verdict           *string `json:"verdict,omitempty"`
	JudgeSubmissionID *string `json:"judgeSubmissionId,omitempty"`
	JudgeStatus       *string `json:"judgeStatus,omitempty"`
	ErrorMessage      *string `json:"errorMessage,omitempty"`
	PassedTests       int     `json:"passedTests"`
	TotalTests        int     `json:"totalTests"`
	CompileTimeMs     *int    `json:"compileTimeMs,omitempty"`
	RunTimeMs         *int    `json:"runTimeMs,omitempty"`
	PeakMemoryKb      *int    `json:"peakMemoryKb,omitempty"`
	Score             int     `json:"score"`
	ExitCode          *int    `json:"exitCode,omitempty"`
	CompileLog        *string `json:"compileLog,omitempty"`
	StdoutTruncated   *string `json:"stdoutTruncated,omitempty"`
	StderrTruncated   *string `json:"stderrTruncated,omitempty"`
	CreatedAt         string  `json:"createdAt"`
	StartedAt         *string `json:"startedAt,omitempty"`
	FinishedAt        *string `json:"finishedAt,omitempty"`
	UpdatedAt         string  `json:"updatedAt"`
	TestResults       []TaskSubmissionTestResult `json:"testResults"`
}

type TaskSubmissionTestResult struct {
	TestCaseID     string  `json:"testCaseId"`
	Status         string  `json:"status"`
	TimeMs         *int    `json:"timeMs,omitempty"`
	MemoryKb       *int    `json:"memoryKb,omitempty"`
	ExitCode       *int    `json:"exitCode,omitempty"`
	CheckerMessage *string `json:"checkerMessage,omitempty"`
}

