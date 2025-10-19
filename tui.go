package main

import (
	"fmt"
	"log"
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
	table              table.Model
	quitting           bool
	activeTab          int
	tabs               []string
	gitData            []TableData
	dockerData         []TableData
	kubesData          []TableData
	width              int
	height             int
	showModal          bool
	selectedImage      string
	showPodDef         bool
	selectedPod        string
	selectedPodNS      string
	podDefTable        table.Model
	deployments        []TableData
	selectedDeployment int
	deploymentPods     []TableData
	selectedPod2       int
	modalStep          int // 0 = deployment selection, 1 = pod selection, 2 = confirmation
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case deploymentsMsg:
		m.deployments = msg.deployments
		return m, nil
	case deploymentPodsMsg:
		m.deploymentPods = msg.pods
		return m, nil
	case podDetailsMsg:
		if msg.err == nil {
			m.initPodDefTable(msg.details)
		} else {
			m.initPodDefTable(nil)
		}
		return m, nil
	case dockerDeleteMsg:
		if msg.success {
			// Refresh Docker data after successful deletion
			return m, m.refreshDockerData()
		}
		// Handle deletion error (could show a message to user)
		return m, nil
	case dockerPullMsg:
		if msg.success {
			// Refresh Docker data after successful pull
			return m, m.refreshDockerData()
		}
		// Handle pull error (could show a message to user)
		return m, nil
	case deploymentMsg:
		// Handle deployment result and reset table selection
		if msg.success {
			// Reset table cursor to first row after successful deployment
			m.table.SetCursor(0)
			// Refresh deployments list to show the new deployment
			return m, m.loadDeployments()
		} else {
			// Log the error for debugging
			if msg.err != nil {
				log.Printf("Deployment failed: %v", msg.err)
			}
		}
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
			// Handle quitting the application
			m.quitting = true
			return m, tea.Quit
		case "1":
			if m.showModal {
				if m.modalStep == 0 {
					if m.selectedDeployment == -1 {
						// Create new deployment - move to creation step
						m.modalStep = 1
						return m, nil
					} else {
						// Update existing deployment - move to confirmation step
						m.modalStep = 2
						return m, nil
					}
				} else if m.modalStep == 1 {
					// Create new deployment
					m.showModal = false
					m.modalStep = 0
					return m, m.createNewDeployment(m.selectedImage)
				} else {
					// Deploy to selected deployment
					m.showModal = false
					m.modalStep = 0
					if len(m.deployments) > 0 && m.selectedDeployment < len(m.deployments) {
						selectedDeployment := m.deployments[m.selectedDeployment]
						return m, m.deployImageToPod(m.selectedImage, selectedDeployment.PodName, selectedDeployment.Namespace)
					}
					return m, nil
				}
			} else {
				// Switch to Git tab
				m.activeTab = 0
				m.updateTableForTab()
				return m, nil
			}
		case "2":
			if m.showModal {
				if m.modalStep == 1 {
					// Go back to deployment selection from creation step
					m.modalStep = 0
					return m, nil
				} else if m.modalStep == 2 {
					// Go back to deployment selection from confirmation step
					m.modalStep = 0
					return m, nil
				} else {
					// Cancel deployment
					m.showModal = false
					m.modalStep = 0
					return m, nil
				}
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
					imageData := m.dockerData[selectedRow]
					m.selectedImage = imageData.ImageTag // Use full image name from registry
					if m.selectedImage == "" {
						m.selectedImage = imageData.ImageID
					}
					m.modalStep = 0
					m.selectedDeployment = -1 // Start with "Create New Deployment" selected
					m.selectedPod2 = 0
					m.showModal = true
					return m, m.loadDeployments()
				}
			} else if m.activeTab == 2 && len(m.kubesData) > 0 {
				selectedRow := m.table.Cursor()
				if selectedRow < len(m.kubesData) {
					m.selectedPod = m.kubesData[selectedRow].PodName
					m.selectedPodNS = m.kubesData[selectedRow].Namespace
					m.showPodDef = true
					return m, m.loadPodDetails()
				}
			}
			return m, nil
		case "esc":
			// Close modal or pod definition view if open, otherwise quit
			if m.showModal {
				m.showModal = false
				m.modalStep = 0
				return m, nil
			} else if m.showPodDef {
				m.showPodDef = false
				return m, nil
			} else {
				// No modal open, quit the application
				m.quitting = true
				return m, tea.Quit
			}
		case "up", "k":
			if m.showModal && m.modalStep == 0 {
				// Only allow navigation if there are deployments, otherwise stay on "Create New"
				if len(m.deployments) > 0 && m.selectedDeployment > -1 {
					m.selectedDeployment--
				}
				return m, nil
			}
		case "down", "j":
			if m.showModal && m.modalStep == 0 {
				// Allow moving from -1 (Create New) to 0 (first deployment) if deployments exist
				if len(m.deployments) > 0 && m.selectedDeployment < len(m.deployments)-1 {
					m.selectedDeployment++
				}
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
		case "ctrl+p":
			// Pull Docker image from registry when on Docker tab
			if m.activeTab == 1 && len(m.dockerData) > 0 && !m.showModal {
				selectedRow := m.table.Cursor()
				if selectedRow < len(m.dockerData) {
					imageTag := m.dockerData[selectedRow].ImageTag
					if imageTag != "" && imageTag != "N/A" {
						return m, m.pullDockerImage(imageTag)
					}
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
			{Title: "Tag", Width: 15},
			{Title: "Size", Width: 12},
			{Title: "Created", Width: 25},
		}
		for _, item := range m.dockerData {
			// Extract repository and tag from RepoTags
			repository := "N/A"
			tag := "N/A"
			if len(item.ImageTag) > 0 && item.ImageTag != "N/A" {
				// Handle localhost:5000/repo:tag format
				imageTag := item.ImageTag

				// Remove localhost:5000/ prefix if present for cleaner display
				if strings.HasPrefix(imageTag, "localhost:5000/") {
					imageTag = strings.TrimPrefix(imageTag, "localhost:5000/")
				}

				// Parse repository:tag format
				lastColonIndex := strings.LastIndex(imageTag, ":")
				if lastColonIndex > 0 {
					repository = imageTag[:lastColonIndex]
					tag = imageTag[lastColonIndex+1:]
				} else {
					repository = imageTag
					tag = "latest"
				}
			}

			rows = append(rows, table.Row{
				truncateString(item.ImageID, 20),
				truncateString(repository, 30),
				truncateString(tag, 15),
				truncateString(item.ImageSize, 12),
				truncateString(item.CreatedAt, 25),
			})
		}
	case 2: // Kubernetes tab
		columns = []table.Column{
			{Title: "Pod Name", Width: 35},
			{Title: "Namespace", Width: 15},
			{Title: "Status", Width: 12},
			{Title: "Restarts", Width: 10},
			{Title: "Age", Width: 15},
			{Title: "Node", Width: 20},
		}
		// Real Kubernetes data
		for _, item := range m.kubesData {
			rows = append(rows, table.Row{
				truncateString(item.PodName, 35),
				item.Namespace,
				item.Status,
				item.Restarts,
				item.Age,
				truncateString(item.NodeName, 20),
			})
		}
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

	instructions := "Press 1-3 to switch tabs, Tab to cycle, Enter to deploy/view, Ctrl+D to delete, Ctrl+P to pull (Docker), 'q' or ESC to quit"

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
	if m.modalStep == 0 {
		// Deployment selection step
		var modalContent strings.Builder
		modalContent.WriteString(fmt.Sprintf("Deploy Docker Image: %s\n\n", m.selectedImage))
		modalContent.WriteString("Select Kubernetes Deployment:\n\n")

		if len(m.deployments) == 0 {
			modalContent.WriteString("Loading deployments...\n\n")
			// Still show "Create New Deployment" option even when loading
			prefix := "→ " // Always selected when no deployments
			modalContent.WriteString(fmt.Sprintf("%s[Create New Deployment]\n\n", prefix))
		} else {
			// Add "Create New Deployment" option at the top
			prefix := "  "
			if m.selectedDeployment == -1 {
				prefix = "→ "
			}
			modalContent.WriteString(fmt.Sprintf("%s[Create New Deployment]\n", prefix))

			for i, deployment := range m.deployments {
				prefix = "  "
				if i == m.selectedDeployment {
					prefix = "→ "
				}
				modalContent.WriteString(fmt.Sprintf("%s%s (%s) - %s\n",
					prefix, deployment.PodName, deployment.Namespace, deployment.Status))
			}
			modalContent.WriteString("\n")
		}

		modalContent.WriteString("Use ↑/↓ to navigate, Enter/1 to select, 2 to cancel, ESC to close")

		return modalStyle.Render(modalContent.String())
	} else if m.modalStep == 1 {
		// New deployment creation step - use same logic as createNewDeployment
		deploymentName := strings.ToLower(m.selectedImage)
		// Replace invalid characters with hyphens
		deploymentName = strings.ReplaceAll(deploymentName, ":", "-")
		deploymentName = strings.ReplaceAll(deploymentName, "/", "-")
		deploymentName = strings.ReplaceAll(deploymentName, "_", "-")
		deploymentName = strings.ReplaceAll(deploymentName, ".", "-")

		// Remove any leading/trailing hyphens and ensure it's not empty
		deploymentName = strings.Trim(deploymentName, "-")
		if deploymentName == "" || deploymentName == "latest" {
			deploymentName = "new-deployment"
		}

		// Ensure it starts with a letter (Kubernetes requirement)
		if len(deploymentName) > 0 && (deploymentName[0] < 'a' || deploymentName[0] > 'z') {
			deploymentName = "app-" + deploymentName
		}

		modalContent := fmt.Sprintf(`Create New Deployment

Image: %s
Deployment Name: %s
Namespace: default
Port: 80
Replicas: 1

This will create a new Kubernetes deployment with:
- 1 replica pod running your image
- Container port 80 exposed
- ImagePullPolicy set to "Never" for local registry
- App label: %s

Options:
[1] Create Deployment
[2] Go Back

Press 1 to create, 2 to go back, or ESC to cancel`, m.selectedImage, deploymentName, deploymentName)

		return modalStyle.Render(modalContent)
	} else {
		// Confirmation step for existing deployment
		selectedDep := ""
		if len(m.deployments) > 0 && m.selectedDeployment < len(m.deployments) {
			selectedDep = m.deployments[m.selectedDeployment].PodName
		}

		modalContent := fmt.Sprintf(`Confirm Deployment

Image: %s
Deployment: %s

This will update the deployment's container image and trigger a rolling update.
All pods in this deployment will be updated with the new image.
Make sure the image is available in your registry!

Options:
[1] Confirm Deploy
[2] Go Back

Press 1 to confirm, 2 to go back, or ESC to cancel`, m.selectedImage, selectedDep)

		return modalStyle.Render(modalContent)
	}
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

// Message types for async operations
type deploymentsMsg struct {
	deployments []TableData
}

type deploymentPodsMsg struct {
	pods []TableData
}

type podDetailsMsg struct {
	details map[string]string
	err     error
}

func (m model) loadDeployments() tea.Cmd {
	return func() tea.Msg {
		deployments, _ := getKubernetesDeployments()
		return deploymentsMsg{deployments: deployments}
	}
}

func (m model) loadPodsForDeployment(deploymentName, namespace string) tea.Cmd {
	return func() tea.Msg {
		pods, _ := getPodsForDeployment(deploymentName, namespace)
		return deploymentPodsMsg{pods: pods}
	}
}

func (m model) loadPodDetails() tea.Cmd {
	return func() tea.Msg {
		details, err := getKubernetesPodDetails(m.selectedPod, m.selectedPodNS)
		return podDetailsMsg{
			details: details,
			err:     err,
		}
	}
}

func (m *model) initPodDefTable(details map[string]string) {
	columns := []table.Column{
		{Title: "Key", Width: 35},
		{Title: "Value", Width: 70},
	}

	var rows []table.Row

	if details != nil {
		// Add details in a logical order
		orderedKeys := []string{
			"Name", "Namespace", "Status", "Node", "Pod IP", "Host IP",
			"Created", "Start Time", "Service Account", "Restart Policy", "DNS Policy",
			"Container Name", "Container Image", "Image Pull Policy", "Container Ports",
			"CPU Request", "Memory Request", "CPU Limit", "Memory Limit",
			"Container Ready", "Restart Count", "Container ID",
			"Ready Condition", "Scheduled Condition", "Initialized Condition",
			"Labels", "Annotations",
		}

		for _, key := range orderedKeys {
			if value, exists := details[key]; exists && value != "" {
				rows = append(rows, table.Row{key, truncateString(value, 70)})
			}
		}

		// Add any remaining details not in the ordered list
		for key, value := range details {
			found := false
			for _, orderedKey := range orderedKeys {
				if key == orderedKey {
					found = true
					break
				}
			}
			if !found && value != "" {
				rows = append(rows, table.Row{key, truncateString(value, 70)})
			}
		}
	} else {
		// Error loading details
		rows = append(rows, table.Row{"Error", "Failed to load pod details"})
	}

	m.podDefTable = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
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

type dockerPullMsg struct {
	success  bool
	imageTag string
	err      error
}

type deploymentMsg struct {
	success bool
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

func (m model) pullDockerImage(imageTag string) tea.Cmd {
	return func() tea.Msg {
		// Execute docker pull command
		cmd := exec.Command("docker", "pull", imageTag)
		err := cmd.Run()

		return dockerPullMsg{
			success:  err == nil,
			imageTag: imageTag,
			err:      err,
		}
	}
}

func (m model) deployImageToPod(imageName, deploymentName, namespace string) tea.Cmd {
	return func() tea.Msg {
		err := deployImageToPod(imageName, deploymentName, namespace)
		return deploymentMsg{
			success: err == nil,
			err:     err,
		}
	}
}

func (m model) createNewDeployment(imageName string) tea.Cmd {
	return func() tea.Msg {
		// Generate a deployment name based on the image name
		deploymentName := strings.ToLower(imageName)
		// Replace invalid characters with hyphens
		deploymentName = strings.ReplaceAll(deploymentName, ":", "-")
		deploymentName = strings.ReplaceAll(deploymentName, "/", "-")
		deploymentName = strings.ReplaceAll(deploymentName, "_", "-")
		deploymentName = strings.ReplaceAll(deploymentName, ".", "-")

		// Remove any leading/trailing hyphens and ensure it's not empty
		deploymentName = strings.Trim(deploymentName, "-")
		if deploymentName == "" || deploymentName == "latest" {
			deploymentName = "new-deployment"
		}

		// Ensure it starts with a letter (Kubernetes requirement)
		if len(deploymentName) > 0 && (deploymentName[0] < 'a' || deploymentName[0] > 'z') {
			deploymentName = "app-" + deploymentName
		}

		err := createKubernetesDeployment(imageName, deploymentName, "default")
		return deploymentMsg{
			success: err == nil,
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
			if len(imageID) > 20 {
				imageID = imageID[:20]
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

func startTUI(gitData []TableData, dockerData []TableData, kubernetesData []TableData) {
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
		kubesData:  kubernetesData,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
