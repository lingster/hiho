package viewport

// Model holds viewport content.
type Model struct {
	Width   int
	Height  int
	content string
}

// New constructs a Model.
func New(width, height int) Model {
	return Model{Width: width, Height: height}
}

// SetContent sets the visible content.
func (m *Model) SetContent(content string) {
	m.content = content
}

// View returns the content.
func (m Model) View() string {
	return m.content
}
