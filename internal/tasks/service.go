package tasks

import (
	"context"
	"strings"

	"code-fight-backend/internal/judge"
)

type JudgeLanguageProvider interface {
	SupportedLanguages(ctx context.Context) ([]judge.SupportedLanguage, error)
}

type Service struct {
	repo  *Repository
	judge JudgeLanguageProvider
}

func NewService(repo *Repository, judgeProvider JudgeLanguageProvider) *Service {
	return &Service{
		repo:  repo,
		judge: judgeProvider,
	}
}

func (s *Service) ListChallenges(ctx context.Context) ([]ChallengeListItem, error) {
	items, err := s.repo.ListChallenges(ctx)
	if err != nil {
		return nil, err
	}

	capabilities, err := s.judgeCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	for i := range items {
		items[i].SupportedLanguages = capabilities.filterLanguages(items[i].SupportedLanguages)
	}

	return items, nil
}

func (s *Service) GetChallengeBySlug(ctx context.Context, slug string) (*Challenge, error) {
	challenge, err := s.repo.GetChallengeBySlug(ctx, slug)
	if err != nil || challenge == nil {
		return challenge, err
	}

	capabilities, err := s.judgeCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	challenge.SupportedLanguages = capabilities.filterLanguages(challenge.SupportedLanguages)
	challenge.LanguageVersions = capabilities.filterVersions(challenge.LanguageVersions, challenge.SupportedLanguages)
	challenge.StarterCodeByLanguage = filterStarterCode(challenge.StarterCodeByLanguage, challenge.SupportedLanguages)

	return challenge, nil
}

func (s *Service) judgeCapabilities(ctx context.Context) (judgeCapabilities, error) {
	supported, err := s.judge.SupportedLanguages(ctx)
	if err != nil {
		return judgeCapabilities{}, err
	}

	return newJudgeCapabilities(supported), nil
}

type judgeCapabilities struct {
	versions map[string]string
}

func newJudgeCapabilities(supported []judge.SupportedLanguage) judgeCapabilities {
	versions := make(map[string]string, len(supported))
	for _, item := range supported {
		language := strings.TrimSpace(item.Language)
		version := strings.TrimSpace(item.Version)
		if language == "" || version == "" {
			continue
		}

		versions[language] = version
	}

	return judgeCapabilities{versions: versions}
}

func (c judgeCapabilities) filterLanguages(languages []string) []string {
	out := make([]string, 0, len(languages))
	for _, language := range normalizeLanguages(languages) {
		if _, ok := c.versions[language]; ok {
			out = append(out, language)
		}
	}

	return out
}

func (c judgeCapabilities) filterVersions(versions map[string]string, languages []string) map[string]string {
	allowed := make(map[string]struct{}, len(languages))
	for _, language := range languages {
		allowed[language] = struct{}{}
	}

	out := make(map[string]string, len(versions))
	for language, version := range versions {
		if _, ok := allowed[language]; !ok {
			continue
		}

		judgeVersion, ok := c.versions[language]
		if !ok || strings.TrimSpace(version) != judgeVersion {
			continue
		}

		out[language] = version
	}

	return out
}

func filterStarterCode(starterCode map[string]string, languages []string) map[string]string {
	if len(starterCode) == 0 {
		return starterCode
	}

	allowed := make(map[string]struct{}, len(languages))
	for _, language := range languages {
		allowed[language] = struct{}{}
	}

	out := make(map[string]string, len(allowed))
	for language, code := range starterCode {
		if _, ok := allowed[language]; ok {
			out[language] = code
		}
	}

	return out
}

