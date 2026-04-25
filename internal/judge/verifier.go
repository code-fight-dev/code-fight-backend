package judge

import (
	"crypto/subtle"
	"errors"
	"strings"
)

var ErrUnauthorizedCallback = errors.New("unauthorized judge callback")

type Verifier struct {
	callbackToken string
}

func NewVerifier(callbackToken string) *Verifier {
	return &Verifier{
		callbackToken: callbackToken,
	}
}

func (v *Verifier) VerifyToken(receivedToken string) error {
	expected := strings.TrimSpace(v.callbackToken)
	received := strings.TrimSpace(receivedToken)

	if expected == "" || received == "" {
		return ErrUnauthorizedCallback
	}

	if subtle.ConstantTimeCompare([]byte(expected), []byte(received)) != 1 {
		return ErrUnauthorizedCallback
	}

	return nil
}