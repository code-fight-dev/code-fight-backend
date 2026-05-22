package matches

import "errors"

var (
	ErrMatchNotFound        = errors.New("match not found")
	ErrSamePlayer           = errors.New("players must be different")
	ErrInvalidWinner        = errors.New("invalid winner")
	ErrInvalidPagination    = errors.New("invalid pagination")
	ErrMatchAlreadyStarted  = errors.New("match already started")
	ErrMatchAlreadyFinished = errors.New("match already finished")
	ErrMatchCancelled       = errors.New("match cancelled")
	ErrRatingAlreadyApplied = errors.New("rating already applied")
	ErrInvalidMatchStatus   = errors.New("invalid match status")
	ErrRatingNotFound       = errors.New("rating not found")

	ErrInvalidTaskMode          = errors.New("invalid task mode")
	ErrInvalidResultType        = errors.New("invalid result type")
	ErrInvalidWinningReason     = errors.New("invalid winning reason")
	ErrInvalidJudgeStatus       = errors.New("invalid judge status")
	ErrTaskNotAssigned          = errors.New("task is not assigned")
	ErrNoTasksAvailable         = errors.New("no tasks available")
	ErrSubmissionUserNotInMatch = errors.New("submission user is not a participant of this match")
	ErrMatchReplayUnavailable   = errors.New("replay is available only for finished or cancelled matches")
	ErrMatchReplayForbidden     = errors.New("replay is available only for match participants")

	ErrMatchReplayCaptureUnavailable = errors.New("replay capture is available only while match is running or during a short post-match grace period")
	ErrReplayEventsEmpty             = errors.New("replay events payload is empty")
	ErrReplayEventsBatchTooLarge     = errors.New("replay events batch is too large")
	ErrReplayInvalidEvent            = errors.New("invalid replay event")
	ErrReplayEventPayloadTooLarge    = errors.New("replay event payload is too large")
	ErrReplayInvalidSnapshot         = errors.New("invalid replay snapshot")
	ErrReplaySnapshotTooLarge        = errors.New("replay snapshot source is too large")
	ErrReplaySeqConflict             = errors.New("replay sequence conflict")
	ErrReplaySnapshotAheadOfEvents   = errors.New("replay snapshot sequence is ahead of ingested events")
)
