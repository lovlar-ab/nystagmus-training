package login

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnableDisable(t *testing.T) {
	a := Agent{Dir: filepath.Join(t.TempDir(), "LaunchAgents")}
	if a.Enabled() {
		t.Fatal("enabled before Enable")
	}
	if err := a.Disable(); err != nil {
		t.Fatalf("Disable without file: %v", err)
	}
	if err := a.Enable("/Applications/A & B.app/Contents/MacOS/x"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(a.path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "<string>/Applications/A &amp; B.app/Contents/MacOS/x</string>") {
		t.Fatalf("path missing or unescaped:\n%s", body)
	}
	if err := a.Disable(); err != nil || a.Enabled() {
		t.Fatalf("Disable: err=%v enabled=%v", err, a.Enabled())
	}
}

func TestSyncOnlyWhenEnabled(t *testing.T) {
	a := Agent{Dir: t.TempDir()}
	if err := a.Sync("/new/x"); err != nil || a.Enabled() {
		t.Fatalf("Sync enabled a disabled agent: err=%v", err)
	}
	if err := a.Enable("/old/x"); err != nil {
		t.Fatal(err)
	}
	if err := a.Sync("/new/x"); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(a.path())
	if !strings.Contains(string(body), "/new/x") {
		t.Fatalf("Sync kept the old path:\n%s", body)
	}
}

func TestEnableNamesFailure(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	err := Agent{Dir: filepath.Join(blocker, "LaunchAgents")}.Enable("/x")
	if err == nil || !strings.Contains(err.Error(), "create ") {
		t.Fatalf("err = %v, want a create error", err)
	}
}
