package main

import (
	"strings"
	"testing"
)

func TestForbidden(t *testing.T) {
	tests := []struct {
		name, source, dependency, want string
	}{
		{"db cannot import setting", module + "models/db", module + "modules/setting", "ARCH-RESOURCE-001"},
		{"redis cannot import graceful", module + "modules/conn", module + "modules/graceful", "ARCH-RESOURCE-001"},
		{"service cannot import router", module + "services/org", module + "routers/api/v1", "ARCH-SERVICE-001"},
		{"service can import model", module + "services/org", module + "models/db", ""},
		{"composition can import setting", module + "routers/common", module + "modules/setting", ""},
		{"external dependency accepted", module + "services/org", "github.com/jackc/pgx/v5", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := forbidden(tt.source, tt.dependency); got != tt.want {
				t.Errorf("forbidden() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckReportsViolations(t *testing.T) {
	input := `{"ImportPath":"github.com/datvtph41107/bdspro/models/db","Imports":["github.com/datvtph41107/bdspro/modules/setting"]}
{"ImportPath":"github.com/datvtph41107/bdspro/services/org","Imports":["github.com/datvtph41107/bdspro/models/db"]}`
	issues, err := check(strings.NewReader(input))
	if err != nil || len(issues) != 1 || issues[0].Rule != "ARCH-RESOURCE-001" {
		t.Fatalf("issues=%v, err=%v", issues, err)
	}
}

func TestCheckRejectsInvalidJSON(t *testing.T) {
	if _, err := check(strings.NewReader("{")); err == nil {
		t.Fatal("expected decoder error")
	}
}
