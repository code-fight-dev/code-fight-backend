package submissions

type CreateSubmissionRequest struct {
	Language        string `json:"language"`
	LanguageVersion string `json:"language_version"`
	SourceCode      string `json:"source_code"`
}

type CreateCodeRunRequest struct {
	Language        string `json:"language"`
	LanguageVersion string `json:"language_version"`
	SourceCode      string `json:"source_code"`
}

type SubmissionTestResultDTO struct {
	TestCaseID     string  `json:"test_case_id"`
	Status         string  `json:"status"`
	TimeMs         *int    `json:"time_ms,omitempty"`
	MemoryKb       *int    `json:"memory_kb,omitempty"`
	ExitCode       *int    `json:"exit_code,omitempty"`
	CheckerMessage *string `json:"checker_message,omitempty"`
}

type CodeRunTestResultDTO struct {
	TestCaseID     string  `json:"test_case_id"`
	Status         string  `json:"status"`
	TimeMs         *int    `json:"time_ms,omitempty"`
	MemoryKb       *int    `json:"memory_kb,omitempty"`
	ExitCode       *int    `json:"exit_code,omitempty"`
	CheckerMessage *string `json:"checker_message,omitempty"`
}

type SubmissionResponse struct {
	ID                string                    `json:"id"`
	MatchID           *string                   `json:"match_id,omitempty"`
	TaskID            string                    `json:"task_id"`
	UserID            string                    `json:"user_id"`
	Language          string                    `json:"language"`
	LanguageVersion   string                    `json:"language_version"`
	Status            string                    `json:"status"`
	Verdict           *string                   `json:"verdict,omitempty"`
	JudgeSubmissionID *string                   `json:"judge_submission_id,omitempty"`
	JudgeStatus       *string                   `json:"judge_status,omitempty"`
	ErrorMessage      *string                   `json:"error_message,omitempty"`
	PassedTests       int                       `json:"passed_tests"`
	TotalTests        int                       `json:"total_tests"`
	CompileTimeMs     *int                      `json:"compile_time_ms,omitempty"`
	RunTimeMs         *int                      `json:"run_time_ms,omitempty"`
	PeakMemoryKb      *int                      `json:"peak_memory_kb,omitempty"`
	Score             int                       `json:"score"`
	ExitCode          *int                      `json:"exit_code,omitempty"`
	CompileLog        *string                   `json:"compile_log,omitempty"`
	StdoutTruncated   *string                   `json:"stdout_truncated,omitempty"`
	StderrTruncated   *string                   `json:"stderr_truncated,omitempty"`
	CreatedAt         string                    `json:"created_at"`
	StartedAt         *string                   `json:"started_at,omitempty"`
	FinishedAt        *string                   `json:"finished_at,omitempty"`
	UpdatedAt         string                    `json:"updated_at"`
	TestResults       []SubmissionTestResultDTO `json:"test_results"`
}

type CodeRunResponse struct {
	ID                string                 `json:"id"`
	TaskID            string                 `json:"task_id"`
	UserID            string                 `json:"user_id"`
	Language          string                 `json:"language"`
	LanguageVersion   string                 `json:"language_version"`
	Status            string                 `json:"status"`
	Verdict           *string                `json:"verdict,omitempty"`
	JudgeSubmissionID *string                `json:"judge_submission_id,omitempty"`
	JudgeStatus       *string                `json:"judge_status,omitempty"`
	ErrorMessage      *string                `json:"error_message,omitempty"`
	PassedTests       int                    `json:"passed_tests"`
	TotalTests        int                    `json:"total_tests"`
	CompileTimeMs     *int                   `json:"compile_time_ms,omitempty"`
	RunTimeMs         *int                   `json:"run_time_ms,omitempty"`
	PeakMemoryKb      *int                   `json:"peak_memory_kb,omitempty"`
	Score             int                    `json:"score"`
	ExitCode          *int                   `json:"exit_code,omitempty"`
	CompileLog        *string                `json:"compile_log,omitempty"`
	StdoutTruncated   *string                `json:"stdout_truncated,omitempty"`
	StderrTruncated   *string                `json:"stderr_truncated,omitempty"`
	CreatedAt         string                 `json:"created_at"`
	StartedAt         *string                `json:"started_at,omitempty"`
	FinishedAt        *string                `json:"finished_at,omitempty"`
	UpdatedAt         string                 `json:"updated_at"`
	TestResults       []CodeRunTestResultDTO `json:"test_results"`
}

func ToSubmissionResponse(s *Submission, tests []SubmissionTestResult) SubmissionResponse {
	resp := SubmissionResponse{
		ID:                s.ID,
		MatchID:           s.MatchID,
		TaskID:            s.TaskID,
		UserID:            s.UserID,
		Language:          s.Language,
		LanguageVersion:   s.LanguageVersion,
		Status:            s.Status,
		Verdict:           s.Verdict,
		JudgeSubmissionID: s.JudgeSubmissionID,
		JudgeStatus:       s.JudgeStatus,
		ErrorMessage:      s.ErrorMessage,
		PassedTests:       s.PassedTests,
		TotalTests:        s.TotalTests,
		CompileTimeMs:     s.CompileTimeMs,
		RunTimeMs:         s.RunTimeMs,
		PeakMemoryKb:      s.PeakMemoryKb,
		Score:             s.Score,
		ExitCode:          s.ExitCode,
		CompileLog:        s.CompileLog,
		StdoutTruncated:   s.StdoutTruncated,
		StderrTruncated:   s.StderrTruncated,
		CreatedAt:         s.CreatedAt.Format(timeLayout),
		UpdatedAt:         s.UpdatedAt.Format(timeLayout),
		TestResults:       make([]SubmissionTestResultDTO, 0, len(tests)),
	}

	if s.StartedAt != nil {
		v := s.StartedAt.Format(timeLayout)
		resp.StartedAt = &v
	}
	if s.FinishedAt != nil {
		v := s.FinishedAt.Format(timeLayout)
		resp.FinishedAt = &v
	}

	for _, t := range tests {
		resp.TestResults = append(resp.TestResults, SubmissionTestResultDTO{
			TestCaseID:     t.TestCaseID,
			Status:         t.Status,
			TimeMs:         t.TimeMs,
			MemoryKb:       t.MemoryKb,
			ExitCode:       t.ExitCode,
			CheckerMessage: t.CheckerMessage,
		})
	}

	return resp
}

func ToCodeRunResponse(run *CodeRun, tests []CodeRunTestResult) CodeRunResponse {
	resp := CodeRunResponse{
		ID:                run.ID,
		TaskID:            run.TaskID,
		UserID:            run.UserID,
		Language:          run.Language,
		LanguageVersion:   run.LanguageVersion,
		Status:            run.Status,
		Verdict:           run.Verdict,
		JudgeSubmissionID: run.JudgeSubmissionID,
		JudgeStatus:       run.JudgeStatus,
		ErrorMessage:      run.ErrorMessage,
		PassedTests:       run.PassedTests,
		TotalTests:        run.TotalTests,
		CompileTimeMs:     run.CompileTimeMs,
		RunTimeMs:         run.RunTimeMs,
		PeakMemoryKb:      run.PeakMemoryKb,
		Score:             run.Score,
		ExitCode:          run.ExitCode,
		CompileLog:        run.CompileLog,
		StdoutTruncated:   run.StdoutTruncated,
		StderrTruncated:   run.StderrTruncated,
		CreatedAt:         run.CreatedAt.Format(timeLayout),
		UpdatedAt:         run.UpdatedAt.Format(timeLayout),
		TestResults:       make([]CodeRunTestResultDTO, 0, len(tests)),
	}

	if run.StartedAt != nil {
		v := run.StartedAt.Format(timeLayout)
		resp.StartedAt = &v
	}
	if run.FinishedAt != nil {
		v := run.FinishedAt.Format(timeLayout)
		resp.FinishedAt = &v
	}

	for _, t := range tests {
		resp.TestResults = append(resp.TestResults, CodeRunTestResultDTO{
			TestCaseID:     t.TestCaseID,
			Status:         t.Status,
			TimeMs:         t.TimeMs,
			MemoryKb:       t.MemoryKb,
			ExitCode:       t.ExitCode,
			CheckerMessage: t.CheckerMessage,
		})
	}

	return resp
}
