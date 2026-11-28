package filesystem

// SystemLauncher opens a file in the operating system's default application.
// macOS uses open, Linux uses xdg-open, and Windows uses the shell open verb.
// Tests do not call it.
type SystemLauncher struct{}

// NewSystemLauncher returns the host default-application launcher.
func NewSystemLauncher() SystemLauncher { return SystemLauncher{} }

// Open launches path. The caller passes a cleaned absolute path.
func (SystemLauncher) Open(path string) error { return launch(path) }
