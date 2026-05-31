package tasks

import (
	"context"
	"testing"

	"code-fight-backend/internal/judge"
)

func TestServiceListChallengesFiltersJudgeSupportedLanguages(t *testing.T) {
	repo := &fakeTaskRepo{
		items: []ChallengeListItem{
			{Slug: "sum", SupportedLanguages: []string{"go", "python", "rust"}},
		},
	}
	service := NewService(repo, fakeJudgeLanguages{{Language: "go", Version: "1.24"}})

	items, err := service.ListChallenges(context.Background())
	if err != nil {
		t.Fatalf("ListChallenges returned error: %v", err)
	}
	if len(items) != 1 || len(items[0].SupportedLanguages) != 1 || items[0].SupportedLanguages[0] != "go" {
		t.Fatalf("items = %#v", items)
	}

	userItems, err := service.ListChallengesForUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListChallengesForUser returned error: %v", err)
	}
	if len(userItems) != 1 || repo.userID != "user-1" {
		t.Fatalf("user items = %#v, userID = %q", userItems, repo.userID)
	}
}

func TestServiceChallengeDetailFiltersLanguagesVersionsAndStarterCode(t *testing.T) {
	challenge := &Challenge{
		Slug:                  "sum",
		SupportedLanguages:    []string{"go", "python", "rust"},
		LanguageVersions:      map[string]string{"go": "1.24", "python": "3.11", "rust": "1.80"},
		StarterCodeByLanguage: map[string]string{"go": "package main", "python": "print()", "rust": "fn main() {}"},
	}
	repo := &fakeTaskRepo{challenge: challenge}
	service := NewService(repo, fakeJudgeLanguages{
		{Language: "go", Version: "1.24"},
		{Language: "python", Version: "3.12"},
	})

	got, err := service.GetChallengeBySlug(context.Background(), "sum")
	if err != nil {
		t.Fatalf("GetChallengeBySlug returned error: %v", err)
	}
	if len(got.SupportedLanguages) != 2 || got.SupportedLanguages[0] != "python" || got.SupportedLanguages[1] != "go" {
		t.Fatalf("supported languages = %#v", got.SupportedLanguages)
	}
	if len(got.LanguageVersions) != 1 || got.LanguageVersions["go"] != "1.24" {
		t.Fatalf("versions = %#v", got.LanguageVersions)
	}
	if len(got.StarterCodeByLanguage) != 2 || got.StarterCodeByLanguage["go"] == "" || got.StarterCodeByLanguage["python"] == "" {
		t.Fatalf("starter code = %#v", got.StarterCodeByLanguage)
	}

	repo.challenge = nil
	got, err = service.GetChallengeByID(context.Background(), "task-1")
	if err != nil || got != nil {
		t.Fatalf("nil challenge = %#v, %v", got, err)
	}

	repo.challenge = challenge
	got, err = service.GetChallengeByID(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetChallengeByID returned error: %v", err)
	}
	if got == nil || len(got.SupportedLanguages) != 2 || got.LanguageVersions["go"] != "1.24" {
		t.Fatalf("id challenge = %#v", got)
	}
}

func TestServiceSimpleRepositoryDelegates(t *testing.T) {
	repo := &fakeTaskRepo{
		stats:       UserTaskStats{SolvedTasks: 2},
		submissions: []TaskSubmission{{ID: "submission-1"}},
	}
	service := NewService(repo, fakeJudgeLanguages{})

	stats, err := service.GetUserTaskStats(context.Background(), "user-1")
	if err != nil || stats.SolvedTasks != 2 {
		t.Fatalf("stats = %#v, %v", stats, err)
	}

	submissions, err := service.ListUserTaskSubmissions(context.Background(), "task-1", "user-1")
	if err != nil || len(submissions) != 1 {
		t.Fatalf("submissions = %#v, %v", submissions, err)
	}
}

type fakeTaskRepo struct {
	items       []ChallengeListItem
	challenge   *Challenge
	stats       UserTaskStats
	submissions []TaskSubmission
	userID      string
}

func (r *fakeTaskRepo) ListChallenges(context.Context) ([]ChallengeListItem, error) {
	return append([]ChallengeListItem(nil), r.items...), nil
}

func (r *fakeTaskRepo) ListChallengesForUser(_ context.Context, userID string) ([]ChallengeListItem, error) {
	r.userID = userID
	return append([]ChallengeListItem(nil), r.items...), nil
}

func (r *fakeTaskRepo) GetUserTaskStats(context.Context, string) (UserTaskStats, error) {
	return r.stats, nil
}

func (r *fakeTaskRepo) ListUserTaskSubmissions(context.Context, string, string) ([]TaskSubmission, error) {
	return r.submissions, nil
}

func (r *fakeTaskRepo) GetChallengeBySlug(context.Context, string) (*Challenge, error) {
	return cloneChallenge(r.challenge), nil
}

func (r *fakeTaskRepo) GetChallengeByID(context.Context, string) (*Challenge, error) {
	return cloneChallenge(r.challenge), nil
}

type fakeJudgeLanguages []judge.SupportedLanguage

func (l fakeJudgeLanguages) SupportedLanguages(context.Context) ([]judge.SupportedLanguage, error) {
	return []judge.SupportedLanguage(l), nil
}

func cloneChallenge(challenge *Challenge) *Challenge {
	if challenge == nil {
		return nil
	}
	out := *challenge
	out.SupportedLanguages = append([]string(nil), challenge.SupportedLanguages...)
	out.LanguageVersions = make(map[string]string, len(challenge.LanguageVersions))
	for language, version := range challenge.LanguageVersions {
		out.LanguageVersions[language] = version
	}
	out.StarterCodeByLanguage = make(map[string]string, len(challenge.StarterCodeByLanguage))
	for language, code := range challenge.StarterCodeByLanguage {
		out.StarterCodeByLanguage[language] = code
	}
	return &out
}
