package cmd

import (
	"context"
	"strings"
	"testing"
)

func TestMigrateRejectsInvalidStepsBeforeConfiguration(t *testing.T) {
	app := NewMainApp(AppVersion{Version: "test"})
	err := app.Run(context.Background(), []string{"bdspro", "migrate", "--steps=-1"})
	if err == nil || !strings.Contains(err.Error(), "--steps cannot be negative") {
		t.Fatalf("migration accepted invalid steps or accessed dependencies: %v", err)
	}
}
