// Package tui provides an interactive terminal user interface for configuring RepoDocs.
package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Theme colors
	primaryColor = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
	successColor = lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}
	errorColor   = lipgloss.AdaptiveColor{Light: "#FE5F86", Dark: "#FE5F86"}
	mutedColor   = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}
	warnColor    = lipgloss.AdaptiveColor{Light: "#FF9500", Dark: "#FFAA33"}

	// TitleStyle is used for main headers
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	// SubtitleStyle is used for section headers
	SubtitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(mutedColor)

	// DescriptionStyle is used for help text
	DescriptionStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	// SuccessStyle is used for success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor)

	// ErrorStyle is used for error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	// WarnStyle is used for warning messages
	WarnStyle = lipgloss.NewStyle().
			Foreground(warnColor)

	// BoxStyle is used for bordered containers
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	// SelectedStyle is used for highlighted menu items
	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	// UnselectedStyle is used for normal menu items
	UnselectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	// HelpStyle is used for keyboard shortcut hints
	HelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)
)

// GetTheme returns the huh theme for forms
func GetTheme() *huh.Theme {
	return huh.ThemeCharm()
}

// GetAccessibleTheme returns an accessible theme for screen readers
func GetAccessibleTheme() *huh.Theme {
	return huh.ThemeBase()
}
