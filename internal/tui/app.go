package tui

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
)

type tab int

const (
	tabSkills tab = iota
	tabRules
	tabPacks
)

var tabNames = []string{"Skills", "Rules", "Packs"}

// view tracks which screen we're on.
type view int

const (
	viewList view = iota
	viewEditor
)

// listItem implements list.DefaultItem for use with the default delegate.
type listItem struct {
	name, desc string
	active     bool
}

func (i listItem) Title() string {
	if i.active {
		return "● " + i.name
	}
	return "  " + i.name
}
func (i listItem) Description() string { return i.desc }
func (i listItem) FilterValue() string { return i.name + " " + i.desc }

type model struct {
	activeTab  tab
	activeView view
	skillsList list.Model
	rulesList  list.Model
	packsList  list.Model
	editor     editorModel
	width      int
	height     int
	status     string
	projectDir string
	hasProject bool
	projectCfg *config.ProjectConfig
}

func newList(title string, items []list.Item) list.Model {
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 80, 20)
	l.Title = title
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	return l
}

func buildSkillItems(activeNames []string) []list.Item {
	skills, err := db.ListSkills()
	if err != nil {
		return nil
	}
	var active, inactive []list.Item
	for _, s := range skills {
		desc := s.Description
		if s.Source != "" {
			desc = fmt.Sprintf("[%s] %s", s.Source, desc)
		}
		item := listItem{name: s.Name, desc: desc, active: slices.Contains(activeNames, s.Name)}
		if item.active {
			active = append(active, item)
		} else {
			inactive = append(inactive, item)
		}
	}
	return append(active, inactive...)
}

func buildRuleItems(activeNames []string) []list.Item {
	rules, err := db.ListRules()
	if err != nil {
		return nil
	}
	var active, inactive []list.Item
	for _, r := range rules {
		item := listItem{name: r.Name, desc: r.Description, active: slices.Contains(activeNames, r.Name)}
		if item.active {
			active = append(active, item)
		} else {
			inactive = append(inactive, item)
		}
	}
	return append(active, inactive...)
}

func buildPackItems(activeNames []string) []list.Item {
	packs, err := db.ListPacks()
	if err != nil {
		return nil
	}
	var active, inactive []list.Item
	for _, p := range packs {
		desc := p.Description
		if desc == "" {
			desc = fmt.Sprintf("%d skills, %d rules", len(p.Skills), len(p.Rules))
		}
		item := listItem{name: p.Name, desc: desc, active: slices.Contains(activeNames, p.Name)}
		if item.active {
			active = append(active, item)
		} else {
			inactive = append(inactive, item)
		}
	}
	return append(active, inactive...)
}

func loadProjectConfig(dir string) *config.ProjectConfig {
	cfg, err := config.LoadProjectConfig(dir)
	if err != nil {
		return &config.ProjectConfig{}
	}
	return cfg
}

func initialModel() model {
	dir, _ := os.Getwd()
	_, err := os.Stat(config.ProjectConfigPath(dir))
	hasProject := err == nil
	var cfg *config.ProjectConfig
	if hasProject {
		cfg = loadProjectConfig(dir)
	} else {
		cfg = &config.ProjectConfig{}
	}
	return model{
		activeTab:  tabSkills,
		activeView: viewList,
		skillsList: newList("Skills", buildSkillItems(cfg.Skills)),
		rulesList:  newList("Rules", buildRuleItems(cfg.Rules)),
		packsList:  newList("Packs", buildPackItems(cfg.Packs)),
		projectDir: dir,
		hasProject: hasProject,
		projectCfg: cfg,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) activeList() *list.Model {
	switch m.activeTab {
	case tabRules:
		return &m.rulesList
	case tabPacks:
		return &m.packsList
	default:
		return &m.skillsList
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.activeView == viewEditor {
		return m.updateEditor(msg)
	}
	return m.updateList(msg)
}

func (m model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.activeList().FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.activeTab = (m.activeTab + 1) % 3
			m.status = ""
			return m, nil
		case "shift+tab":
			m.activeTab = (m.activeTab + 2) % 3
			m.status = ""
			return m, nil
		case "d":
			return m.deleteSelected()
		case "a":
			return m.toggleProject()
		case "n":
			return m.newItem()
		case "e", "enter":
			return m.openEditor()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		listHeight := m.height - 7
		m.skillsList.SetSize(m.width-4, listHeight)
		m.rulesList.SetSize(m.width-4, listHeight)
		m.packsList.SetSize(m.width-4, listHeight)
		return m, nil
	}

	var cmd tea.Cmd
	switch m.activeTab {
	case tabSkills:
		m.skillsList, cmd = m.skillsList.Update(msg)
	case tabRules:
		m.rulesList, cmd = m.rulesList.Update(msg)
	case tabPacks:
		m.packsList, cmd = m.packsList.Update(msg)
	}
	return m, cmd
}

func (m model) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.activeView = viewList
			if m.editor.saved {
				m.status = fmt.Sprintf("Saved %s", m.editor.nameInput.Value())
				switch m.editor.kind {
				case editSkill:
					m.skillsList.SetItems(buildSkillItems(m.projectCfg.Skills))
				case editRule:
					m.rulesList.SetItems(buildRuleItems(m.projectCfg.Rules))
				case editPack:
					m.packsList.SetItems(buildPackItems(m.projectCfg.Packs))
				}
			}
			return m, nil
		case "ctrl+s":
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(msg)
			if m.editor.saved {
				m.status = fmt.Sprintf("Saved %s", m.editor.nameInput.Value())
				switch m.editor.kind {
				case editSkill:
					m.skillsList.SetItems(buildSkillItems(m.projectCfg.Skills))
				case editRule:
					m.rulesList.SetItems(buildRuleItems(m.projectCfg.Rules))
				case editPack:
					m.packsList.SetItems(buildPackItems(m.projectCfg.Packs))
				}
				m.activeView = viewList
			}
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

func (m model) newItem() (tea.Model, tea.Cmd) {
	switch m.activeTab {
	case tabSkills:
		m.editor = newEditor(editSkill, m.width, m.height)
	case tabRules:
		m.editor = newEditor(editRule, m.width, m.height)
	case tabPacks:
		m.editor = newEditor(editPack, m.width, m.height)
	}
	m.activeView = viewEditor
	return m, m.editor.focusActive()
}

func (m model) openEditor() (tea.Model, tea.Cmd) {
	sel := m.activeList().SelectedItem()
	if sel == nil {
		return m, nil
	}
	item := sel.(listItem)

	switch m.activeTab {
	case tabSkills:
		s, err := db.LoadSkill(item.name)
		if err != nil {
			m.status = fmt.Sprintf("Error: %v", err)
			return m, nil
		}
		m.editor = newEditorFromSkill(s, m.width, m.height)
		m.activeView = viewEditor
		return m, m.editor.focusActive()

	case tabRules:
		r, err := db.LoadRule(item.name)
		if err != nil {
			m.status = fmt.Sprintf("Error: %v", err)
			return m, nil
		}
		m.editor = newEditorFromRule(r, m.width, m.height)
		m.activeView = viewEditor
		return m, m.editor.focusActive()

	case tabPacks:
		p, err := db.LoadPack(item.name)
		if err != nil {
			m.status = fmt.Sprintf("Error: %v", err)
			return m, nil
		}
		m.editor = newEditorFromPack(p, m.width, m.height)
		m.activeView = viewEditor
		return m, m.editor.focusActive()
	}
	return m, nil
}

func (m model) deleteSelected() (tea.Model, tea.Cmd) {
	l := m.activeList()
	sel := l.SelectedItem()
	if sel == nil {
		return m, nil
	}
	item := sel.(listItem)
	var err error
	switch m.activeTab {
	case tabSkills:
		err = db.DeleteSkill(item.name)
	case tabRules:
		err = db.DeleteRule(item.name)
	case tabPacks:
		err = db.DeletePack(item.name)
	}
	if err != nil {
		m.status = fmt.Sprintf("Error: %v", err)
		return m, nil
	}
	m.status = fmt.Sprintf("Deleted %s", item.name)
	switch m.activeTab {
	case tabSkills:
		m.skillsList.SetItems(buildSkillItems(m.projectCfg.Skills))
	case tabRules:
		m.rulesList.SetItems(buildRuleItems(m.projectCfg.Rules))
	case tabPacks:
		m.packsList.SetItems(buildPackItems(m.projectCfg.Packs))
	}
	return m, nil
}

func (m model) toggleProject() (tea.Model, tea.Cmd) {
	if !m.hasProject {
		m.status = "No .agent-manager.toml in current directory (run 'agm init' first)"
		return m, nil
	}

	sel := m.activeList().SelectedItem()
	if sel == nil {
		return m, nil
	}
	item := sel.(listItem)

	cfg, err := config.LoadProjectConfig(m.projectDir)
	if err != nil {
		m.status = fmt.Sprintf("Error loading config: %v", err)
		return m, nil
	}

	var kind, action string
	switch m.activeTab {
	case tabSkills:
		kind = "skill"
		if i := slices.Index(cfg.Skills, item.name); i >= 0 {
			cfg.Skills = slices.Delete(cfg.Skills, i, i+1)
			action = "Removed"
		} else {
			cfg.Skills = append(cfg.Skills, item.name)
			action = "Added"
		}
	case tabRules:
		kind = "rule"
		if i := slices.Index(cfg.Rules, item.name); i >= 0 {
			cfg.Rules = slices.Delete(cfg.Rules, i, i+1)
			action = "Removed"
		} else {
			cfg.Rules = append(cfg.Rules, item.name)
			action = "Added"
		}
	case tabPacks:
		kind = "pack"
		if i := slices.Index(cfg.Packs, item.name); i >= 0 {
			cfg.Packs = slices.Delete(cfg.Packs, i, i+1)
			action = "Removed"
		} else {
			cfg.Packs = append(cfg.Packs, item.name)
			action = "Added"
		}
	}

	if err := config.SaveProjectConfig(m.projectDir, cfg); err != nil {
		m.status = fmt.Sprintf("Error saving config: %v", err)
		return m, nil
	}

	m.projectCfg = cfg
	m.status = fmt.Sprintf("%s %s %q", action, kind, item.name)

	// Refresh current list to update active markers
	switch m.activeTab {
	case tabSkills:
		m.skillsList.SetItems(buildSkillItems(m.projectCfg.Skills))
	case tabRules:
		m.rulesList.SetItems(buildRuleItems(m.projectCfg.Rules))
	case tabPacks:
		m.packsList.SetItems(buildPackItems(m.projectCfg.Packs))
	}
	return m, nil
}

func (m model) View() string {
	if m.activeView == viewEditor {
		return m.editor.View()
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("agent-manager"))
	b.WriteString("\n")

	var tabs []string
	for i, name := range tabNames {
		if tab(i) == m.activeTab {
			tabs = append(tabs, activeTab.Render(name))
		} else {
			tabs = append(tabs, inactiveTab.Render(name))
		}
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
	b.WriteString("\n\n")

	b.WriteString(m.activeList().View())
	b.WriteString("\n")

	if m.status != "" {
		b.WriteString(statusStyle.Render(m.status))
		b.WriteString("\n")
	}

	help := "tab/shift+tab: switch • n: new • e: edit • d: delete • /: filter • q: quit"
	if m.hasProject {
		help = "tab/shift+tab: switch • n: new • e: edit • a: toggle project • d: delete • /: filter • q: quit"
	}
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
