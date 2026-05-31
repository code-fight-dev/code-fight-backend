package judge

import (
	"errors"
	"testing"
)

func TestVerifierVerifyToken(t *testing.T) {
	tests := []struct {
		name    string
		stored  string
		got     string
		wantErr bool
	}{
		{
			name:   "valid token",
			stored: "callback-token",
			got:    "callback-token",
		},
		{
			name:   "trims whitespace",
			stored: " callback-token ",
			got:    "\tcallback-token\n",
		},
		{
			name:    "empty stored token",
			stored:  "",
			got:     "callback-token",
			wantErr: true,
		},
		{
			name:    "empty received token",
			stored:  "callback-token",
			got:     " ",
			wantErr: true,
		},
		{
			name:    "wrong token",
			stored:  "callback-token",
			got:     "wrong-token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewVerifier(tt.stored).VerifyToken(tt.got)
			if tt.wantErr {
				if !errors.Is(err, ErrUnauthorizedCallback) {
					t.Fatalf("VerifyToken error = %v, want ErrUnauthorizedCallback", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("VerifyToken returned error: %v", err)
			}
		})
	}
}
