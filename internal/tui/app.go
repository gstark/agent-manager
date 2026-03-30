package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gstark/agent-manager/internal/db"
)

type tab int

const (
	tabSkills tab = iota
	tabRules
	tabPacks
)

var tabNames = []string{"Skills", "Rules", "Packs"}

// listItem implements list.DefaultItem for use with the default delegate.
type listItem struct {
	name, desc string
}

func (i listItem) Title() string       { return i.name }
func (i listItem) Description() string { return i.desc }
func (i listItem) FilterValue() string { return i.name }

type model struct {
	activeTab  tab
	skillsList list.Model
	rulesList  list.Model
	packsList  list.Model
	width      int
	height     int
	status     string
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

func buildSkillItems() []list.Item {
	skills, err := db.ListSkills()
	if err != nil {
		return nil
	}
	items := make([]list.Item, len(skills))
	for i, s := range skills {
		desc := s.Description
		if s.Source != "" {
			desc = fmt.Sprintf("[%s] %s", s.Source, desc)
		}
		items[i] = listItem{name: s.Name, desc: desc}
	}
	return items
}

func buildRuleItems() []list.Item {
	rules, err := db.ListRules()
	if err != nil {
		return nil
	}
	items := make([]list.Item, len(rules))
	for i, r := range rules {
		items[i] = listItem{name: r.Name, desc: r.Description}
	}
	return items
}

func buildPackItems() []list.Item {
	packs, err := db.ListPacks()
	if err != nil {
		return nil
	}
	items := make([]list.Item, len(packs))
	for i, p := range packs {
		desc := p.Description
		if desc == "" {
			desc = fmt.Sprintf("%d skills, %d rules", len(p.Skills), len(p.Rules))
		}
		items[i] = listItem{name: p.Name, desc: desc}
	}
	return items
}

func initialModel() model {
	return model{
		activeTab:  tabSkills,
		skillsList: newList("Skills", buildSkillItems()),
		rulesList:  newList("Rules", buildRuleItems()),
		packsList:  newList("Packs", buildPackItems()),
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't intercept keys when filtering
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
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		listHeight := m.height - 7 // title + tabs + status + help
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
	// Rebuild the active list
	switch m.activeTab {
	case tabSkills:
		m.skillsList.SetItems(buildSkillItems())
	case tabRules:
		m.rulesList.SetItems(buildRuleItems())
	case tabPacks:
		m.packsList.SetItems(buildPackItems())
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("agent-manager"))
	b.WriteString("\n")

	// Tab bar
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

	// Active list
	b.WriteString(m.activeList().View())
	b.WriteString("\n")

	// Status
	if m.status != "" {
		b.WriteString(statusStyle.Render(m.status))
		b.WriteString("\n")
	}

	// Help
	b.WriteString(helpStyle.Render("tab/shift+tab: switch tabs • d: delete • /: filter • q: quit"))

	return b.String()
}

func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
