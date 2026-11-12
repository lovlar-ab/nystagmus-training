// Package login opens the app at login.
package login

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const Label = "local.nystagmus-training"

// Agent is one launchd agent file.
type Agent struct{ Dir string }

// Default uses ~/Library/LaunchAgents.
func Default() (Agent, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Agent{}, fmt.Errorf("find home folder: %w", err)
	}
	return Agent{Dir: filepath.Join(home, "Library", "LaunchAgents")}, nil
}

func (a Agent) path() string { return filepath.Join(a.Dir, Label+".plist") }

func (a Agent) Enabled() bool {
	_, err := os.Stat(a.path())
	return err == nil
}

// Enable writes the agent for exe.
func (a Agent) Enable(exe string) error {
	if err := os.MkdirAll(a.Dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", a.Dir, err)
	}
	if err := os.WriteFile(a.path(), []byte(plist(exe)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", a.path(), err)
	}
	return nil
}

func (a Agent) Disable() error {
	if err := os.Remove(a.path()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", a.path(), err)
	}
	return nil
}

// Sync repoints an enabled agent at exe.
func (a Agent) Sync(exe string) error {
	if !a.Enabled() {
		return nil
	}
	return a.Enable(exe)
}

func plist(exe string) string {
	var escaped strings.Builder
	// EscapeText only fails on writer errors.
	_ = xml.EscapeText(&escaped, []byte(exe))
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + Label + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + escaped.String() + `</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>ProcessType</key>
	<string>Interactive</string>
</dict>
</plist>
`
}
