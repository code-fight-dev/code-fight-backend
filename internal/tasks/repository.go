package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type challengePayload struct {
	ID                    string            `json:"id"`
	Title                 string            `json:"title"`
	Summary               string            `json:"summary"`
	Description           []string          `json:"description"`
	Examples              []Example         `json:"examples"`
	Constraints           []string          `json:"constraints"`
	Notes                 []string          `json:"notes"`
	Tags                  []string          `json:"tags"`
	Category              string            `json:"category"`
	Kind                  string            `json:"kind"`
	SupportedLanguages    []string          `json:"supportedLanguages"`
	StarterCodeByLanguage map[string]string `json:"starterCodeByLanguage"`
	AcceptanceRate        int               `json:"acceptanceRate"`
	EstimatedMinutes      int               `json:"estimatedMinutes"`
	Attempts              int               `json:"attempts"`
	Popularity            int               `json:"popularity"`
	CreatedAt             string            `json:"createdAt"`
	Progress              string            `json:"progress"`
}

type challengeRow struct {
	TaskID              string
	Slug                string
	Title               string
	Difficulty          string
	CreatedAt           time.Time
	PayloadRaw          []byte
	Languages           []string
	AcceptedSubmissions int
	TotalSubmissions    int
}

type languageVersionRow struct {
	Language        string
	LanguageVersion string
}

func (r *Repository) ListChallenges(ctx context.Context) ([]ChallengeListItem, error) {
	const q = `
select
	t.id::text,
	t.slug,
	t.title,
	t.difficulty,
	t.created_at,
	coalesce(t.frontend_payload, '{}'::jsonb)::text,
	coalesce(
		array_agg(distinct tl.language) filter (where tl.is_enabled),
		array[]::text[]
	),
	coalesce(stats.accepted_submissions, 0)::int,
	coalesce(stats.total_submissions, 0)::int
from tasks t
left join task_languages tl on tl.task_id = t.id
left join lateral (
	select
		count(*) filter (
			where s.status = 'finished'
			  and s.verdict = 'accepted'
		) as accepted_submissions,
		count(*) as total_submissions
	from submissions s
	where s.task_id = t.id
) stats on true
where t.is_active = true
group by t.id, stats.accepted_submissions, stats.total_submissions
order by t.created_at desc
`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ChallengeListItem, 0)

	for rows.Next() {
		row, err := scanChallengeRow(rows)
		if err != nil {
			return nil, err
		}

		payload := decodePayload(row.PayloadRaw)

		item := ChallengeListItem{
			ID:                 payloadID(payload, row.Slug),
			TaskID:             row.TaskID,
			Slug:               row.Slug,
			Title:              fallbackString(payload.Title, row.Title),
			Difficulty:         toFrontendDifficulty(row.Difficulty),
			Summary:            payload.Summary,
			Tags:               payload.Tags,
			Category:           payload.Category,
			Kind:               payloadKind(payload),
			SupportedLanguages: normalizeLanguages(payloadLanguagesAllowedByTask(payload, row.Languages)),
			AcceptanceRate:     acceptanceRate(row.AcceptedSubmissions, row.TotalSubmissions),
			EstimatedMinutes:   payload.EstimatedMinutes,
			Attempts:           row.TotalSubmissions,
			Popularity:         payload.Popularity,
			CreatedAt:          payloadCreatedAt(payload, row.CreatedAt),
			Progress:           payloadProgress(payload),
			FrontendPayload:        normalizedPayloadRaw(row.PayloadRaw),
			AcceptedSubmissions:    row.AcceptedSubmissions,
			AcceptedSubmissionRate: acceptanceRate(row.AcceptedSubmissions, row.TotalSubmissions),
			TotalSubmissions:       row.TotalSubmissions,
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repository) GetUserTaskStats(ctx context.Context, userID string) (UserTaskStats, error) {
	const q = `
with user_task_status as (
	select
		task_id,
		bool_or(status = 'finished' and verdict = 'accepted') as solved
	from submissions
	where user_id = $1
	group by task_id
)
select
	count(*) filter (where solved)::int,
	count(*) filter (where not solved)::int
from user_task_status
`

	var stats UserTaskStats
	if err := r.db.QueryRow(ctx, q, userID).Scan(&stats.SolvedTasks, &stats.InProgressTasks); err != nil {
		return UserTaskStats{}, err
	}

	return stats, nil
}

func (r *Repository) ListUserTaskSubmissions(
	ctx context.Context,
	taskID string,
	userID string,
) ([]TaskSubmission, error) {
	const q = `
select
	id::text,
	match_id::text,
	task_id::text,
	user_id::text,
	language,
	language_version,
	source_code,
	status,
	verdict,
	judge_submission_id,
	judge_status,
	error_message,
	passed_tests,
	total_tests,
	compile_time_ms,
	run_time_ms,
	peak_memory_kb,
	score,
	exit_code,
	compile_log,
	stdout_truncated,
	stderr_truncated,
	created_at,
	started_at,
	finished_at,
	updated_at,
	coalesce(test_results.items, '[]'::jsonb)::text
from submissions
left join lateral (
	select jsonb_agg(
		jsonb_build_object(
			'testCaseId', str.test_case_id::text,
			'status', str.status,
			'timeMs', str.time_ms,
			'memoryKb', str.memory_kb,
			'exitCode', str.exit_code,
			'checkerMessage', str.checker_message
		)
		order by str.created_at asc
	) as items
	from submission_test_results str
where str.submission_id = submissions.id
) test_results on true
where submissions.task_id = $1
  and submissions.user_id = $2
order by submissions.created_at desc
limit 100
`

	rows, err := r.db.Query(ctx, q, taskID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TaskSubmission, 0)
	for rows.Next() {
		var item TaskSubmission
		var createdAt time.Time
		var updatedAt time.Time
		var startedAt *time.Time
		var finishedAt *time.Time
		var testResultsText string

		if err := rows.Scan(
			&item.ID,
			&item.MatchID,
			&item.TaskID,
			&item.UserID,
			&item.Language,
			&item.LanguageVersion,
			&item.SourceCode,
			&item.Status,
			&item.Verdict,
			&item.JudgeSubmissionID,
			&item.JudgeStatus,
			&item.ErrorMessage,
			&item.PassedTests,
			&item.TotalTests,
			&item.CompileTimeMs,
			&item.RunTimeMs,
			&item.PeakMemoryKb,
			&item.Score,
			&item.ExitCode,
			&item.CompileLog,
			&item.StdoutTruncated,
			&item.StderrTruncated,
			&createdAt,
			&startedAt,
			&finishedAt,
			&updatedAt,
			&testResultsText,
		); err != nil {
			return nil, err
		}

		item.CreatedAt = formatTime(createdAt)
		item.StartedAt = formatOptionalTime(startedAt)
		item.FinishedAt = formatOptionalTime(finishedAt)
		item.UpdatedAt = formatTime(updatedAt)
		item.TestResults = decodeSubmissionTestResults([]byte(testResultsText))
		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *Repository) GetChallengeBySlug(ctx context.Context, slug string) (*Challenge, error) {
	const q = `
select
	t.id::text,
	t.slug,
	t.title,
	t.difficulty,
	t.created_at,
	coalesce(t.frontend_payload, '{}'::jsonb)::text,
	coalesce(
		array_agg(distinct tl.language) filter (where tl.is_enabled),
		array[]::text[]
	),
	(
		select count(*)::int
		from submissions s
		where s.task_id = t.id
		  and s.status = 'finished'
		  and s.verdict = 'accepted'
	),
	(
		select count(*)::int
		from submissions s
		where s.task_id = t.id
	)
from tasks t
left join task_languages tl on tl.task_id = t.id
where t.slug = $1
  and t.is_active = true
group by t.id
`

	row := r.db.QueryRow(ctx, q, slug)
	challengeRowValue, err := scanChallengeDetailRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	payload := decodePayload(challengeRowValue.PayloadRaw)
	languages, versions, err := r.listLanguageVersions(ctx, challengeRowValue.TaskID)
	if err != nil {
		return nil, err
	}

	tests, err := r.listSampleTests(ctx, challengeRowValue.TaskID)
	if err != nil {
		return nil, err
	}

	out := &Challenge{
		ID:                    payloadID(payload, challengeRowValue.Slug),
		TaskID:                challengeRowValue.TaskID,
		Slug:                  challengeRowValue.Slug,
		Title:                 fallbackString(payload.Title, challengeRowValue.Title),
		Difficulty:            toFrontendDifficulty(challengeRowValue.Difficulty),
		Summary:               payload.Summary,
		Description:           payload.Description,
		Examples:              payload.Examples,
		Constraints:           payload.Constraints,
		Notes:                 payload.Notes,
		Tags:                  payload.Tags,
		Category:              payload.Category,
		Kind:                  payloadKind(payload),
		SupportedLanguages:    normalizeLanguages(payloadLanguagesAllowedByTask(payload, languages)),
		LanguageVersions:      versions,
		StarterCodeByLanguage: payload.StarterCodeByLanguage,
		AcceptanceRate:        acceptanceRate(challengeRowValue.AcceptedSubmissions, challengeRowValue.TotalSubmissions),
		EstimatedMinutes:      payload.EstimatedMinutes,
		Attempts:              challengeRowValue.TotalSubmissions,
		Popularity:            payload.Popularity,
		AcceptedSubmissions:    challengeRowValue.AcceptedSubmissions,
		AcceptedSubmissionRate: acceptanceRate(challengeRowValue.AcceptedSubmissions, challengeRowValue.TotalSubmissions),
		TotalSubmissions:       challengeRowValue.TotalSubmissions,
		CreatedAt:              payloadCreatedAt(payload, challengeRowValue.CreatedAt),
		Progress:               payloadProgress(payload),
		TestCases:              tests,
		FrontendPayload:        normalizedPayloadRaw(challengeRowValue.PayloadRaw),
	}

	return out, nil
}

func (r *Repository) listLanguageVersions(
	ctx context.Context,
	taskID string,
) ([]string, map[string]string, error) {
	const q = `
select language, language_version
from task_languages
where task_id = $1
  and is_enabled = true
order by language asc, language_version desc
`

	rows, err := r.db.Query(ctx, q, taskID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	versionMap := make(map[string]string)
	languageSet := make(map[string]struct{})

	for rows.Next() {
		var item languageVersionRow
		if err := rows.Scan(&item.Language, &item.LanguageVersion); err != nil {
			return nil, nil, err
		}

		if _, ok := versionMap[item.Language]; !ok {
			versionMap[item.Language] = item.LanguageVersion
		}
		languageSet[item.Language] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	languages := make([]string, 0, len(languageSet))
	for language := range languageSet {
		languages = append(languages, language)
	}

	slices.Sort(languages)

	return languages, versionMap, nil
}

func (r *Repository) listSampleTests(ctx context.Context, taskID string) ([]TestCase, error) {
	const q = `
select order_no, input_blob_ref, expected_blob_ref
from task_test_cases
where task_id = $1
  and is_active = true
  and kind = 'sample'
order by order_no asc
`

	rows, err := r.db.Query(ctx, q, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tests := make([]TestCase, 0)

	for rows.Next() {
		var orderNo int
		var input string
		var expected string

		if err := rows.Scan(&orderNo, &input, &expected); err != nil {
			return nil, err
		}

		displayNo := len(tests) + 1
		tests = append(tests, TestCase{
			Name:           fmt.Sprintf("Case %d", displayNo),
			Input:          input,
			ExpectedOutput: expected,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tests, nil
}

func scanChallengeRow(row interface {
	Scan(dest ...any) error
}) (challengeRow, error) {
	var out challengeRow
	var payloadText string

	err := row.Scan(
		&out.TaskID,
		&out.Slug,
		&out.Title,
		&out.Difficulty,
		&out.CreatedAt,
		&payloadText,
		&out.Languages,
		&out.AcceptedSubmissions,
		&out.TotalSubmissions,
	)
	if err != nil {
		return challengeRow{}, err
	}

	out.PayloadRaw = []byte(payloadText)

	return out, nil
}

func scanChallengeDetailRow(row interface {
	Scan(dest ...any) error
}) (challengeRow, error) {
	var out challengeRow
	var payloadText string

	err := row.Scan(
		&out.TaskID,
		&out.Slug,
		&out.Title,
		&out.Difficulty,
		&out.CreatedAt,
		&payloadText,
		&out.Languages,
		&out.AcceptedSubmissions,
		&out.TotalSubmissions,
	)
	if err != nil {
		return challengeRow{}, err
	}

	out.PayloadRaw = []byte(payloadText)

	return out, nil
}

func decodePayload(raw []byte) challengePayload {
	var payload challengePayload
	_ = json.Unmarshal(raw, &payload)
	return payload
}

func decodeSubmissionTestResults(raw []byte) []TaskSubmissionTestResult {
	var out []TaskSubmissionTestResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return []TaskSubmissionTestResult{}
	}
	if out == nil {
		return []TaskSubmissionTestResult{}
	}
	return out
}

func normalizedPayloadRaw(raw []byte) json.RawMessage {
	if len(raw) == 0 || !json.Valid(raw) {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(raw)
}

func acceptanceRate(accepted, total int) int {
	if total <= 0 {
		return 0
	}
	return int(float64(accepted) / float64(total) * 100)
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := formatTime(*value)
	return &formatted
}

func payloadID(payload challengePayload, slug string) string {
	if strings.TrimSpace(payload.ID) != "" {
		return payload.ID
	}
	return slug
}

func payloadKind(payload challengePayload) string {
	kind := strings.TrimSpace(payload.Kind)
	if kind == "" {
		return "algorithmic"
	}
	return kind
}

func payloadProgress(payload challengePayload) string {
	progress := strings.TrimSpace(payload.Progress)
	if progress == "" {
		return "not-started"
	}
	return progress
}

func payloadCreatedAt(payload challengePayload, fallback time.Time) string {
	if strings.TrimSpace(payload.CreatedAt) != "" {
		return payload.CreatedAt
	}
	return fallback.UTC().Format("2006-01-02")
}

func payloadLanguages(payload challengePayload, fallback []string) []string {
	if len(payload.SupportedLanguages) > 0 {
		return payload.SupportedLanguages
	}
	return fallback
}

func payloadLanguagesAllowedByTask(payload challengePayload, taskLanguages []string) []string {
	payloadLanguagesValue := payloadLanguages(payload, taskLanguages)
	if len(taskLanguages) == 0 {
		return []string{}
	}
	if len(payloadLanguagesValue) == 0 {
		return payloadLanguagesValue
	}

	allowed := make(map[string]struct{}, len(taskLanguages))
	for _, language := range taskLanguages {
		normalized := strings.TrimSpace(language)
		if normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}

	out := make([]string, 0, len(payloadLanguagesValue))
	for _, language := range payloadLanguagesValue {
		normalized := strings.TrimSpace(language)
		if _, ok := allowed[normalized]; ok {
			out = append(out, language)
		}
	}

	return out
}

func normalizeLanguages(raw []string) []string {
	unique := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))

	for _, language := range raw {
		normalized := strings.TrimSpace(language)
		if normalized == "" {
			continue
		}
		if _, ok := unique[normalized]; ok {
			continue
		}
		unique[normalized] = struct{}{}
		out = append(out, normalized)
	}

	order := map[string]int{
		"typescript": 1,
		"javascript": 2,
		"python":     3,
		"java":       4,
		"cpp":        5,
		"go":         6,
		"rust":       7,
		"zig":        8,
		"csharp":     9,
		"c":          10,
		"ruby":       11,
		"php":        12,
		"sql":        13,
	}

	slices.SortFunc(out, func(left, right string) int {
		leftOrder, leftOK := order[left]
		rightOrder, rightOK := order[right]

		switch {
		case leftOK && rightOK:
			return leftOrder - rightOrder
		case leftOK:
			return -1
		case rightOK:
			return 1
		default:
			return strings.Compare(left, right)
		}
	})

	return out
}

func toFrontendDifficulty(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "easy":
		return "Easy"
	case "hard":
		return "Hard"
	default:
		return "Medium"
	}
}

func fallbackString(primary, fallback string) string {
	trimmed := strings.TrimSpace(primary)
	if trimmed != "" {
		return trimmed
	}
	return fallback
}
