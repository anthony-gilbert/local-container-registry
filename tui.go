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
)

type TableData struct {
	CommitSHA     string
	PRDescription string
	ImageID       string
	ImageSize     string
	ImageTag      string
}

type model struct {
	table    table.Model
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 12) // Leave space for ASCII art and instructions
		return m, nil
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
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

	return fmt.Sprintf("%s\n\n%s\n\nPress 'q' to quit", styledArt, baseStyle.Render(m.table.View()))
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func startTUI(data []TableData) {
	columns := []table.Column{
		{Title: "Commit SHA", Width: 42},
		{Title: "PR Description", Width: 35},
		{Title: "Image ID", Width: 15},
		{Title: "Image Size", Width: 12},
		{Title: "Image Tag", Width: 15},
	}

	var rows []table.Row
	for _, item := range data {
		rows = append(rows, table.Row{
			item.CommitSHA, // Show full SHA without truncation
			truncateString(item.PRDescription, 35),
			truncateString(item.ImageID, 15),
			truncateString(item.ImageSize, 12),
			truncateString(item.ImageTag, 15),
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
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

	m := model{table: t}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
