package app

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
)

func TestUpdateRejectsFilter(t *testing.T) {
	ac := &Context{Config: &config.Effective{}}
	_, err := Update(context.Background(), ac, UpdateOptions{
		Install: InstallOptions{Filter: []string{"alpha"}},
	})
	if err == nil || apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("expected usage error, got %v", err)
	}
}
