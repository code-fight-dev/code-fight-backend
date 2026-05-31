package tasks

import (
	"testing"

	"code-fight-backend/internal/judge"
)

func TestJudgeCapabilitiesFilterLanguagesAndVersions(t *testing.T) {
	capabilities := newJudgeCapabilities([]judge.SupportedLanguage{
		{Language: " go ", Version: " 1.24 "},
		{Language: "python", Version: "3.12"},
		{Language: "", Version: "ignored"},
		{Language: "rust", Version: ""},
	})

	languages := capabilities.filterLanguages([]string{"python", "rust", "go", "go"})
	if len(languages) != 2 || languages[0] != "python" || languages[1] != "go" {
		t.Fatalf("languages = %#v", languages)
	}

	versions := capabilities.filterVersions(map[string]string{
		"go":     "1.24",
		"python": "3.11",
		"rust":   "1.80",
	}, languages)
	if len(versions) != 1 || versions["go"] != "1.24" {
		t.Fatalf("versions = %#v", versions)
	}
}

func TestFilterStarterCode(t *testing.T) {
	starter := map[string]string{
		"go":     "package main",
		"python": "print()",
	}

	filtered := filterStarterCode(starter, []string{"go"})
	if len(filtered) != 1 || filtered["go"] != "package main" {
		t.Fatalf("filtered starter = %#v", filtered)
	}

	if got := filterStarterCode(nil, []string{"go"}); got != nil {
		t.Fatalf("nil starter = %#v", got)
	}
}
