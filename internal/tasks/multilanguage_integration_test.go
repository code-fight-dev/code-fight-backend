package tasks

import (
	"strings"
	"testing"

	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/testsupport"
)

func TestJavaCatalogRequiresJudgeCapability(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	repo := NewRepository(db)
	for _, version := range []string{"17", "21"} {
		service := NewService(repo, fakeJudgeLanguages{{Language: "java", Version: version}})
		challenge, err := service.GetChallengeBySlug(ctx, "mirror-log")
		if err != nil || challenge == nil {
			t.Fatalf("Mirror Log catalog: %+v, %v", challenge, err)
		}
		if got := challenge.LanguageVersions["java"]; (version == "21" && got != "21") || (version != "21" && got != "") {
			t.Fatalf("Java %s capability exposed catalog version %q", version, got)
		}
		if version == "21" && !strings.Contains(challenge.StarterCodeByLanguage["java"], "public class Main") {
			t.Fatal("Java starter does not define Main")
		}
	}
}

func TestMultilanguageCatalogMatchesJudgeCapabilities(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)

	provider := fakeJudgeLanguages{
		{Language: "c", Version: "23"},
		{Language: "go", Version: "1.24"},
		{Language: "python", Version: "3.12"},
		{Language: "cpp", Version: "20"},
		{Language: "javascript", Version: "node24"},
		{Language: "typescript", Version: "5.9.3"},
	}

	service := NewService(NewRepository(db), provider)

	slugs := []string{
		"signal-stabilizer",
		"mirror-log",
		"cache-key-collisions",
		"packet-window-median",
		"courier-routes",
		"build-pipeline-order",
		"adaptive-pricing",
		"binary-signal-boundary",
		"archive-compression",
		"colony-perimeter",
		"latency-budget",
		"quantum-safe-lock",
		"priority-inbox",
		"trade-ledger-balance",
		"unioned-accounts",
		"sensor-drift",
	}

	runtimes := []judge.SupportedLanguage{
		{Language: "c", Version: "23"},
		{Language: "cpp", Version: "20"},
		{Language: "javascript", Version: "node24"},
		{Language: "typescript", Version: "5.9.3"},
	}

	for _, slug := range slugs {
		t.Run(slug, func(t *testing.T) {
			challenge, err := service.GetChallengeBySlug(ctx, slug)
			if err != nil {
				t.Fatal(err)
			}

			if challenge == nil {
				t.Fatal("missing built-in challenge")
			}

			for _, runtime := range runtimes {
				t.Run(runtime.Language, func(t *testing.T) {
					version := challenge.LanguageVersions[runtime.Language]
					if version != runtime.Version {
						t.Fatalf(
							"language %s: version = %q, want %q",
							runtime.Language,
							version,
							runtime.Version,
						)
					}

					starter := challenge.StarterCodeByLanguage[runtime.Language]
					if !strings.Contains(strings.ToLower(starter), "solve(") {
						t.Fatalf(
							"language %s: starter code has no solve function",
							runtime.Language,
						)
					}

					var enabled bool

					err := db.QueryRow(ctx, `
						SELECT is_enabled
						FROM task_languages
						WHERE task_id = $1
						  AND language = $2
						  AND language_version = $3
					`,
						challenge.TaskID,
						runtime.Language,
						runtime.Version,
					).Scan(&enabled)

					if err != nil {
						t.Fatalf(
							"query language %s: %v",
							runtime.Language,
							err,
						)
					}

					if !enabled {
						t.Fatalf(
							"language %s version %s is disabled",
							runtime.Language,
							runtime.Version,
						)
					}
				})
			}

			var obsolete int

			err = db.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM task_languages
				WHERE task_id = $1
				  AND is_enabled = true
				  AND (language, language_version) IN (
					  ('cpp', 'gnu++23'),
					  ('javascript', 'node22'),
					  ('typescript', '5.8'),
					  ('c', 'c23')
				  )
			`, challenge.TaskID).Scan(&obsolete)

			if err != nil {
				t.Fatalf("query obsolete runtimes: %v", err)
			}

			if obsolete != 0 {
				t.Fatalf(
					"found %d enabled obsolete runtimes",
					obsolete,
				)
			}
		})
	}
}
