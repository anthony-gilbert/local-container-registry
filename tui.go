package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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

	tabContainerStyle = lipgloss.NewStyle()

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	modalStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			Background(lipgloss.Color("#1A1A1A")).
			Width(50).
			Height(10)

	overlayStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#000000")).
			Width(100).
			Height(100)
)

type model struct {
	table           table.Model
	quitting        bool
	activeTab       int
	tabs            []string
	gitData         []TableData
	dockerData      []TableData
	kubesData       []TableData
	width           int
	height          int
	showModal       bool
	selectedImage   string
	showPodDef      bool
	selectedPod     string
	podDefTable     table.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case dockerDeleteMsg:
		if msg.success {
			// Refresh Docker data after successful deletion
			return m, m.refreshDockerData()
		}
		// Handle deletion error (could show a message to user)
		return m, nil
	case dockerRefreshMsg:
		// Update Docker data and refresh table
		m.dockerData = msg.data
		if m.activeTab == 1 {
			m.updateTableForTab()
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 15) // Leave space for ASCII art, tabs, and instructions
		// Also update pod definition table if it exists
		if m.podDefTable.Columns() != nil {
			m.podDefTable.SetWidth(msg.Width)
			m.podDefTable.SetHeight(msg.Height - 15)
		}
		return m, nil
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "1":
			if m.showModal {
				// Deploy to Kubernetes action
				m.showModal = false
				// TODO: Add actual deployment logic here
				return m, nil
			} else {
				// Switch to Git tab
				m.activeTab = 0
				m.updateTableForTab()
				return m, nil
			}
		case "2":
			if m.showModal {
				// Cancel deployment
				m.showModal = false
				return m, nil
			} else {
				// Switch to Docker tab
				m.activeTab = 1
				m.updateTableForTab()
				return m, nil
			}
		case "3":
			if m.showModal {
				// No action for 3 in modal
				return m, nil
			} else {
				// Switch to Kubernetes tab
				m.activeTab = 2
				m.updateTableForTab()
				return m, nil
			}
		case "tab":
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			m.updateTableForTab()
			return m, nil
		case "enter":
			// Show modal on Docker tab or pod definition on Kubernetes tab
			if m.activeTab == 1 && len(m.dockerData) > 0 {
				selectedRow := m.table.Cursor()
				if selectedRow < len(m.dockerData) {
					m.selectedImage = m.dockerData[selectedRow].ImageID
					m.showModal = true
				}
			} else if m.activeTab == 2 && len(m.kubesData) > 0 {
				selectedRow := m.table.Cursor()
				if selectedRow < len(m.kubesData) {
					m.selectedPod = "local-container-registry-pod" // Use actual pod name
					m.showPodDef = true
					m.initPodDefTable()
				}
			}
			return m, nil
		case "esc":
			// Close modal or pod definition view if open
			if m.showModal {
				m.showModal = false
				return m, nil
			} else if m.showPodDef {
				m.showPodDef = false
				return m, nil
			}
		case "ctrl+d":
			// Delete Docker image when on Docker tab
			if m.activeTab == 1 && len(m.dockerData) > 0 && !m.showModal {
				selectedRow := m.table.Cursor()
				if selectedRow < len(m.dockerData) {
					imageID := m.dockerData[selectedRow].ImageID
					return m, m.deleteDockerImage(imageID)
				}
			}
		}
	}

	// Update the appropriate table based on current view
	if m.showPodDef {
		m.podDefTable, cmd = m.podDefTable.Update(msg)
	} else {
		m.table, cmd = m.table.Update(msg)
	}
	return m, cmd
}

func (m *model) updateTableForTab() {
	// Add panic recovery to prevent unexpected exits
	defer func() {
		if r := recover(); r != nil {
			// If there's a panic, just return without doing anything
			// This prevents the program from crashing
			return
		}
	}()

	// Validate that we have a valid table
	if m.table.Columns() == nil {
		return
	}

	var columns []table.Column
	var rows []table.Row

	switch m.activeTab {
	case 0: // Git tab
		columns = []table.Column{
			{Title: "Commit SHA", Width: 42},
			{Title: "PR Description", Width: 40},
			{Title: "Author", Width: 20},
			{Title: "PushedAt", Width: 20},
		}
		if len(m.gitData) > 0 {
			for _, item := range m.gitData {
				rows = append(rows, table.Row{
					item.CommitSHA,
					truncateString(item.PRDescription, 40),
					"N/A", // Placeholder for author
					item.PushedAt,
				})
			}
		} else {
			// Add a placeholder row if no data
			rows = append(rows, table.Row{
				"No data available",
				"",
				"",
				"",
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
				item.CreatedAt,
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
	default:
		// Default to Git tab if something goes wrong
		columns = []table.Column{
			{Title: "Commit SHA", Width: 42},
			{Title: "PR Description", Width: 40},
			{Title: "Author", Width: 20},
			{Title: "PushedAt", Width: 20},
		}
		for _, item := range m.gitData {
			rows = append(rows, table.Row{
				item.CommitSHA,
				truncateString(item.PRDescription, 40),
				"N/A", // Placeholder for author
				item.PushedAt,
			})
		}
	}

	// Safely update table with error handling
	if len(columns) > 0 {
		// Use defer to catch any panics from SetColumns
		func() {
			defer func() {
				if r := recover(); r != nil {
					// If SetColumns panics, just ignore it
					return
				}
			}()
			m.table.SetColumns(columns)
		}()
	}

	if len(rows) >= 0 {
		// Use defer to catch any panics from SetRows
		func() {
			defer func() {
				if r := recover(); r != nil {
					// If SetRows panics, just ignore it
					return
				}
			}()
			m.table.SetRows(rows)
		}()
	}
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

	// Render tabs with spacing
	var tabsRender []string
	for i, tab := range m.tabs {
		if i == m.activeTab {
			tabsRender = append(tabsRender, activeTabStyle.Render(tab))
		} else {
			tabsRender = append(tabsRender, inactiveTabStyle.Render(tab))
		}
		// Add space after each tab except the last one
		if i < len(m.tabs)-1 {
			tabsRender = append(tabsRender, " ")
		}
	}

	tabsRow := lipgloss.JoinHorizontal(lipgloss.Top, tabsRender...)
	tabs := tabContainerStyle.Render(tabsRow)

	instructions := "Press 1-3 to switch tabs, Tab to cycle, Enter to deploy/view, Ctrl+D to delete (Docker), 'q' to quit"

	// Create border style with proper width that encompasses both tabs and table
	containerStyle := baseStyle.Width(m.width - 2) // Account for border padding
	
	// Create separator between tabs and table
	separatorWidth := m.width - 4
	if separatorWidth < 0 {
		separatorWidth = 0 // Prevent negative repeat count
	}
	separatorLine := strings.Repeat("─", separatorWidth)
	separator := separatorStyle.Render(separatorLine)
	
	// Combine tabs, separator, and table, then apply border around all
	tabsAndTable := lipgloss.JoinVertical(lipgloss.Left, tabs, separator, m.table.View())
	borderedContainer := containerStyle.Render(tabsAndTable)
	
	mainView := fmt.Sprintf("%s\n\n%s\n\n%s", styledArt, borderedContainer, instructions)
	
	// Show modal if active
	if m.showModal {
		modal := m.renderModal()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal, lipgloss.WithWhitespaceChars("░"))
	}
	
	// Show pod definition view if active
	if m.showPodDef {
		return m.renderPodDefView()
	}
	
	return mainView
}

func (m model) renderModal() string {
	modalContent := fmt.Sprintf(`Deploy Docker Image

Selected Image: %s

Options:
[1] Deploy to Kubernetes
[2] Cancel

Press 1 to deploy, 2 to cancel, or ESC to close`, m.selectedImage)

	return modalStyle.Render(modalContent)
}

func (m model) renderPodDefView() string {
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

	title := fmt.Sprintf("Pod Definition: %s", m.selectedPod)
	titleStyled := titleStyle.Render(title)

	instructions := "Press ESC to go back to main view"

	// Create border style with proper width
	containerStyle := baseStyle.Width(m.width - 2)
	borderedTable := containerStyle.Render(m.podDefTable.View())

	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", styledArt, titleStyled, borderedTable, instructions)
}

func (m *model) initPodDefTable() {
	columns := []table.Column{
		{Title: "Key", Width: 30},
		{Title: "Value", Width: 60},
	}

	rows := []table.Row{
		{"Name", m.selectedPod},
		{"Namespace", "default"},
		{"Status", "Running"},
		{"Image", "local-container-registry:latest"},
		{"Restart Policy", "Always"},
		{"CPU Request", "100m"},
		{"Memory Request", "128Mi"},
		{"CPU Limit", "500m"},
		{"Memory Limit", "512Mi"},
		{"Port", "8080"},
		{"Created", "2024-01-15 10:30:00"},
		{"Labels", "app=local-container-registry"},
	}

	m.podDefTable = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(15),
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
	m.podDefTable.SetStyles(s)
}

// Message types for async operations
type dockerDeleteMsg struct {
	success bool
	imageID string
	err     error
}

func (m model) deleteDockerImage(imageID string) tea.Cmd {
	return func() tea.Msg {
		// Execute docker rmi command
		cmd := exec.Command("docker", "rmi", "-f", imageID)
		err := cmd.Run()
		
		return dockerDeleteMsg{
			success: err == nil,
			imageID: imageID,
			err:     err,
		}
	}
}

func (m model) refreshDockerData() tea.Cmd {
	return func() tea.Msg {
		// Get fresh Docker data
		dockerImages, err := getDockerImagesInfo()
		if err != nil {
			return dockerDeleteMsg{success: false, err: err}
		}
		
		// Convert to table data format
		var dockerTableData []TableData
		for _, dockerImg := range dockerImages {
			imageID := dockerImg.ID
			if len(imageID) > 12 {
				imageID = imageID[:12]
			}

			imageTag := "N/A"
			if len(dockerImg.RepoTags) > 0 && dockerImg.RepoTags[0] != "<none>:<none>" {
				imageTag = dockerImg.RepoTags[0]
			}

			imageSize := dockerImg.Size
			if dockerImg.Size == "" || dockerImg.Size == "N/A" {
				imageSize = "N/A"
			}

			dockerTableData = append(dockerTableData, TableData{
				ImageID:   imageID,
				ImageSize: imageSize,
				ImageTag:  imageTag,
				CreatedAt: dockerImg.CreatedAt,
			})
		}
		
		return dockerRefreshMsg{data: dockerTableData}
	}
}

type dockerRefreshMsg struct {
	data []TableData
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func startTUI(gitData []TableData, dockerData []TableData) {
	// Initialize tabs
	tabs := []string{"Git", "Docker", "Kubernetes"}

	// Initialize Git tab columns and rows
	gitColumns := []table.Column{
		{Title: "Commit SHA", Width: 42},
		{Title: "PR Description", Width: 40},
		{Title: "Author", Width: 20},
		{Title: "PushedAt", Width: 20},
	}

	var gitRows []table.Row
	for _, item := range gitData {
		gitRows = append(gitRows, table.Row{
			item.CommitSHA,
			truncateString(item.PRDescription, 40),
			"N/A", // Placeholder for author
			item.PushedAt,
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
		gitData:    gitData,
		dockerData: dockerData,
		kubesData:  gitData, // Reuse git data for kubernetes tab
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
