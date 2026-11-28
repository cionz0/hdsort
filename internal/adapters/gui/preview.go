package gui

type osPreview struct{}

func (osPreview) Preview(path string) ([]byte, bool, error) {
	return systemPreview(path)
}
