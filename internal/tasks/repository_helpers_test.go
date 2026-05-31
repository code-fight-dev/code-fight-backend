package tasks

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRepositoryHelperPayloadDefaults(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	payload := challengePayload{}

	if got := payloadID(payload, "two-sum"); got != "two-sum" {
		t.Fatalf("payloadID = %q", got)
	}
	if got := payloadKind(payload); got != "algorithmic" {
		t.Fatalf("payloadKind = %q", got)
	}
	if got := payloadProgress(payload); got != "not-started" {
		t.Fatalf("payloadProgress = %q", got)
	}
	if got := payloadCreatedAt(payload, now); got != "2026-05-30" {
		t.Fatalf("payloadCreatedAt = %q", got)
	}
	if got := toFrontendDifficulty(" HARD "); got != "Hard" {
		t.Fatalf("toFrontendDifficulty = %q", got)
	}
	if got := toFrontendDifficulty("unknown"); got != "Medium" {
		t.Fatalf("toFrontendDifficulty default = %q", got)
	}
	if got := fallbackString("  title  ", "fallback"); got != "title" {
		t.Fatalf("fallbackString primary = %q", got)
	}
	if got := fallbackString(" ", "fallback"); got != "fallback" {
		t.Fatalf("fallbackString fallback = %q", got)
	}
}

func TestDecodeAndNormalizePayload(t *testing.T) {
	payload := decodePayload([]byte(`{"id":"sum","title":"Sum","supportedLanguages":["go"]}`))
	if payload.ID != "sum" || payload.Title != "Sum" {
		t.Fatalf("payload = %#v", payload)
	}

	empty := decodePayload([]byte(`{`))
	if empty.ID != "" {
		t.Fatalf("invalid payload = %#v", empty)
	}

	if string(normalizedPayloadRaw(nil)) != "{}" {
		t.Fatalf("nil payload raw = %s", normalizedPayloadRaw(nil))
	}
	if string(normalizedPayloadRaw([]byte(`{`))) != "{}" {
		t.Fatalf("invalid payload raw = %s", normalizedPayloadRaw([]byte(`{`)))
	}
	raw := normalizedPayloadRaw([]byte(`{"ok":true}`))
	if !json.Valid(raw) || string(raw) != `{"ok":true}` {
		t.Fatalf("raw payload = %s", raw)
	}
}

func TestDecodeSubmissionTestResults(t *testing.T) {
	results := decodeSubmissionTestResults([]byte(`[{"testCaseId":"case-1","status":"accepted"}]`))
	if len(results) != 1 || results[0].TestCaseID != "case-1" {
		t.Fatalf("results = %#v", results)
	}
	if got := decodeSubmissionTestResults([]byte(`null`)); len(got) != 0 {
		t.Fatalf("null results = %#v, want empty", got)
	}
	if got := decodeSubmissionTestResults([]byte(`{`)); len(got) != 0 {
		t.Fatalf("invalid results = %#v, want empty", got)
	}
}

func TestTaskPresentationHelpers(t *testing.T) {
	if got := acceptanceRate(3, 4); got != 75 {
		t.Fatalf("acceptanceRate = %d, want 75", got)
	}
	if got := acceptanceRate(3, 0); got != 0 {
		t.Fatalf("acceptanceRate with zero total = %d", got)
	}

	now := time.Date(2026, 5, 30, 10, 20, 30, 0, time.FixedZone("UTC+2", 2*60*60))
	if got := formatTime(now); got != "2026-05-30T08:20:30Z" {
		t.Fatalf("formatTime = %q", got)
	}
	if formatOptionalTime(nil) != nil {
		t.Fatal("formatOptionalTime(nil) != nil")
	}
	if got := *formatOptionalTime(&now); got != "2026-05-30T08:20:30Z" {
		t.Fatalf("formatOptionalTime = %q", got)
	}

	if got := resolveUserProgress("locked", false, 0); got != "locked" {
		t.Fatalf("locked progress = %q", got)
	}
	if got := resolveUserProgress("locked", false, 2); got != "in-progress" {
		t.Fatalf("in-progress = %q", got)
	}
	if got := resolveUserProgress("not-started", true, 0); got != "solved" {
		t.Fatalf("solved = %q", got)
	}
}

func TestNormalizeLanguagesAndFiltering(t *testing.T) {
	languages := normalizeLanguages([]string{" go ", "python", "", "go", "zig", "aaa"})
	want := []string{"python", "go", "zig", "aaa"}
	if len(languages) != len(want) {
		t.Fatalf("languages = %#v", languages)
	}
	for i := range want {
		if languages[i] != want[i] {
			t.Fatalf("languages[%d] = %q, want %q", i, languages[i], want[i])
		}
	}

	payload := challengePayload{SupportedLanguages: []string{"go", "python", "rust"}}
	filtered := payloadLanguagesAllowedByTask(payload, []string{"python", "go"})
	if len(filtered) != 2 || filtered[0] != "go" || filtered[1] != "python" {
		t.Fatalf("filtered languages = %#v", filtered)
	}
	if got := payloadLanguagesAllowedByTask(challengePayload{}, nil); len(got) != 0 {
		t.Fatalf("languages with no task languages = %#v", got)
	}
}
