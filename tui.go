package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	baseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(2).
		Align(lipgloss.Center)

	activeTabStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Padding(0, 1)

	tabContainerStyle = lipgloss.NewStyle().
		MarginBottom(1)
)



type model struct {
	table      table.Model
	quitting   bool
	activeTab  int
	tabs       []string
	gitData    []TableData
	dockerData []TableData
	kubesData  []TableData
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 15) // Leave space for ASCII art, tabs, and instructions
		return m, nil
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "1":
			m.activeTab = 0
			m.updateTableForTab()
			return m, nil
		case "2":
			m.activeTab = 1
			m.updateTableForTab()
			return m, nil
		case "3":
			m.activeTab = 2
			m.updateTableForTab()
			return m, nil
		case "tab":
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			m.updateTableForTab()
			return m, nil
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *model) updateTableForTab() {
	var columns []table.Column
	var rows []table.Row

	switch m.activeTab {
	case 0: // Git tab
		columns = []table.Column{
			{Title: "Commit SHA", Width: 42},
			{Title: "PR Description", Width: 50},
			{Title: "Author", Width: 20},
			{Title: "Date", Width: 20},
		}
		for _, item := range m.gitData {
			rows = append(rows, table.Row{
				item.CommitSHA,
				truncateString(item.PRDescription, 50),
				"N/A", // Placeholder for author
				"N/A", // Placeholder for date
			})
		}
	case 1: // Docker tab
		columns = []table.Column{
			{Title: "Image ID", Width: 20},
			{Title: "Repository", Width: 30},
			{Title: "Tag", Width: 20},
			{Title: "Size", Width: 15},
			{Title: "Created", Width: 20},
		}
		for _, item := range m.dockerData {
			rows = append(rows, table.Row{
				truncateString(item.ImageID, 20),
				"local-container-registry",
				"latest",
				item.ImageSize,
				"N/A", // Placeholder for created date
			})
		}
	case 2: // Kubernetes tab
		columns = []table.Column{
			{Title: "Pod Name", Width: 30},
			{Title: "Namespace", Width: 20},
			{Title: "Status", Width: 15},
			{Title: "Restarts", Width: 10},
			{Title: "Age", Width: 15},
		}
		// Placeholder data for Kubernetes
		rows = append(rows, table.Row{
			"local-container-registry-pod",
			"default",
			"Running",
			"0",
			"N/A",
		})
	}

	m.table.SetColumns(columns)
	m.table.SetRows(rows)
}

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	asciiArt := `
██╗            ██████╗           ██████╗ 
██║           ██╔════╝           ██╔══██╗
██║           ██║                ██████╔╝
██║           ██║                ██╔══██╗
███████╗      ╚██████╗           ██║  ██║
       ╚══════╝ ocal  ╚═════╝ container ╚═╝  ╚═╝ egistry
`

	artStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Align(lipgloss.Center)

	styledArt := artStyle.Render(asciiArt)

	// Render tabs
	var tabsRender []string
	for i, tab := range m.tabs {
		if i == m.activeTab {
			tabsRender = append(tabsRender, activeTabStyle.Render(tab))
		} else {
			tabsRender = append(tabsRender, inactiveTabStyle.Render(tab))
		}
	}
	
	tabsRow := lipgloss.JoinHorizontal(lipgloss.Top, tabsRender...)
	tabs := tabContainerStyle.Render(tabsRow)

	instructions := "Press 1-3 to switch tabs, Tab to cycle, 'q' to quit"

	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", styledArt, tabs, baseStyle.Render(m.table.View()), instructions)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func startTUI(data []TableData) {
	// Initialize tabs
	tabs := []string{"Git", "Docker", "Kubernetes"}
	
	// Initialize Git tab columns and rows
	gitColumns := []table.Column{
		{Title: "Commit SHA", Width: 42},
		{Title: "PR Description", Width: 50},
		{Title: "Author", Width: 20},
		{Title: "Date", Width: 20},
	}

	var gitRows []table.Row
	for _, item := range data {
		gitRows = append(gitRows, table.Row{
			item.CommitSHA,
			truncateString(item.PRDescription, 50),
			"N/A", // Placeholder for author
			"N/A", // Placeholder for date
		})
	}

	t := table.New(
		table.WithColumns(gitColumns),
		table.WithRows(gitRows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := model{
		table:      t,
		activeTab:  0,
		tabs:       tabs,
		gitData:    data,
		dockerData: data, // Reuse same data for now
		kubesData:  data, // Reuse same data for now
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
