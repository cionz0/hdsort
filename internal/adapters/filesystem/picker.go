package filesystem

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// SystemPicker is the native folder dialog for this operating system.
// Tests do not call it. They pass a fake picker to the server.
type SystemPicker struct{}

// NewSystemPicker returns the host folder dialog.
func NewSystemPicker() SystemPicker { return SystemPicker{} }

// Pick opens the system folder dialog and returns the chosen directory.
func (SystemPicker) Pick() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return output("osascript", "-e", `POSIX path of (choose folder with prompt "Choose a folder to scan")`)
	case "windows":
		return output("powershell", "-NoProfile", "-Command", `
Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.FolderBrowserDialog
if ($dialog.ShowDialog() -ne [System.Windows.Forms.DialogResult]::OK) { exit 1 }
$dialog.SelectedPath
`)
	default:
		return output("zenity", "--file-selection", "--directory", "--title=Choose a folder to scan")
	}
}

func output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("folder dialog: %s", msg)
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", fmt.Errorf("folder dialog: empty path")
	}
	return path, nil
}
