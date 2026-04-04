package matches

import "errors"

var (
	ErrMatchNotFound      = errors.New("match not found")
	ErrSamePlayer         = errors.New("players must be different")
	ErrInvalidWinner      = errors.New("invalid winner")
	ErrInvalidPagination  = errors.New("invalid pagination")
	ErrMatchAlreadyStarted  = errors.New("match already started")
	ErrMatchAlreadyFinished = errors.New("match already finished")
	ErrMatchCancelled       = errors.New("match cancelled")
	ErrRatingAlreadyApplied = errors.New("rating already applied")
	ErrInvalidMatchStatus   = errors.New("invalid match status")
	ErrRatingNotFound       = errors.New("rating not found")

	ErrInvalidTaskMode      = errors.New("invalid task mode")
	ErrInvalidResultType    = errors.New("invalid result type")
	ErrInvalidWinningReason = errors.New("invalid winning reason")
	ErrInvalidJudgeStatus   = errors.New("invalid judge status")
	ErrTaskNotAssigned      = errors.New("task is not assigned")
	ErrNoTasksAvailable     = errors.New("no tasks available")
)