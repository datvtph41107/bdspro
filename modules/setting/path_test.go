package setting

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestResolvePathsDefaultFromSourceDirectory(t *testing.T) {
    t.Setenv("BDS_WORK_DIR","")
    cwd, err := os.Getwd()
    if err != nil { t.Fatal(err) }
    got, err := ResolvePaths(PathOptions{})
    if err != nil {t.Fatal(err)}
    want:=filepath.Join(cwd,"custom","conf","app.ini")
    if got.ConfigFile != want {t.Fatalf("config=%q want %q",got.ConfigFile,want)}
}
func TestResolvePathsExplicitConfigWinsWorkDir(t *testing.T) {
    t.Setenv("BDS_WORK_DIR",t.TempDir())
    root:=t.TempDir()
    got,err:=ResolvePaths(PathOptions{Config:"custom/conf/test.ini",WorkPath:root})
    if err!=nil {t.Fatal(err)}
    if got.WorkPath!=root {t.Fatalf("work path %q",got.WorkPath)}
    if !strings.HasSuffix(got.ConfigFile,filepath.Join("custom","conf","test.ini")) {t.Fatalf("explicit config: %q",got.ConfigFile)}
}
func TestResolvePathsRejectsRelativeWorkPath(t *testing.T) {
    t.Setenv("BDS_WORK_DIR","")
    if _,err:=ResolvePaths(PathOptions{WorkPath:"relative"});err==nil {t.Fatal("relative work path accepted")}
}
