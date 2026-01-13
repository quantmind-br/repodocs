package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/quantmind-br/repodocs-go/internal/config"
)

type state int

const (
	stateMenu state = iota
	stateForm
	stateConfirm
	stateSaved
	stateError
)

type Model struct {
	state          state
	values         *ConfigValues
	originalConfig *config.Config
	menuIndex      int
	currentForm    *huh.Form
	err            error
	width          int
	height         int
	dirty          bool
	saveFunc       func(*config.Config) error
	accessible     bool
}

type Options struct {
	Config     *config.Config
	SaveFunc   func(*config.Config) error
	Accessible bool
}

func NewModel(opts Options) Model {
	cfg := opts.Config
	if cfg == nil {
		cfg = config.Default()
	}

	return Model{
		state:          stateMenu,
		values:         FromConfig(cfg),
		originalConfig: cfg,
		saveFunc:       opts.SaveFunc,
		accessible:     opts.Accessible,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.state == stateForm && m.currentForm != nil {
			form, cmd := m.currentForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.currentForm = f
			}
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case stateMenu:
			return m.updateMenu(msg)
		case stateForm:
			return m.updateForm(msg)
		case stateConfirm:
			return m.updateConfirm(msg)
		case stateSaved, stateError:
			return m, tea.Quit
		}
	}

	if m.state == stateForm && m.currentForm != nil {
		form, cmd := m.currentForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.currentForm = f
		}
		if m.currentForm.State == huh.StateCompleted {
			m.dirty = true
			m.state = stateMenu
			return m, nil
		}
		return m, cmd
	}

	return m, nil
}

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if m.dirty {
			m.state = stateConfirm
			return m, nil
		}
		return m, tea.Quit

	case "up", "k":
		if m.menuIndex > 0 {
			m.menuIndex--
		}

	case "down", "j":
		if m.menuIndex < len(Categories) {
			m.menuIndex++
		}

	case "enter":
		if m.menuIndex == len(Categories) {
			return m.handleSave()
		}
		m.state = stateForm
		category := Categories[m.menuIndex]
		m.currentForm = GetFormForCategory(category.ID, m.values)
		if m.accessible {
			m.currentForm = m.currentForm.WithAccessible(true)
		}
		return m, m.currentForm.Init()

	case "s":
		return m.handleSave()

	case "esc":
		if m.dirty {
			m.state = stateConfirm
			return m, nil
		}
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.state = stateMenu
		return m, nil
	}
	if m.currentForm != nil {
		form, cmd := m.currentForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.currentForm = f
		}
		if m.currentForm.State == huh.StateCompleted {
			m.dirty = true
			m.state = stateMenu
			return m, nil
		}
		return m, cmd
	}
	return m, nil
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.handleSave()
	case "n", "N", "esc":
		return m, tea.Quit
	case "c":
		m.state = stateMenu
	}
	return m, nil
}

func (m Model) handleSave() (tea.Model, tea.Cmd) {
	cfg, err := m.values.ToConfig()
	if err != nil {
		m.state = stateError
		m.err = err
		return m, nil
	}

	if m.saveFunc != nil {
		if err := m.saveFunc(cfg); err != nil {
			m.state = stateError
			m.err = err
			return m, nil
		}
	}

	m.state = stateSaved
	m.dirty = false
	return m, nil
}

func (m Model) View() string {
	var s strings.Builder

	header := TitleStyle.Render("RepoDocs Configuration")
	s.WriteString(header)
	s.WriteString("\n\n")

	switch m.state {
	case stateMenu:
		s.WriteString(m.renderMenu())
	case stateForm:
		if m.currentForm != nil {
			s.WriteString(m.currentForm.View())
		}
	case stateConfirm:
		s.WriteString(m.renderConfirm())
	case stateSaved:
		s.WriteString(SuccessStyle.Render("Configuration saved successfully!"))
		s.WriteString("\n\nPress any key to exit.")
	case stateError:
		s.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n\nPress any key to exit.")
	}

	return s.String()
}

func (m Model) renderMenu() string {
	var s strings.Builder

	for i, cat := range Categories {
		cursor := "  "
		style := UnselectedStyle
		if i == m.menuIndex {
			cursor = "> "
			style = SelectedStyle
		}
		line := fmt.Sprintf("%s%s %s", cursor, cat.Icon, cat.Name)
		s.WriteString(style.Render(line))
		if i == m.menuIndex {
			s.WriteString(DescriptionStyle.Render("  " + cat.Description))
		}
		s.WriteString("\n")
	}

	saveStyle := UnselectedStyle
	saveCursor := "  "
	if m.menuIndex == len(Categories) {
		saveCursor = "> "
		saveStyle = SelectedStyle
	}
	saveText := fmt.Sprintf("%s Save Configuration", saveCursor)
	if m.dirty {
		saveText += " *"
	}
	s.WriteString("\n")
	s.WriteString(saveStyle.Render(saveText))
	s.WriteString("\n\n")

	help := HelpStyle.Render("↑/↓ navigate • enter select • s save • q quit")
	s.WriteString(help)

	return s.String()
}

func (m Model) renderConfirm() string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(warnColor).
		Padding(1, 2).
		Render("You have unsaved changes.\n\nSave before quitting?\n\n[y] Yes  [n] No  [c] Cancel")

	return box
}

func Run(opts Options) error {
	p := tea.NewProgram(NewModel(opts), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
