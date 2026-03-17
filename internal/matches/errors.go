package matches

import "errors"

var (
	ErrMatchNotFound        = errors.New("match not found")
	ErrMatchAlreadyStarted  = errors.New("match already started")
	ErrMatchAlreadyFinished = errors.New("match already finished")
	ErrMatchCancelled       = errors.New("match cancelled")
	ErrInvalidMatchStatus   = errors.New("invalid match status")
	ErrInvalidWinner        = errors.New("invalid winner")
	ErrSamePlayer           = errors.New("players must be different")
	ErrInvalidPagination    = errors.New("invalid pagination")
	ErrRatingAlreadyApplied = errors.New("rating already applied")
	ErrRatingNotFound       = errors.New("rating not found")
)