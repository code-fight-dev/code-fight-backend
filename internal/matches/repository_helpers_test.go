package matches

import (
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestNewRepositoryAndTaskSelector(t *testing.T) {
	if repo := NewRepository(nil); repo == nil {
		t.Fatal("NewRepository returned nil")
	}
	if selector := NewDBTaskSelector(nil); selector == nil {
		t.Fatal("NewDBTaskSelector returned nil")
	}
}

func TestValidateFinishInput(t *testing.T) {
	player1 := uuid.New()
	player2 := uuid.New()
	match := Match{Player1ID: player1, Player2ID: player2}

	tests := []struct {
		name    string
		input   FinishMatchInput
		wantErr error
	}{
		{name: "draw", input: FinishMatchInput{ResultType: MatchResultTypeDraw}},
		{name: "player1 win", input: FinishMatchInput{ResultType: MatchResultTypePlayer1Win, WinnerID: &player1}},
		{name: "player2 win", input: FinishMatchInput{ResultType: MatchResultTypePlayer2Win, WinnerID: &player2}},
		{name: "draw with winner", input: FinishMatchInput{ResultType: MatchResultTypeDraw, WinnerID: &player1}, wantErr: ErrInvalidWinner},
		{name: "wrong winner", input: FinishMatchInput{ResultType: MatchResultTypePlayer1Win, WinnerID: &player2}, wantErr: ErrInvalidWinner},
		{name: "cancelled", input: FinishMatchInput{ResultType: MatchResultTypeCancelled}, wantErr: ErrInvalidMatchStatus},
		{name: "unknown", input: FinishMatchInput{ResultType: "unknown"}, wantErr: ErrInvalidMatchStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFinishInput(match, tt.input)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("validateFinishInput returned error: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateFinishInput error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestEloHelpers(t *testing.T) {
	player1 := uuid.New()
	player2 := uuid.New()
	match := Match{Player1ID: player1, Player2ID: player2}

	result1, result2, err := eloResultsForMatch(match, FinishMatchInput{ResultType: MatchResultTypeDraw})
	if err != nil || result1 != 0.5 || result2 != 0.5 {
		t.Fatalf("draw results = %f/%f, %v", result1, result2, err)
	}

	result1, result2, err = eloResultsForMatch(match, FinishMatchInput{ResultType: MatchResultTypePlayer1Win, WinnerID: &player1})
	if err != nil || result1 != 1 || result2 != 0 {
		t.Fatalf("player1 results = %f/%f, %v", result1, result2, err)
	}

	if _, _, err := eloResultsForMatch(match, FinishMatchInput{ResultType: "bad"}); !errors.Is(err, ErrInvalidMatchStatus) {
		t.Fatalf("eloResultsForMatch error = %v", err)
	}

	if score := expectedScore(1200, 1200); math.Abs(score-0.5) > 0.0001 {
		t.Fatalf("expectedScore = %f, want 0.5", score)
	}
	if got := kFactor(0); got != 40 {
		t.Fatalf("kFactor new player = %d", got)
	}
	if got := kFactor(30); got != 20 {
		t.Fatalf("kFactor established player = %d", got)
	}
	if got := calculateNewRating(1200, 1200, 0, 1); got != 1220 {
		t.Fatalf("calculateNewRating = %d, want 1220", got)
	}
}

func TestForeignKeyViolationAndPointer(t *testing.T) {
	status := ptrJudgeStatus(JudgeStatusFinished)
	if status == nil || *status != JudgeStatusFinished {
		t.Fatalf("ptrJudgeStatus = %v", status)
	}

	if !isForeignKeyViolation(&pgconn.PgError{Code: "23503"}) {
		t.Fatal("isForeignKeyViolation = false for 23503")
	}
	if isForeignKeyViolation(&pgconn.PgError{Code: "23505"}) {
		t.Fatal("isForeignKeyViolation = true for non-FK code")
	}
	if isForeignKeyViolation(errors.New("plain")) {
		t.Fatal("isForeignKeyViolation = true for plain error")
	}
}

func TestPickNormalModeDifficultyOrder(t *testing.T) {
	for i := 0; i < 100; i++ {
		order := pickNormalModeDifficultyOrder()
		if len(order) != 3 {
			t.Fatalf("order length = %d", len(order))
		}
		seen := map[TaskDifficulty]bool{}
		for _, difficulty := range order {
			seen[difficulty] = true
		}
		if !seen[TaskDifficultyEasy] || !seen[TaskDifficultyMedium] || !seen[TaskDifficultyHard] {
			t.Fatalf("order missing difficulty: %#v", order)
		}
	}
}
