package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gstark/agent-manager/internal/db"
)

// editKind tracks whether we're editing a skill or a rule.
type editKind int

const (
	editSkill editKind = iota
	editRule
)

// field indexes for tab navigation
const (
	fieldName = iota
	fieldDescription
	fieldExtra // source (skill) or paths (rule)
	fieldBody
	fieldCount
)

type editorModel struct {
	kind        editKind
	original    string // original name for overwrite detection
	nameInput   textinput.Model
	descInput   textinput.Model
	extraInput  textinput.Model // source for skills, comma-separated paths for rules
	bodyArea    textarea.Model
	activeField int
	width       int
	height      int
	err         string
	saved       bool
}

func newEditorFromSkill(s *db.Skill, width, height int) editorModel {
	m := newEditor(editSkill, width, height)
	m.original = s.Name
	m.nameInput.SetValue(s.Name)
	m.descInput.SetValue(s.Description)
	m.extraInput.SetValue(s.Source)
	m.bodyArea.SetValue(s.Body)
	return m
}

func newEditorFromRule(r *db.Rule, width, height int) editorModel {
	m := newEditor(editRule, width, height)
	m.original = r.Name
	m.nameInput.SetValue(r.Name)
	m.descInput.SetValue(r.Description)
	m.extraInput.SetValue(strings.Join(r.Paths, ", "))
	m.bodyArea.SetValue(r.Body)
	return m
}

func newEditor(kind editKind, width, height int) editorModel {
	name := textinput.New()
	name.Placeholder = "name"
	name.CharLimit = 80
	name.Width = (width - 20)
	name.Focus()

	desc := textinput.New()
	desc.Placeholder = "description"
	desc.CharLimit = 200
	desc.Width = (width - 20)

	extra := textinput.New()
	if kind == editSkill {
		extra.Placeholder = "source (e.g. skills.sh/owner/repo@skill)"
	} else {
		extra.Placeholder = "paths (comma-separated, e.g. **/*.rb, **/*.go)"
	}
	extra.CharLimit = 200
	extra.Width = (width - 20)

	body := textarea.New()
	body.Placeholder = "Markdown body..."
	body.ShowLineNumbers = false
	body.SetWidth(width - 6)
	body.SetHeight(height - 16)

	return editorModel{
		kind:        kind,
		nameInput:   name,
		descInput:   desc,
		extraInput:  extra,
		bodyArea:    body,
		activeField: fieldName,
		width:       width,
		height:      height,
	}
}

func (m *editorModel) focusActive() tea.Cmd {
	m.nameInput.Blur()
	m.descInput.Blur()
	m.extraInput.Blur()
	m.bodyArea.Blur()

	switch m.activeField {
	case fieldName:
		return m.nameInput.Focus()
	case fieldDescription:
		return m.descInput.Focus()
	case fieldExtra:
		return m.extraInput.Focus()
	case fieldBody:
		return m.bodyArea.Focus()
	}
	return nil
}

func (m editorModel) save() (string, error) {
	name := strings.TrimSpace(m.nameInput.Value())
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}

	switch m.kind {
	case editSkill:
		// Delete old if name changed
		if m.original != "" && m.original != name {
			db.DeleteSkill(m.original)
		}
		s := &db.Skill{
			Name:        name,
			Description: strings.TrimSpace(m.descInput.Value()),
			Source:      strings.TrimSpace(m.extraInput.Value()),
			Body:        m.bodyArea.Value(),
		}
		return name, db.SaveSkill(s)

	case editRule:
		if m.original != "" && m.original != name {
			db.DeleteRule(m.original)
		}
		var paths []string
		raw := strings.TrimSpace(m.extraInput.Value())
		if raw != "" {
			for _, p := range strings.Split(raw, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					paths = append(paths, p)
				}
			}
		}
		r := &db.Rule{
			Name:        name,
			Description: strings.TrimSpace(m.descInput.Value()),
			Paths:       paths,
			Body:        m.bodyArea.Value(),
		}
		return name, db.SaveRule(r)
	}
	return "", fmt.Errorf("unknown edit kind")
}

func (m editorModel) Update(msg tea.Msg) (editorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			name, err := m.save()
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
			m.saved = true
			m.err = ""
			_ = name
			return m, nil

		case "esc":
			// Signal cancel — handled by parent
			return m, nil

		case "tab":
			if m.activeField == fieldBody {
				// In body, tab inserts text — use ctrl+n to move
				break
			}
			m.activeField = (m.activeField + 1) % fieldCount
			return m, m.focusActive()

		case "shift+tab":
			if m.activeField == fieldBody {
				break
			}
			m.activeField = (m.activeField + fieldCount - 1) % fieldCount
			return m, m.focusActive()

		case "ctrl+n":
			// Always moves to next field, even from body
			m.activeField = (m.activeField + 1) % fieldCount
			return m, m.focusActive()

		case "ctrl+p":
			m.activeField = (m.activeField + fieldCount - 1) % fieldCount
			return m, m.focusActive()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.nameInput.Width = (m.width - 20)
		m.descInput.Width = (m.width - 20)
		m.extraInput.Width = (m.width - 20)
		m.bodyArea.SetWidth(m.width - 6)
		m.bodyArea.SetHeight(m.height - 16)
	}

	// Delegate to active field
	var cmd tea.Cmd
	switch m.activeField {
	case fieldName:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case fieldDescription:
		m.descInput, cmd = m.descInput.Update(msg)
	case fieldExtra:
		m.extraInput, cmd = m.extraInput.Update(msg)
	case fieldBody:
		m.bodyArea, cmd = m.bodyArea.Update(msg)
	}
	return m, cmd
}

func (m editorModel) View() string {
	var b strings.Builder

	kindLabel := "Skill"
	extraLabel := "Source"
	if m.kind == editRule {
		kindLabel = "Rule"
		extraLabel = "Paths"
	}

	title := fmt.Sprintf("Edit %s", kindLabel)
	if m.original == "" {
		title = fmt.Sprintf("New %s", kindLabel)
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().
		Width(12).
		Align(lipgloss.Right).
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		PaddingRight(1)

	activeIndicator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)

	renderField := func(label string, field int, view string) {
		indicator := "  "
		if m.activeField == field {
			indicator = activeIndicator.Render("▸ ")
		}
		b.WriteString(indicator)
		b.WriteString(labelStyle.Render(label))
		b.WriteString(view)
		b.WriteString("\n")
	}

	renderField("Name", fieldName, m.nameInput.View())
	renderField("Description", fieldDescription, m.descInput.View())
	renderField(extraLabel, fieldExtra, m.extraInput.View())

	b.WriteString("\n")

	// Body with indicator
	indicator := "  "
	if m.activeField == fieldBody {
		indicator = activeIndicator.Render("▸ ")
	}
	b.WriteString(indicator)
	b.WriteString(labelStyle.Render("Body"))
	b.WriteString("\n")

	bodyBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6C6C6C")).
		Padding(0, 1)
	b.WriteString(bodyBorder.Render(m.bodyArea.View()))
	b.WriteString("\n\n")

	// Error
	if m.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
		b.WriteString(errStyle.Render("Error: " + m.err))
		b.WriteString("\n")
	}

	// Help
	help := "tab/shift+tab: fields • ctrl+n/p: fields (from body) • ctrl+s: save • esc: cancel"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}
