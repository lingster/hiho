package lipgloss

import (
	"fmt"
	"strings"
)

// Color represents an ANSI color code.
type Color string

// Style models simple styling options.
type Style struct {
	bold       bool
	marginTop  int
	fg         Color
	bg         Color
	paddingH   int
	paddingV   int
	border     bool
	width      int
	height     int
	reverse    bool
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

// Foreground sets the foreground color.
func (s Style) Foreground(c Color) Style {
	s.fg = c
	return s
}

// Background sets the background color.
func (s Style) Background(c Color) Style {
	s.bg = c
	return s
}

// Padding sets vertical and horizontal padding.
func (s Style) Padding(vertical, horizontal int) Style {
	s.paddingV = vertical
	s.paddingH = horizontal
	return s
}

// Border enables or disables a box border around the content.
func (s Style) Border(enabled bool) Style {
	s.border = enabled
	return s
}

// Width sets a fixed width for the styled content.
func (s Style) Width(w int) Style {
	s.width = w
	return s
}

// Height sets a fixed height for the styled content.
func (s Style) Height(h int) Style {
	s.height = h
	return s
}

// Reverse enables reverse video (swap fg/bg) for highlighting.
func (s Style) Reverse(enabled bool) Style {
	s.reverse = enabled
	return s
}

// Render applies styling with ANSI escape codes.
func (s Style) Render(str string) string {
	var builder strings.Builder

	// Top margin
	for i := 0; i < s.marginTop; i++ {
		builder.WriteString("\n")
	}

	// Split content into lines
	lines := strings.Split(str, "\n")

	// Apply horizontal padding to each line
	padding := strings.Repeat(" ", s.paddingH)
	for i, line := range lines {
		lines[i] = padding + line + padding
	}

	// Calculate content width (for fixed width or border)
	contentWidth := s.width
	if contentWidth == 0 {
		for _, line := range lines {
			if len(line) > contentWidth {
				contentWidth = len(line)
			}
		}
	}

	// Pad lines to fixed width
	for i, line := range lines {
		if len(line) < contentWidth {
			lines[i] = line + strings.Repeat(" ", contentWidth-len(line))
		} else if len(line) > contentWidth && s.width > 0 {
			lines[i] = line[:contentWidth]
		}
	}

	// Pad to fixed height
	if s.height > 0 {
		for len(lines) < s.height {
			lines = append(lines, strings.Repeat(" ", contentWidth))
		}
		if len(lines) > s.height {
			lines = lines[:s.height]
		}
	}

	// Build ANSI escape sequence
	var codes []string
	if s.bold {
		codes = append(codes, "1")
	}
	if s.reverse {
		codes = append(codes, "7")
	}
	if s.fg != "" {
		codes = append(codes, fmt.Sprintf("38;5;%s", s.fg))
	}
	if s.bg != "" {
		codes = append(codes, fmt.Sprintf("48;5;%s", s.bg))
	}

	ansiStart := ""
	ansiEnd := ""
	if len(codes) > 0 {
		ansiStart = fmt.Sprintf("\033[%sm", strings.Join(codes, ";"))
		ansiEnd = "\033[0m"
	}

	// Render with or without border
	if s.border {
		// Border characters
		const (
			topLeft     = "┌"
			topRight    = "┐"
			bottomLeft  = "└"
			bottomRight = "┘"
			horizontal  = "─"
			vertical    = "│"
		)

		// Top border
		builder.WriteString(topLeft)
		builder.WriteString(strings.Repeat(horizontal, contentWidth))
		builder.WriteString(topRight)
		builder.WriteString("\n")

		// Content lines with side borders
		for _, line := range lines {
			builder.WriteString(vertical)
			builder.WriteString(ansiStart)
			builder.WriteString(line)
			builder.WriteString(ansiEnd)
			builder.WriteString(vertical)
			builder.WriteString("\n")
		}

		// Bottom border
		builder.WriteString(bottomLeft)
		builder.WriteString(strings.Repeat(horizontal, contentWidth))
		builder.WriteString(bottomRight)
	} else {
		// No border - just render styled content
		for i, line := range lines {
			builder.WriteString(ansiStart)
			builder.WriteString(line)
			builder.WriteString(ansiEnd)
			if i < len(lines)-1 {
				builder.WriteString("\n")
			}
		}
	}

	return builder.String()
}

// JoinVertical concatenates segments vertically.
func JoinVertical(_ Position, parts ...string) string {
	return strings.Join(parts, "\n")
}

// JoinHorizontal concatenates segments horizontally, aligning multi-line strings row by row.
func JoinHorizontal(_ Position, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	// Split each part into lines
	partLines := make([][]string, len(parts))
	maxLines := 0
	for i, part := range parts {
		partLines[i] = strings.Split(part, "\n")
		if len(partLines[i]) > maxLines {
			maxLines = len(partLines[i])
		}
	}

	// Calculate the visible width of each part (width of first line or longest line)
	partWidths := make([]int, len(parts))
	for i, lines := range partLines {
		for _, line := range lines {
			w := visibleWidth(line)
			if w > partWidths[i] {
				partWidths[i] = w
			}
		}
	}

	// Build result by concatenating corresponding lines
	var result strings.Builder
	for row := 0; row < maxLines; row++ {
		for i, lines := range partLines {
			if row < len(lines) {
				result.WriteString(lines[row])
				// Pad to width if not last part and line is shorter
				if i < len(parts)-1 {
					w := visibleWidth(lines[row])
					if w < partWidths[i] {
						result.WriteString(strings.Repeat(" ", partWidths[i]-w))
					}
				}
			} else {
				// Pad with spaces if this part has fewer lines
				if i < len(parts)-1 {
					result.WriteString(strings.Repeat(" ", partWidths[i]))
				}
			}
		}
		if row < maxLines-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// visibleWidth calculates the visible width of a string, ignoring ANSI escape codes.
func visibleWidth(s string) int {
	width := 0
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		width++
	}
	return width
}

// Position is a placeholder for compatibility.
type Position int

// Left is the default join position.
const Left Position = 0

// Top position for horizontal joins.
const Top Position = 1
