package matches

import (
	"context"
	"errors"
	"math/rand/v2"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTaskSelector struct {
	db *pgxpool.Pool
}

func NewDBTaskSelector(db *pgxpool.Pool) *DBTaskSelector {
	return &DBTaskSelector{db: db}
}

func (s *DBTaskSelector) SelectForMatch(ctx context.Context, mode TaskMode) (SelectedTask, error) {
	switch mode {
	case TaskModeHard:
		task, err := s.getRandomActiveByDifficulty(ctx, TaskDifficultyHard)
		if err == nil {
			return task, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return SelectedTask{}, err
		}
		return SelectedTask{}, ErrNoTasksAvailable

	case TaskModeNormal, "":
		difficulties := pickNormalModeDifficultyOrder()

		for _, difficulty := range difficulties {
			task, err := s.getRandomActiveByDifficulty(ctx, difficulty)
			if err == nil {
				return task, nil
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return SelectedTask{}, err
			}
		}

		return SelectedTask{}, ErrNoTasksAvailable

	default:
		return SelectedTask{}, ErrInvalidTaskMode
	}
}

func (s *DBTaskSelector) getRandomActiveByDifficulty(ctx context.Context, difficulty TaskDifficulty) (SelectedTask, error) {
	const query = `
		select id, difficulty
		from tasks
		where is_active = true
		  and difficulty = $1
		order by random()
		limit 1
	`

	var selected SelectedTask
	err := s.db.QueryRow(ctx, query, difficulty).Scan(
		&selected.TaskID,
		&selected.Difficulty,
	)
	if err != nil {
		return SelectedTask{}, err
	}

	return selected, nil
}

func pickNormalModeDifficultyOrder() []TaskDifficulty {
	primary := weightedNormalDifficulty()

	switch primary {
	case TaskDifficultyEasy:
		return []TaskDifficulty{
			TaskDifficultyEasy,
			TaskDifficultyMedium,
			TaskDifficultyHard,
		}
	case TaskDifficultyMedium:
		return []TaskDifficulty{
			TaskDifficultyMedium,
			TaskDifficultyEasy,
			TaskDifficultyHard,
		}
	case TaskDifficultyHard:
		return []TaskDifficulty{
			TaskDifficultyHard,
			TaskDifficultyMedium,
			TaskDifficultyEasy,
		}
	default:
		return []TaskDifficulty{
			TaskDifficultyEasy,
			TaskDifficultyMedium,
			TaskDifficultyHard,
		}
	}
}

func weightedNormalDifficulty() TaskDifficulty {
	n := rand.IntN(100)

	switch {
	case n < 60:
		return TaskDifficultyEasy
	case n < 95:
		return TaskDifficultyMedium
	default:
		return TaskDifficultyHard
	}
}