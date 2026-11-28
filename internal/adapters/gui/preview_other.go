//go:build !darwin && !windows

package gui

func systemPreview(string) ([]byte, bool, error) {
	return nil, false, nil
}
