package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/apimachinery/pkg/util/duration"
)

// updateClusterList handles updates in cluster list view
func (m Model) updateClusterList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	// Don't handle keys if we're in input mode
	if m.isInputFocused() {
		return m, nil
	}
	
	switch msg.String() {
	case "r", "F5":
		m.loading = true
		return m, m.loadClusters()
	case "c":
		m.state = CreateClusterView
		m.createForm = NewCreateForm()
		return m, nil
	case "d", "enter":
		if cluster := m.getCurrentCluster(); cluster != nil {
			m.loading = true
			return m, m.loadClusterDetail(cluster.Namespace, cluster.Name)
		}
	case "e":
		if cluster := m.getCurrentCluster(); cluster != nil {
			m.state = EditClusterView
			m.editForm = NewEditForm(cluster)
			return m, nil
		}
	case "x", "delete":
		if cluster := m.getCurrentCluster(); cluster != nil {
			m.state = DeleteConfirmView
			m.deleteTarget = fmt.Sprintf("%s/%s", cluster.Namespace, cluster.Name)
			m.deleteInput = ""
			return m, nil
		}
	case "k":
		if cluster := m.getCurrentCluster(); cluster != nil {
			m.loading = true
			return m, m.loadKubeconfig(cluster.Namespace, cluster.Name)
		}
	case "y":
		if cluster := m.getCurrentCluster(); cluster != nil {
			m.loading = true
			return m, m.loadClusterDetail(cluster.Namespace, cluster.Name)
		}
	case "9":
		if cluster := m.getCurrentCluster(); cluster != nil {
			m.runningK9s = true
			return m, launchK9s(cluster.Namespace, cluster.Name)
		}
	case "n":
		// TODO: Implement namespace selector
		return m, nil
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// updateDetailView handles updates in cluster detail view
func (m Model) updateDetailView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.state = ClusterListView
		return m, nil
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// updateCreateView handles updates in create cluster view
func (m Model) updateCreateView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = ClusterListView
		return m, nil
	case "enter":
		if m.createForm.IsComplete() {
			cluster := m.createForm.ToCluster()
			m.loading = true
			return m, m.createCluster(cluster)
		} else {
			m.createForm.Next()
		}
		return m, nil
	case "tab":
		m.createForm.Next()
		return m, nil
	case "shift+tab":
		m.createForm.Previous()
		return m, nil
	}

	m.createForm.Update(msg)
	return m, nil
}

// updateEditView handles updates in edit cluster view
func (m Model) updateEditView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = ClusterListView
		return m, nil
	case "enter":
		if m.editForm.IsComplete() {
			cluster := m.editForm.ToCluster()
			m.loading = true
			return m, m.updateCluster(cluster)
		} else {
			m.editForm.Next()
		}
		return m, nil
	case "tab":
		m.editForm.Next()
		return m, nil
	case "shift+tab":
		m.editForm.Previous()
		return m, nil
	}

	m.editForm.Update(msg)
	return m, nil
}

// updateDeleteView handles updates in delete confirmation view
func (m Model) updateDeleteView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = ClusterListView
		return m, nil
	case "enter":
		targetName := strings.Split(m.deleteTarget, "/")[1]
		if m.deleteInput == targetName {
			parts := strings.Split(m.deleteTarget, "/")
			if len(parts) == 2 {
				m.loading = true
				return m, m.deleteCluster(parts[0], parts[1])
			}
		}
		return m, nil
	case "backspace":
		if len(m.deleteInput) > 0 {
			m.deleteInput = m.deleteInput[:len(m.deleteInput)-1]
		}
		return m, nil
	default:
		// Only add printable characters
		if len(msg.String()) == 1 && msg.String()[0] >= 32 && msg.String()[0] <= 126 {
			m.deleteInput += msg.String()
		}
		return m, nil
	}
}

// updateKubeconfigView handles updates in kubeconfig view
func (m Model) updateKubeconfigView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.state = ClusterListView
		return m, nil
	case "s":
		// TODO: Implement save to file
		return m, nil
	case "c":
		// TODO: Implement copy to clipboard
		return m, nil
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// updateFilterView handles updates in filter view (deprecated, using command mode now)
func (m Model) updateFilterView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.state = ClusterListView
		return m, nil
	case "enter":
		m.filter = m.textInput.Value()
		m.applyFilter()
		m.state = ClusterListView
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// updateCommandView handles updates in command view
func (m Model) updateCommandView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.state = m.lastState
		return m, nil
	case "enter":
		return m.executeCommand()
	}

	m.commandInput, cmd = m.commandInput.Update(msg)
	return m, cmd
}

// updateHelpView handles updates in help view
func (m Model) updateHelpView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = m.lastState
		return m, nil
	}
	return m, nil
}

// executeCommand executes the command entered in command mode
func (m Model) executeCommand() (Model, tea.Cmd) {
	command := strings.TrimSpace(m.commandInput.Value())
	
	switch m.commandMode {
	case "/":
		// Filter mode
		m.filter = command
		m.applyFilter()
		m.state = m.lastState
		return m, nil
		
	case ":":
		// Command mode
		if command == "" {
			m.state = m.lastState
			return m, nil
		}
		
		// Parse commands
		parts := strings.Fields(command)
		if len(parts) == 0 {
			m.state = m.lastState
			return m, nil
		}
		
		switch parts[0] {
		case "q", "quit":
			return m, tea.Quit
		case "r", "refresh":
			m.loading = true
			m.state = m.lastState
			return m, m.loadClusters()
		case "clear":
			m.error = ""
			m.filter = ""
			m.applyFilter()
			m.state = m.lastState
			return m, nil
		case "ns", "namespace":
			if len(parts) > 1 {
				m.namespace = parts[1]
				m.loading = true
				m.state = m.lastState
				return m, m.loadClusters()
			}
		case "help":
			m.lastState = m.state
			m.state = HelpView
			return m, nil
		default:
			m.error = fmt.Sprintf("Unknown command: %s", parts[0])
		}
	}
	
	m.state = m.lastState
	return m, nil
}

// viewClusterList renders the cluster list view
func (m Model) viewClusterList() string {
	if m.loading {
		return lipgloss.NewStyle().Foreground(catMauve).Margin(2, 0).
			Render(fmt.Sprintf("%s Loading clusters...", m.spinner.View()))
	}

	if len(m.filteredClusters) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(catOverlay1).Margin(2, 0).Align(lipgloss.Center)
		if len(m.clusters) == 0 {
			return emptyStyle.Render("No clusters found. Press 'c' to create a new cluster.")
		}
		return emptyStyle.Render(fmt.Sprintf("No clusters match filter '%s'. Press '/' to change filter.", m.filter))
	}

	return m.renderStyledTable()
}

// renderStyledTable applies k9s-style colors to the table
func (m Model) renderStyledTable() string {
	if len(m.filteredClusters) == 0 {
		return m.table.View()
	}

	// Get the base table view
	tableView := m.table.View()
	lines := strings.Split(tableView, "\n")
	
	if len(lines) < 2 {
		return tableView
	}

	// Style the header (first line)
	headerStyle := lipgloss.NewStyle().
		Foreground(colorTableHeader).
		Bold(true)
	lines[0] = headerStyle.Render(lines[0])

	// Style the data rows
	for i := 1; i < len(lines); i++ {
		if i-1 >= len(m.filteredClusters) {
			break
		}
		
		line := lines[i]
		
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		// Parse the line to apply colors to specific columns
		fields := strings.Fields(line)
		if len(fields) >= 7 {
			// Color the status field (index 6)
			status := fields[6]
			switch status {
			case "Running":
				fields[6] = lipgloss.NewStyle().Foreground(colorRunning).Render(status)
			case "Provisioning":
				fields[6] = lipgloss.NewStyle().Foreground(colorPending).Render(status)
			case "Failed":
				fields[6] = lipgloss.NewStyle().Foreground(colorFailed).Render(status)
			}
			
			// Color the mode field (index 2)
			mode := fields[2]
			switch mode {
			case "shared":
				fields[2] = lipgloss.NewStyle().Foreground(colorModeShared).Render(mode)
			case "virtual":
				fields[2] = lipgloss.NewStyle().Foreground(colorModeVirtual).Render(mode)
			}
			
			// Color the namespace field (index 1)
			fields[1] = lipgloss.NewStyle().Foreground(colorNamespace).Render(fields[1])
			
			// Color the age field (last field)
			if len(fields) > 7 {
				fields[len(fields)-1] = lipgloss.NewStyle().Foreground(colorAge).Render(fields[len(fields)-1])
			}
		}
		
		// Check if this is the selected row
		if i-1 == m.table.Cursor() {
			// Apply selected row style
			styledLine := strings.Join(fields, " ")
			lines[i] = lipgloss.NewStyle().
				Foreground(colorSelectedText).
				Background(colorSelectedBg).
				Width(m.width-2).
				Render(styledLine)
		} else {
			lines[i] = strings.Join(fields, " ")
		}
	}

	return strings.Join(lines, "\n")
}

// viewClusterDetail renders the cluster detail view (Catppuccin Mocha)
func (m Model) viewClusterDetail() string {
	if m.loading {
		return lipgloss.NewStyle().Foreground(catMauve).Margin(2, 0).
			Render(fmt.Sprintf("%s Loading cluster details...", m.spinner.View()))
	}

	return m.viewport.View()
}

// viewCreateCluster renders the create cluster view (centered modal)
func (m Model) viewCreateCluster() string {
	if m.loading {
		return lipgloss.NewStyle().Foreground(catMauve).Margin(2, 0).
			Render(fmt.Sprintf("%s Creating cluster...", m.spinner.View()))
	}

	if m.createForm == nil {
		return lipgloss.NewStyle().Foreground(catRed).Margin(2, 0).
			Render("Error: Create form not initialized")
	}

	m.createForm.width = m.width
	m.createForm.height = m.height - 8
	return m.createForm.View()
}

// viewEditCluster renders the edit cluster view (centered modal)
func (m Model) viewEditCluster() string {
	if m.loading {
		return lipgloss.NewStyle().Foreground(catMauve).Margin(2, 0).
			Render(fmt.Sprintf("%s Updating cluster...", m.spinner.View()))
	}

	if m.editForm == nil {
		return lipgloss.NewStyle().Foreground(catRed).Margin(2, 0).
			Render("Error: Edit form not initialized")
	}

	m.editForm.width = m.width
	m.editForm.height = m.height - 8
	return m.editForm.View()
}

// viewDeleteConfirm renders the delete confirmation modal (Catppuccin Mocha, centered)
func (m Model) viewDeleteConfirm() string {
	targetName := strings.Split(m.deleteTarget, "/")[1]

	titleStyle := lipgloss.NewStyle().
		Foreground(catRed).
		Bold(true).
		Align(lipgloss.Center).
		Width(54)

	clusterStyle := lipgloss.NewStyle().Foreground(catPeach).Bold(true)
	promptStyle := lipgloss.NewStyle().Foreground(catYellow).Bold(true)
	inputStyle := lipgloss.NewStyle().Foreground(catText)
	warningStyle := lipgloss.NewStyle().Foreground(catRed)

	content := fmt.Sprintf(`You are about to delete cluster:

%s

%s
All cluster resources will be permanently deleted.

Type the cluster name to confirm: %s

%s`,
		clusterStyle.Render(m.deleteTarget),
		warningStyle.Render("⚠  This action CANNOT be undone!"),
		promptStyle.Render(targetName),
		inputStyle.Render("► "+m.deleteInput+"█"))

	modalStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(catRed).
		Background(catMantle).
		Padding(1, 3).
		Width(60).
		Align(lipgloss.Center)

	modal := modalStyle.Render(
		titleStyle.Render("⚠  DELETE CLUSTER") + "\n\n" + content,
	)

	return lipgloss.Place(
		m.width, m.height-8,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// viewKubeconfig renders the kubeconfig view (Catppuccin Mocha)
func (m Model) viewKubeconfig() string {
	if m.loading {
		return lipgloss.NewStyle().Foreground(catMauve).Margin(2, 0).
			Render(fmt.Sprintf("%s Loading kubeconfig...", m.spinner.View()))
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(catPeach).
		Bold(true).
		Margin(0, 0, 1, 1)

	return headerStyle.Render("📋 Kubeconfig") + "\n" + m.viewport.View()
}

// viewFilter renders the filter view (Catppuccin Mocha, centered)
func (m Model) viewFilter() string {
	titleStyle := lipgloss.NewStyle().Foreground(catPeach).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(catOverlay1)
	valueStyle := lipgloss.NewStyle().Foreground(catYellow)

	content := fmt.Sprintf("%s\n\n%s %s\n%s %d of %d\n\n%s",
		titleStyle.Render("Filter Clusters"),
		labelStyle.Render("Current:"),
		valueStyle.Render(m.filter),
		labelStyle.Render("Showing:"),
		len(m.filteredClusters),
		len(m.clusters),
		m.textInput.View())

	modalStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(catBlue).
		Background(catMantle).
		Padding(1, 3).
		Width(50)

	modal := modalStyle.Render(content)

	return lipgloss.Place(
		m.width, m.height-8,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// viewHelp renders the help view (Catppuccin Mocha, centered)
func (m Model) viewHelp() string {
	sectionStyle := lipgloss.NewStyle().Foreground(catPeach).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(catMauve).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(catSubtext1)

	line := func(key, desc string) string {
		return fmt.Sprintf("  %s  %s", keyStyle.Width(16).Render(key), descStyle.Render(desc))
	}

	helpContent := strings.Join([]string{
		sectionStyle.Render("NAVIGATION"),
		line("↑/↓", "Navigate list"),
		line("enter, d", "View details/describe"),
		line("esc", "Go back / cancel"),
		line("q, ctrl+c", "Quit application"),
		"",
		sectionStyle.Render("CLUSTER OPERATIONS"),
		line("c", "Create new cluster"),
		line("e", "Edit cluster"),
		line("x, delete", "Delete cluster"),
		line("k", "View kubeconfig"),
		line("y", "View YAML"),
		"",
		sectionStyle.Render("FILTERING & SEARCH"),
		line("/", "Filter clusters"),
		line(":", "Command mode"),
		line("?", "Show this help"),
		"",
		sectionStyle.Render("COMMANDS"),
		line(":q, :quit", "Quit application"),
		line(":r, :refresh", "Refresh data"),
		line(":ns <name>", "Switch namespace"),
		line(":clear", "Clear filter and errors"),
		"",
		sectionStyle.Render("FORMS"),
		line("tab", "Next field"),
		line("shift+tab", "Previous field"),
		line("enter", "Submit / next step"),
		line("space", "Toggle options"),
	}, "\n")

	titleStyle := lipgloss.NewStyle().
		Foreground(catPeach).
		Bold(true).
		Align(lipgloss.Center)

	modalWidth := 56
	if m.width > 70 {
		modalWidth = 60
	}

	modalStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(catLavender).
		Background(catMantle).
		Padding(1, 3).
		Width(modalWidth)

	title := titleStyle.Width(modalWidth - 8).Render("⌨  KEYBOARD SHORTCUTS")
	modal := modalStyle.Render(title + "\n\n" + helpContent)

	return lipgloss.Place(
		m.width, m.height-4,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// formatClusterDetails formats cluster details in k9s YAML style (Catppuccin Mocha)
func (m Model) formatClusterDetails() string {
	if m.selectedCluster == nil {
		return "No cluster selected"
	}

	cluster := m.selectedCluster
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().Foreground(catPeach).Bold(true)
	separatorStyle := lipgloss.NewStyle().Foreground(catSurface2)

	content.WriteString(titleStyle.Render(fmt.Sprintf("Cluster: %s/%s", cluster.Namespace, cluster.Name)) + "\n")
	content.WriteString(separatorStyle.Render(strings.Repeat("─", 50)) + "\n\n")

	// Key-value styling
	keyStyle := lipgloss.NewStyle().Foreground(colorYamlKey)
	valueStyle := lipgloss.NewStyle().Foreground(colorYamlValue)
	statusStyle := lipgloss.NewStyle().Foreground(colorYamlStatus)
	sectionStyle := lipgloss.NewStyle().Foreground(colorYamlHeader).Bold(true)

	// Metadata section
	content.WriteString(sectionStyle.Render("Metadata:") + "\n")
	content.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render("name"), valueStyle.Render(cluster.Name)))
	content.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render("namespace"), valueStyle.Render(cluster.Namespace)))
	content.WriteString(fmt.Sprintf("  %s: %s (%s ago)\n", 
		keyStyle.Render("created"), 
		valueStyle.Render(cluster.CreationTimestamp.Format("2006-01-02 15:04:05")),
		valueStyle.Render(duration.HumanDuration(time.Since(cluster.CreationTimestamp.Time)))))
	
	if cluster.DeletionTimestamp != nil {
		content.WriteString(fmt.Sprintf("  %s: %s\n", 
			keyStyle.Render("deleting"), 
			valueStyle.Render(cluster.DeletionTimestamp.Format("2006-01-02 15:04:05"))))
	}

	// Labels
	if len(cluster.Labels) > 0 {
		content.WriteString(fmt.Sprintf("  %s:\n", keyStyle.Render("labels")))
		for k, v := range cluster.Labels {
			content.WriteString(fmt.Sprintf("    %s: %s\n", keyStyle.Render(k), valueStyle.Render(v)))
		}
	}

	content.WriteString("\n")

	// Spec section
	content.WriteString(sectionStyle.Render("Specification:") + "\n")
	content.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render("mode"), valueStyle.Render(cluster.Spec.Mode)))
	if cluster.Spec.Version != "" {
		content.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render("version"), valueStyle.Render(cluster.Spec.Version)))
	}
	if cluster.Spec.Servers != nil {
		content.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render("servers"), valueStyle.Render(fmt.Sprintf("%d", *cluster.Spec.Servers))))
	}
	if cluster.Spec.Agents != nil {
		content.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render("agents"), valueStyle.Render(fmt.Sprintf("%d", *cluster.Spec.Agents))))
	}
	if cluster.Spec.ClusterCIDR != "" {
		content.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render("clusterCIDR"), valueStyle.Render(cluster.Spec.ClusterCIDR)))
	}
	if cluster.Spec.ServiceCIDR != "" {
		content.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render("serviceCIDR"), valueStyle.Render(cluster.Spec.ServiceCIDR)))
	}
	if cluster.Spec.ClusterDNS != "" {
		content.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render("clusterDNS"), valueStyle.Render(cluster.Spec.ClusterDNS)))
	}

	// Persistence
	if cluster.Spec.Persistence != nil {
		content.WriteString(fmt.Sprintf("  %s:\n", keyStyle.Render("persistence")))
		content.WriteString(fmt.Sprintf("    %s: %s\n", keyStyle.Render("type"), valueStyle.Render(cluster.Spec.Persistence.Type)))
		if cluster.Spec.Persistence.StorageClassName != "" {
			content.WriteString(fmt.Sprintf("    %s: %s\n", keyStyle.Render("storageClass"), valueStyle.Render(cluster.Spec.Persistence.StorageClassName)))
		}
		if cluster.Spec.Persistence.StorageRequestSize != "" {
			content.WriteString(fmt.Sprintf("    %s: %s\n", keyStyle.Render("storageSize"), valueStyle.Render(cluster.Spec.Persistence.StorageRequestSize)))
		}
	}

	content.WriteString("\n")

	// Status section
	content.WriteString(sectionStyle.Render("Status:") + "\n")
	
	// Color the phase based on status
	var phaseStyle lipgloss.Style
	switch cluster.Status.Phase {
	case "Running":
		phaseStyle = statusStyle.Foreground(colorRunning)
	case "Provisioning":
		phaseStyle = statusStyle.Foreground(colorPending)
	case "Failed":
		phaseStyle = statusStyle.Foreground(colorFailed)
	default:
		phaseStyle = statusStyle
	}
	
	content.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render("phase"), phaseStyle.Render(cluster.Status.Phase)))
	
	if cluster.Status.HostVersion != "" {
		content.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render("hostVersion"), valueStyle.Render(cluster.Status.HostVersion)))
	}

	// Conditions
	if len(cluster.Status.Conditions) > 0 {
		content.WriteString(fmt.Sprintf("  %s:\n", keyStyle.Render("conditions")))
		for _, condition := range cluster.Status.Conditions {
			conditionStatusText := string(condition.Status)
			var styledStatus string
			if conditionStatusText == "True" {
				styledStatus = statusStyle.Foreground(colorRunning).Render(conditionStatusText)
			} else {
				styledStatus = statusStyle.Foreground(colorFailed).Render(conditionStatusText)
			}
			
			content.WriteString(fmt.Sprintf("    - %s: %s (%s)\n", 
				keyStyle.Render(string(condition.Type)), 
				styledStatus, 
				valueStyle.Render(condition.Reason)))
			if condition.Message != "" {
				content.WriteString(fmt.Sprintf("      %s\n", valueStyle.Render(condition.Message)))
			}
		}
	}

	return content.String()
}