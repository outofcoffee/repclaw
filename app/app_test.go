package app

import (
	"context"
	"strings"
	"testing"
)

func TestRun_RequiresClient(t *testing.T) {
	err := Run(context.Background(), RunOptions{})
	if err == nil {
		t.Fatal("expected error when Client is nil")
	}
	if !strings.Contains(err.Error(), "Client is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
