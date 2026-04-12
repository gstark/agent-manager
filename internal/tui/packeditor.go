package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gstark/agent-manager/internal/db"
)

// packSection tracks which section of the pack editor is active.
type packSection int

const (
	sectionName packSection = iota
	sectionDescription
	sectionSkills
	sectionRules
	sectionSave
)

type toggleItem struct {
	name        string
	description string
	enabled     bool
}

type packEditorModel struct {
	original    string // original name for rename detection
	policies    []string
	nameInput   textinput.Model
	descInput   textinput.Model
	skills      []toggleItem
	rules       []toggleItem
	section     packSection
	cursor      int // cursor within skills/rules list
	width       int
	height      int
	err         string
	saved       bool
	quitting    bool
}

func newPackEditorModel(p *db.Pack) packEditorModel {
	name := textinput.New()
	name.Placeholder = "name"
	name.CharLimit = 80
	name.Focus()

	desc := textinput.New()
	desc.Placeholder = "description"
	desc.CharLimit = 200

	allSkills, _ := db.ListSkills()
	skillItems := make([]toggleItem, len(allSkills))
	for i, s := range allSkills {
		skillItems[i] = toggleItem{
			name:        s.Name,
			description: s.Description,
			enabled:     slices.Contains(p.Skills, s.Name),
		}
	}

	allRules, _ := db.ListRules()
	ruleItems := make([]toggleItem, len(allRules))
	for i, r := range allRules {
		ruleItems[i] = toggleItem{
			name:        r.Name,
			description: r.Description,
			enabled:     slices.Contains(p.Rules, r.Name),
		}
	}

	m := packEditorModel{
		original:  p.Name,
		policies:  p.Policies,
		nameInput: name,
		descInput: desc,
		skills:    skillItems,
		rules:     ruleItems,
		section:   sectionName,
	}

	m.nameInput.SetValue(p.Name)
	m.descInput.SetValue(p.Description)

	return m
}

func (m packEditorModel) save() error {
	name := strings.TrimSpace(m.nameInput.Value())
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if m.original != "" && m.original != name {
		db.DeletePack(m.original)
	}

	var skills []string
	for _, s := range m.skills {
		if s.enabled {
			skills = append(skills, s.name)
		}
	}
	var rules []string
	for _, r := range m.rules {
		if r.enabled {
			rules = append(rules, r.name)
		}
	}

	p := &db.Pack{
		Name:        name,
		Description: strings.TrimSpace(m.descInput.Value()),
		Skills:      skills,
		Rules:       rules,
		Policies:    m.policies,
	}
	return db.SavePack(p)
}

func (m *packEditorModel) focusSection() tea.Cmd {
	m.nameInput.Blur()
	m.descInput.Blur()
	switch m.section {
	case sectionName:
		return m.nameInput.Focus()
	case sectionDescription:
		return m.descInput.Focus()
	}
	return nil
}

// Update returns packEditorModel (not tea.Model) so it can be embedded in the TUI.
func (m packEditorModel) Update(msg tea.Msg) (packEditorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.nameInput.Width = m.width - 20
		m.descInput.Width = m.width - 20
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		switch key {
		case "ctrl+s":
			if err := m.save(); err != nil {
				m.err = err.Error()
				return m, nil
			}
			m.saved = true
			return m, nil
		case "esc":
			m.quitting = true
			return m, nil
		}

		// Section navigation
		switch key {
		case "tab":
			if m.section == sectionName || m.section == sectionDescription {
				m.section++
				m.cursor = 0
				return m, m.focusSection()
			}
			if m.section == sectionSkills {
				m.section = sectionRules
				m.cursor = 0
				return m, nil
			}
			if m.section == sectionRules {
				m.section = sectionSave
				m.cursor = 0
				return m, nil
			}
			if m.section == sectionSave {
				m.section = sectionName
				m.cursor = 0
				return m, m.focusSection()
			}
		case "shift+tab":
			switch m.section {
			case sectionName:
				m.section = sectionSave
			case sectionDescription:
				m.section = sectionName
			case sectionSkills:
				m.section = sectionDescription
			case sectionRules:
				m.section = sectionSkills
			case sectionSave:
				m.section = sectionRules
			}
			m.cursor = 0
			return m, m.focusSection()
		}

		// Section-specific keys
		switch m.section {
		case sectionSkills:
			switch key {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.skills)-1 {
					m.cursor++
				}
			case " ", "enter":
				if m.cursor < len(m.skills) {
					m.skills[m.cursor].enabled = !m.skills[m.cursor].enabled
				}
			}
			return m, nil

		case sectionRules:
			switch key {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.rules)-1 {
					m.cursor++
				}
			case " ", "enter":
				if m.cursor < len(m.rules) {
					m.rules[m.cursor].enabled = !m.rules[m.cursor].enabled
				}
			}
			return m, nil

		case sectionSave:
			if key == "enter" {
				if err := m.save(); err != nil {
					m.err = err.Error()
					return m, nil
				}
				m.saved = true
				return m, nil
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.section {
	case sectionName:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case sectionDescription:
		m.descInput, cmd = m.descInput.Update(msg)
	}
	return m, cmd
}

func (m packEditorModel) View() string {
	var b strings.Builder

	title := "Edit Pack"
	if m.original == "" {
		title = "New Pack"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().
		Width(12).
		Align(lipgloss.Right).
		Reverse(true).
		Bold(true).
		PaddingRight(1)

	activeIndicator := lipgloss.NewStyle().
		Foreground(accent).
		Bold(true)

	indicator := func(section packSection) string {
		if m.section == section {
			return activeIndicator.Render("▸ ")
		}
		return "  "
	}

	// Name
	b.WriteString(indicator(sectionName))
	b.WriteString(labelStyle.Render("Name"))
	b.WriteString(m.nameInput.View())
	b.WriteString("\n")

	// Description
	b.WriteString(indicator(sectionDescription))
	b.WriteString(labelStyle.Render("Description"))
	b.WriteString(m.descInput.View())
	b.WriteString("\n\n")

	// Skills section
	sectionHeader := lipgloss.NewStyle().Bold(true).Foreground(accent)
	dimStyle := lipgloss.NewStyle().Foreground(dimText)
	selectedLine := lipgloss.NewStyle().Reverse(true)

	b.WriteString(indicator(sectionSkills))
	b.WriteString(sectionHeader.Render("Skills"))
	if len(m.skills) == 0 {
		b.WriteString(dimStyle.Render("  (no skills found)"))
	}
	b.WriteString("\n")

	maxVisible := 10
	renderToggleList := func(items []toggleItem, cursor int, active bool) {
		start := 0
		if cursor >= maxVisible {
			start = cursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(items) {
			end = len(items)
		}

		if start > 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("    ↑ %d more\n", start)))
		}

		for i := start; i < end; i++ {
			item := items[i]
			check := "[ ]"
			if item.enabled {
				check = "[✓]"
			}
			line := fmt.Sprintf("    %s %s", check, item.name)
			if item.description != "" {
				line += dimStyle.Render("  " + item.description)
			}
			if active && i == cursor {
				checkAndName := fmt.Sprintf("    %s %s", check, item.name)
				b.WriteString(selectedLine.Render(checkAndName))
				if item.description != "" {
					b.WriteString(dimStyle.Render("  " + item.description))
				}
			} else {
				if item.enabled {
					b.WriteString(lipgloss.NewStyle().Foreground(accent).Render(line))
				} else {
					b.WriteString(line)
				}
			}
			b.WriteString("\n")
		}

		if end < len(items) {
			b.WriteString(dimStyle.Render(fmt.Sprintf("    ↓ %d more\n", len(items)-end)))
		}
	}

	renderToggleList(m.skills, m.cursor, m.section == sectionSkills)
	b.WriteString("\n")

	// Rules section
	b.WriteString(indicator(sectionRules))
	b.WriteString(sectionHeader.Render("Rules"))
	if len(m.rules) == 0 {
		b.WriteString(dimStyle.Render("  (no rules found)"))
	}
	b.WriteString("\n")

	renderToggleList(m.rules, m.cursor, m.section == sectionRules)
	b.WriteString("\n")

	// Save button
	saveStyle := lipgloss.NewStyle().
		Padding(0, 3).
		Bold(true)
	if m.section == sectionSave {
		saveStyle = saveStyle.
			Background(accent).
			Foreground(lipgloss.Color("#000000"))
	} else {
		saveStyle = saveStyle.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondary)
	}
	b.WriteString("  " + saveStyle.Render("Save") + "\n")

	// Error
	if m.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
		b.WriteString("\n" + errStyle.Render("Error: "+m.err) + "\n")
	}

	// Help
	help := "tab/shift+tab: sections • ↑/↓: navigate • space: toggle • enter: select • ctrl+s: save • esc: cancel"
	b.WriteString("\n" + helpStyle.Render(help))

	return b.String()
}

// standalonePackEditor wraps packEditorModel as a tea.Model for CLI use.
type standalonePackEditor struct {
	inner packEditorModel
}

func (m standalonePackEditor) Init() tea.Cmd {
	return textinput.Blink
}

func (m standalonePackEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kmsg, ok := msg.(tea.KeyMsg); ok && kmsg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.inner, cmd = m.inner.Update(msg)
	if m.inner.saved || m.inner.quitting {
		return m, tea.Quit
	}
	return m, cmd
}

func (m standalonePackEditor) View() string {
	if m.inner.quitting && m.inner.saved {
		return fmt.Sprintf("Saved pack %q\n", m.inner.nameInput.Value())
	}
	if m.inner.quitting {
		return ""
	}
	return m.inner.View()
}

// RunPackEditor launches a standalone pack editor TUI for the given pack.
func RunPackEditor(p *db.Pack) error {
	m := standalonePackEditor{inner: newPackEditorModel(p)}
	prog := tea.NewProgram(m, tea.WithAltScreen())
	_, err := prog.Run()
	return err
}
