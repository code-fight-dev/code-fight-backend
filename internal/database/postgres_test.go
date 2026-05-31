package database

import (
	"context"
	"strings"
	"testing"
)

func TestConnectRejectsInvalidURL(t *testing.T) {
	_, err := Connect(context.Background(), "://not-a-dsn")
	if err == nil || !strings.Contains(err.Error(), "parse database url") {
		t.Fatalf("Connect error = %v", err)
	}
}
