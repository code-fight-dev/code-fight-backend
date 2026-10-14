package tasks

import (
	"strings"
	"testing"

	"code-fight-backend/internal/database"
	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/testsupport"
)

func TestMultilanguageCatalogMatchesJudgeCapabilities(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	provider := fakeJudgeLanguages{{Language: "go", Version: "1.24"}, {Language: "python", Version: "3.12"}, {Language: "cpp", Version: "20"}, {Language: "javascript", Version: "node24"}, {Language: "typescript", Version: "5.9.3"}}
	service := NewService(NewRepository(db), provider)
	for _, slug := range []string{"signal-stabilizer", "mirror-log", "cache-key-collisions", "packet-window-median", "courier-routes", "build-pipeline-order", "adaptive-pricing", "binary-signal-boundary", "archive-compression", "colony-perimeter", "latency-budget", "quantum-safe-lock", "priority-inbox", "trade-ledger-balance", "unioned-accounts", "sensor-drift"} {
		t.Run(slug, func(t *testing.T) {
			challenge, err := service.GetChallengeBySlug(ctx, slug)
			if err != nil {
				t.Fatal(err)
			}
			if challenge == nil {
				t.Fatal("missing built-in challenge")
			}
			for _, runtime := range []judge.SupportedLanguage{{Language: "cpp", Version: "20"}, {Language: "javascript", Version: "node24"}, {Language: "typescript", Version: "5.9.3"}} {
				if challenge.LanguageVersions[runtime.Language] != runtime.Version || !strings.Contains(strings.ToLower(challenge.StarterCodeByLanguage[runtime.Language]), "solve(") {
					t.Fatalf("runtime=%+v challenge=%+v", runtime, challenge)
				}
				var enabled bool
				if err := db.QueryRow(ctx, `select is_enabled from task_languages where task_id=$1 and language=$2 and language_version=$3`, challenge.TaskID, runtime.Language, runtime.Version).Scan(&enabled); err != nil || !enabled {
					t.Fatalf("runtime disabled: %s %v", runtime.Language, err)
				}
			}
			var obsolete int
			if err := db.QueryRow(ctx, `select count(*) from task_languages where task_id=$1 and is_enabled and (language,language_version) in (('cpp','gnu++23'),('javascript','node22'),('typescript','5.8'))`, challenge.TaskID).Scan(&obsolete); err != nil || obsolete != 0 {
				t.Fatalf("obsolete runtimes=%d err=%v", obsolete, err)
			}
		})
	}
}
