package setting

import (
    "os"
    "path/filepath"
    "testing"
)
func TestLoadDatabaseSettingsDoesNotRequireHTTPOrJWT(t *testing.T) {
    file:=filepath.Join(t.TempDir(),"app.ini")
    data:="RUN_MODE = development\n[database]\nDB_SOURCE = postgresql://example:5432/bdspro\nMIGRATION_URL = file://db/migration\n"
    if err:=os.WriteFile(file,[]byte(data),0600);err!=nil {t.Fatal(err)}
    cfg,err:=LoadDatabaseSettings(file)
    if err!=nil {t.Fatal(err)}
    if cfg.DBSource!="postgresql://example:5432/bdspro" {t.Fatalf("source: %q",cfg.DBSource)}
}
