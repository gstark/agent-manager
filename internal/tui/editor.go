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

// editKind tracks whether we're editing a skill, rule, or pack.
type editKind int

const (
	editSkill editKind = iota
	editRule
	editPack
)

// field indexes for tab navigation
const (
	fieldName = iota
	fieldDescription
	fieldExtra  // source (skill), paths (rule), or skills (pack)
	fieldExtra2 // rules (pack only)
	fieldBody
)

type editorModel struct {
	kind        editKind
	original    string // original name for overwrite detection
	nameInput   textinput.Model
	descInput   textinput.Model
	extraInput  textinput.Model // source for skills, paths for rules, skills for packs
	extra2Input textinput.Model // rules for packs
	bodyArea    textarea.Model
	activeField int
	width       int
	height      int
	err         string
	saved       bool
}

func (m editorModel) fieldCount() int {
	if m.kind == editPack {
		return 4 // name, desc, skills, rules (no body)
	}
	return 5 // name, desc, extra, (skip extra2), body
}

func (m editorModel) nextField(current int) int {
	next := current + 1
	if m.kind != editPack && next == fieldExtra2 {
		next++ // skip extra2 for skills/rules
	}
	if m.kind == editPack && next == fieldBody {
		next = fieldName // wrap around, packs have no body
	}
	if next > fieldBody {
		next = fieldName
	}
	return next
}

func (m editorModel) prevField(current int) int {
	prev := current - 1
	if prev < fieldName {
		if m.kind == editPack {
			prev = fieldExtra2
		} else {
			prev = fieldBody
		}
	}
	if m.kind != editPack && prev == fieldExtra2 {
		prev-- // skip extra2 for skills/rules
	}
	return prev
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

func newEditorFromPack(p *db.Pack, width, height int) editorModel {
	m := newEditor(editPack, width, height)
	m.original = p.Name
	m.nameInput.SetValue(p.Name)
	m.descInput.SetValue(p.Description)
	m.extraInput.SetValue(strings.Join(p.Skills, ", "))
	m.extra2Input.SetValue(strings.Join(p.Rules, ", "))
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
	switch kind {
	case editSkill:
		extra.Placeholder = "source (e.g. skills.sh/owner/repo@skill)"
	case editRule:
		extra.Placeholder = "paths (comma-separated, e.g. **/*.rb, **/*.go)"
	case editPack:
		extra.Placeholder = "skills (comma-separated)"
	}
	extra.CharLimit = 500
	extra.Width = (width - 20)

	extra2 := textinput.New()
	extra2.Placeholder = "rules (comma-separated)"
	extra2.CharLimit = 500
	extra2.Width = (width - 20)

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
		extra2Input: extra2,
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
	m.extra2Input.Blur()
	m.bodyArea.Blur()

	switch m.activeField {
	case fieldName:
		return m.nameInput.Focus()
	case fieldDescription:
		return m.descInput.Focus()
	case fieldExtra:
		return m.extraInput.Focus()
	case fieldExtra2:
		return m.extra2Input.Focus()
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
		r := &db.Rule{
			Name:        name,
			Description: strings.TrimSpace(m.descInput.Value()),
			Paths:       splitCommaList(m.extraInput.Value()),
			Body:        m.bodyArea.Value(),
		}
		return name, db.SaveRule(r)

	case editPack:
		if m.original != "" && m.original != name {
			db.DeletePack(m.original)
		}
		p := &db.Pack{
			Name:        name,
			Description: strings.TrimSpace(m.descInput.Value()),
			Skills:      splitCommaList(m.extraInput.Value()),
			Rules:       splitCommaList(m.extra2Input.Value()),
		}
		return name, db.SavePack(p)
	}
	return "", fmt.Errorf("unknown edit kind")
}

func splitCommaList(s string) []string {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return nil
	}
	var result []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
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
			return m, nil

		case "tab":
			if m.activeField == fieldBody {
				break
			}
			m.activeField = m.nextField(m.activeField)
			return m, m.focusActive()

		case "shift+tab":
			if m.activeField == fieldBody {
				break
			}
			m.activeField = m.prevField(m.activeField)
			return m, m.focusActive()

		case "ctrl+n":
			m.activeField = m.nextField(m.activeField)
			return m, m.focusActive()

		case "ctrl+p":
			m.activeField = m.prevField(m.activeField)
			return m, m.focusActive()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.nameInput.Width = (m.width - 20)
		m.descInput.Width = (m.width - 20)
		m.extraInput.Width = (m.width - 20)
		m.extra2Input.Width = (m.width - 20)
		m.bodyArea.SetWidth(m.width - 6)
		m.bodyArea.SetHeight(m.height - 16)
	}

	var cmd tea.Cmd
	switch m.activeField {
	case fieldName:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case fieldDescription:
		m.descInput, cmd = m.descInput.Update(msg)
	case fieldExtra:
		m.extraInput, cmd = m.extraInput.Update(msg)
	case fieldExtra2:
		m.extra2Input, cmd = m.extra2Input.Update(msg)
	case fieldBody:
		m.bodyArea, cmd = m.bodyArea.Update(msg)
	}
	return m, cmd
}

func (m editorModel) View() string {
	var b strings.Builder

	var kindLabel, extraLabel string
	switch m.kind {
	case editSkill:
		kindLabel = "Skill"
		extraLabel = "Source"
	case editRule:
		kindLabel = "Rule"
		extraLabel = "Paths"
	case editPack:
		kindLabel = "Pack"
		extraLabel = "Skills"
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

	if m.kind == editPack {
		renderField("Rules", fieldExtra2, m.extra2Input.View())
	}

	if m.kind != editPack {
		b.WriteString("\n")

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
	} else {
		b.WriteString("\n")
	}

	if m.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
		b.WriteString(errStyle.Render("Error: " + m.err))
		b.WriteString("\n")
	}

	help := "tab/shift+tab: fields • ctrl+n/p: fields (from body) • ctrl+s: save • esc: cancel"
	if m.kind == editPack {
		help = "tab/shift+tab: fields • ctrl+s: save • esc: cancel"
	}
	b.WriteString(helpStyle.Render(help))

	return b.String()
}
