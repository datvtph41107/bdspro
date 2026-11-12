package architecture

import (
	"os"
	"path/filepath"
	"testing"
)

// The infrastructure reset is a deletion contract, not a compatibility shim.
func TestNoLegacyDomainSource(t *testing.T) {
	root := filepath.Join("..", "..")
	removed := []string{
		"models/account", "models/audit", "models/auth", "models/oauth",
		"models/organization", "services/auth", "services/oauth",
		"services/org", "services/user", "services/security",
		"services/worker", "services/jobs", "services/mail",
		"services/notification", "services/convert", "routers/common",
		"routers/web", "routers/api/v1/user", "routers/api/v1/org",
		"routers/api/v1/oauth", "modules/structs",
	}
	for _, name := range removed {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			t.Errorf("legacy domain source remains: %s", name)
		} else if !os.IsNotExist(err) {
			t.Errorf("checking %s: %v", name, err)
		}
	}
	for _, name := range []string{
		"db/migration", "db/query", "models/db/sqlc",
		"modules/web", "modules/graceful", "services/context",
		"routers/init.go", "routers/api/v1/api.go",
	} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("required infrastructure missing: %s: %v", name, err)
		}
	}
}
