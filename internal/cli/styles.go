package cli

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// ─── Color constants (ANSI 256 basic) ────────────────────────────────

var (
	colorGreen   = lipgloss.Color("2")
	colorRed     = lipgloss.Color("1")
	colorYellow  = lipgloss.Color("3")
	colorCyan    = lipgloss.Color("6")
	colorBlue    = lipgloss.Color("4")
	colorMagenta = lipgloss.Color("5")
	colorGray    = lipgloss.Color("8")
)

// ─── Semantic styles ─────────────────────────────────────────────────

var (
	StyleSuccess = lipgloss.NewStyle().Foreground(colorGreen)
	StyleError   = lipgloss.NewStyle().Foreground(colorRed)
	StyleWarning = lipgloss.NewStyle().Foreground(colorYellow)
	StyleInfo    = lipgloss.NewStyle().Foreground(colorCyan)
	StyleDim     = lipgloss.NewStyle().Foreground(colorGray)
	StyleBold    = lipgloss.NewStyle().Bold(true)
)

// ─── Element styles ──────────────────────────────────────────────────

var (
	StyleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)
	StyleID    = lipgloss.NewStyle().Foreground(colorCyan)
	StyleLabel = lipgloss.NewStyle().Foreground(colorMagenta)
	StyleKey   = lipgloss.NewStyle().Bold(true)
)

// ─── Status / Priority helpers ───────────────────────────────────────

// StatusStyle returns the appropriate style for a given status.
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "done":
		return lipgloss.NewStyle().Foreground(colorGreen)
	case "in-progress":
		return lipgloss.NewStyle().Foreground(colorCyan)
	case "in-review":
		return lipgloss.NewStyle().Foreground(colorBlue)
	case "blocked":
		return lipgloss.NewStyle().Foreground(colorRed)
	case "on-hold":
		return lipgloss.NewStyle().Foreground(colorYellow)
	case "urgent":
		return lipgloss.NewStyle().Bold(true).Foreground(colorRed)
	default: // "todo"
		return lipgloss.NewStyle().Foreground(colorGray)
	}
}

// PriorityStyle returns the appropriate style for a given priority.
func PriorityStyle(priority string) lipgloss.Style {
	switch priority {
	case "high":
		return lipgloss.NewStyle().Foreground(colorRed)
	case "low":
		return lipgloss.NewStyle().Foreground(colorGray)
	default: // "medium"
		return lipgloss.NewStyle().Foreground(colorYellow)
	}
}

// RenderStatusBadge renders a status string with its appropriate color.
func RenderStatusBadge(status string) string {
	return StatusStyle(status).Render(status)
}

// RenderPriorityBadge renders a priority string with its appropriate color.
func RenderPriorityBadge(priority string) string {
	return PriorityStyle(priority).Render(priority)
}

// ─── Render helpers ──────────────────────────────────────────────────

// RenderSuccess renders a green checkmark + message.
func RenderSuccess(msg string) string {
	return StyleSuccess.Render("✓ " + msg)
}

// RenderError renders a red cross + message.
func RenderError(msg string) string {
	return StyleError.Render("✗ " + msg)
}

// RenderKeyValue renders a styled "key: value" pair.
func RenderKeyValue(key, value string) string {
	return fmt.Sprintf("%s %s", StyleDim.Render(key+":"), value)
}

// RenderLabels renders labels joined with commas, each in magenta.
func RenderLabels(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	styled := make([]string, len(labels))
	for i, l := range labels {
		styled[i] = StyleLabel.Render(l)
	}
	return strings.Join(styled, StyleDim.Render(", "))
}

// RenderTags renders tags in magenta (alias for labels).
func RenderTags(tags []string) string {
	return RenderLabels(tags)
}

// RenderSeparator renders a horizontal line in dim gray.
func RenderSeparator(width int) string {
	return StyleDim.Render(strings.Repeat("─", width))
}

// RenderACCheckbox renders an acceptance criterion with index and checkbox.
func RenderACCheckbox(index int, text string, completed bool) string {
	num := StyleDim.Render(fmt.Sprintf("%d.", index))
	if completed {
		return fmt.Sprintf("  %s %s %s", num, StyleSuccess.Render("[✓]"), StyleDim.Render(text))
	}
	return fmt.Sprintf("  %s %s %s", num, StyleDim.Render("[ ]"), text)
}

// RenderSectionHeader renders a bold section header.
func RenderSectionHeader(title string) string {
	return StyleBold.Render(title)
}

// RenderBadge renders text in a colored badge style.
func RenderBadge(text string, c color.Color) string {
	return lipgloss.NewStyle().Foreground(c).Render("[" + text + "]")
}

// RenderTableHeader renders a table header row with bold styling.
func RenderTableHeader(columns ...string) string {
	styled := make([]string, len(columns))
	for i, col := range columns {
		styled[i] = StyleBold.Render(col)
	}
	return strings.Join(styled, "")
}
