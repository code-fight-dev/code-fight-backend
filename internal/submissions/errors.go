package submissions

import "errors"

var (
	ErrSubmissionNotFound      = errors.New("submission not found")
	ErrTaskNotFound            = errors.New("task not found")
	ErrTaskInactive            = errors.New("task is inactive")
	ErrTaskLanguageNotAllowed  = errors.New("task language is not allowed")
	ErrMatchNotFound           = errors.New("match not found")
	ErrMatchNotStarted         = errors.New("match is not started")
	ErrMatchTaskNotAssigned    = errors.New("match task is not assigned")
	ErrUserNotMatchParticipant = errors.New("user is not a participant of this match")
	ErrInvalidSourceCode       = errors.New("invalid source code")
	ErrJudgeCallbackUnauthorized = errors.New("judge callback unauthorized")
	ErrJudgeSubmitFailed       = errors.New("failed to submit to judge")
	ErrDuplicateJudgeSubmissionID = errors.New("duplicate judge submission id")
)