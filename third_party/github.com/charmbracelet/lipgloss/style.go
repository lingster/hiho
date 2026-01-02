package lipgloss

import "strings"

// Style models simple styling options.
type Style struct {
	bold      bool
	marginTop int
}

// NewStyle constructs a Style.
func NewStyle() Style { return Style{} }

// Bold toggles bold rendering.
func (s Style) Bold(enabled bool) Style {
	s.bold = enabled
	return s
}

// MarginTop sets the top margin lines.
func (s Style) MarginTop(lines int) Style {
	s.marginTop = lines
	return s
}

// Render applies minimal styling.
func (s Style) Render(str string) string {
	var builder strings.Builder
	for i := 0; i < s.marginTop; i++ {
		builder.WriteString("\n")
	}
	if s.bold {
		builder.WriteString(str)
	} else {
		builder.WriteString(str)
	}
	return builder.String()
}

// JoinVertical concatenates segments vertically.
func JoinVertical(_ Position, parts ...string) string {
	return strings.Join(parts, "\n")
}

// Position is a placeholder for compatibility.
type Position int

// Left is the default join position.
const Left Position = 0
