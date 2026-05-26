package auth

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email already in use")
	ErrUsernameTaken      = errors.New("username already in use")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidInput       = errors.New("invalid input")
	ErrOAuthUnavailable   = errors.New("oauth provider is not configured")
	ErrOAuthStateInvalid  = errors.New("oauth state is invalid")
	ErrOAuthEmailRequired = errors.New("oauth provider did not return a verified email")
	ErrOAuthAccountLinked = errors.New("oauth account is already linked")
	ErrResetCodeInvalid   = errors.New("password reset code is invalid")
	ErrResetCodeExpired   = errors.New("password reset code is expired")
	ErrCaptchaInvalid     = errors.New("captcha verification failed")
	ErrCaptchaUnavailable = errors.New("captcha verification is unavailable")
)
