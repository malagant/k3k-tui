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
	
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
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
	case "/":
		m.state = FilterView
		m.textInput.SetValue(m.filter)
		m.textInput.Focus()
		return m, nil
	case "n":
		// TODO: Implement namespace selector
		return m, nil
	case "esc":
		m.error = ""
		return m, nil
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// updateDetailView handles updates in cluster detail view
func (m Model) updateDetailView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
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
	case "q", "ctrl+c":
		return m, tea.Quit
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
	case "q", "ctrl+c":
		return m, tea.Quit
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
	case "q", "ctrl+c":
		return m, tea.Quit
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
		m.deleteInput += msg.String()
		return m, nil
	}
}

// updateKubeconfigView handles updates in kubeconfig view
func (m Model) updateKubeconfigView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
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

// updateFilterView handles updates in filter view
func (m Model) updateFilterView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
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

// viewClusterList renders the cluster list view
func (m Model) viewClusterList() string {
	if m.loading {
		return fmt.Sprintf("\n%s Loading clusters...\n", m.spinner.View())
	}

	if len(m.filteredClusters) == 0 {
		if len(m.clusters) == 0 {
			return "\nNo clusters found. Press 'c' to create a new cluster.\n"
		} else {
			return fmt.Sprintf("\nNo clusters match filter '%s'. Press '/' to change filter or 'esc' to clear.\n", m.filter)
		}
	}

	content := m.table.View()
	
	if m.filter != "" {
		filterInfo := fmt.Sprintf("\nFilter: %s (showing %d of %d clusters)", 
			m.filter, len(m.filteredClusters), len(m.clusters))
		content += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(filterInfo)
	}

	return content
}

// viewClusterDetail renders the cluster detail view
func (m Model) viewClusterDetail() string {
	if m.loading {
		return fmt.Sprintf("\n%s Loading cluster details...\n", m.spinner.View())
	}

	return m.viewport.View()
}

// viewCreateCluster renders the create cluster view
func (m Model) viewCreateCluster() string {
	if m.loading {
		return fmt.Sprintf("\n%s Creating cluster...\n", m.spinner.View())
	}

	if m.createForm == nil {
		return "Initializing create form..."
	}

	return m.createForm.View()
}

// viewEditCluster renders the edit cluster view
func (m Model) viewEditCluster() string {
	if m.loading {
		return fmt.Sprintf("\n%s Updating cluster...\n", m.spinner.View())
	}

	if m.editForm == nil {
		return "Initializing edit form..."
	}

	return m.editForm.View()
}

// viewDeleteConfirm renders the delete confirmation view
func (m Model) viewDeleteConfirm() string {
	targetName := strings.Split(m.deleteTarget, "/")[1]
	
	content := fmt.Sprintf(`
⚠️  Delete Cluster Confirmation

You are about to delete cluster: %s

This action cannot be undone. The cluster and all its resources will be permanently deleted.

Type the cluster name to confirm: %s

Input: %s`, m.deleteTarget, targetName, m.deleteInput)

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Width(60)

	return style.Render(content)
}

// viewKubeconfig renders the kubeconfig view
func (m Model) viewKubeconfig() string {
	if m.loading {
		return fmt.Sprintf("\n%s Loading kubeconfig...\n", m.spinner.View())
	}

	header := "Kubeconfig for cluster\n" + strings.Repeat("=", 50) + "\n\n"
	return header + m.viewport.View()
}

// viewFilter renders the filter view
func (m Model) viewFilter() string {
	content := fmt.Sprintf(`
Filter Clusters

Current filter: %s
Clusters shown: %d of %d

%s`, m.filter, len(m.filteredClusters), len(m.clusters), m.textInput.View())

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(50)

	return style.Render(content)
}

// formatClusterDetails formats cluster details for display
func (m Model) formatClusterDetails() string {
	if m.selectedCluster == nil {
		return "No cluster selected"
	}

	cluster := m.selectedCluster
	var content strings.Builder

	content.WriteString(fmt.Sprintf("Cluster: %s/%s\n", cluster.Namespace, cluster.Name))
	content.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Metadata
	content.WriteString("Metadata:\n")
	content.WriteString(fmt.Sprintf("  Name: %s\n", cluster.Name))
	content.WriteString(fmt.Sprintf("  Namespace: %s\n", cluster.Namespace))
	content.WriteString(fmt.Sprintf("  Created: %s (%s ago)\n", 
		cluster.CreationTimestamp.Format("2006-01-02 15:04:05"),
		duration.HumanDuration(time.Since(cluster.CreationTimestamp.Time))))
	
	if cluster.DeletionTimestamp != nil {
		content.WriteString(fmt.Sprintf("  Deleting: %s\n", 
			cluster.DeletionTimestamp.Format("2006-01-02 15:04:05")))
	}

	// Labels
	if len(cluster.Labels) > 0 {
		content.WriteString("  Labels:\n")
		for k, v := range cluster.Labels {
			content.WriteString(fmt.Sprintf("    %s: %s\n", k, v))
		}
	}

	// Annotations
	if len(cluster.Annotations) > 0 {
		content.WriteString("  Annotations:\n")
		for k, v := range cluster.Annotations {
			content.WriteString(fmt.Sprintf("    %s: %s\n", k, v))
		}
	}

	content.WriteString("\n")

	// Spec
	content.WriteString("Specification:\n")
	content.WriteString(fmt.Sprintf("  Mode: %s\n", cluster.Spec.Mode))
	if cluster.Spec.Version != "" {
		content.WriteString(fmt.Sprintf("  Version: %s\n", cluster.Spec.Version))
	}
	if cluster.Spec.Servers != nil {
		content.WriteString(fmt.Sprintf("  Servers: %d\n", *cluster.Spec.Servers))
	}
	if cluster.Spec.Agents != nil {
		content.WriteString(fmt.Sprintf("  Agents: %d\n", *cluster.Spec.Agents))
	}
	if cluster.Spec.ClusterCIDR != "" {
		content.WriteString(fmt.Sprintf("  Cluster CIDR: %s\n", cluster.Spec.ClusterCIDR))
	}
	if cluster.Spec.ServiceCIDR != "" {
		content.WriteString(fmt.Sprintf("  Service CIDR: %s\n", cluster.Spec.ServiceCIDR))
	}
	if cluster.Spec.ClusterDNS != "" {
		content.WriteString(fmt.Sprintf("  Cluster DNS: %s\n", cluster.Spec.ClusterDNS))
	}

	// Persistence
	if cluster.Spec.Persistence != nil {
		content.WriteString("  Persistence:\n")
		content.WriteString(fmt.Sprintf("    Type: %s\n", cluster.Spec.Persistence.Type))
		if cluster.Spec.Persistence.StorageClassName != "" {
			content.WriteString(fmt.Sprintf("    Storage Class: %s\n", cluster.Spec.Persistence.StorageClassName))
		}
		if cluster.Spec.Persistence.StorageRequestSize != "" {
			content.WriteString(fmt.Sprintf("    Storage Size: %s\n", cluster.Spec.Persistence.StorageRequestSize))
		}
	}

	// Node Selector
	if len(cluster.Spec.NodeSelector) > 0 {
		content.WriteString("  Node Selector:\n")
		for k, v := range cluster.Spec.NodeSelector {
			content.WriteString(fmt.Sprintf("    %s: %s\n", k, v))
		}
	}

	// TLS SANs
	if len(cluster.Spec.TLSSANs) > 0 {
		content.WriteString("  TLS SANs:\n")
		for _, san := range cluster.Spec.TLSSANs {
			content.WriteString(fmt.Sprintf("    - %s\n", san))
		}
	}

	// Server Args
	if len(cluster.Spec.ServerArgs) > 0 {
		content.WriteString("  Server Args:\n")
		for _, arg := range cluster.Spec.ServerArgs {
			content.WriteString(fmt.Sprintf("    - %s\n", arg))
		}
	}

	// Agent Args
	if len(cluster.Spec.AgentArgs) > 0 {
		content.WriteString("  Agent Args:\n")
		for _, arg := range cluster.Spec.AgentArgs {
			content.WriteString(fmt.Sprintf("    - %s\n", arg))
		}
	}

	content.WriteString("\n")

	// Status
	content.WriteString("Status:\n")
	content.WriteString(fmt.Sprintf("  Phase: %s\n", cluster.Status.Phase))
	if cluster.Status.HostVersion != "" {
		content.WriteString(fmt.Sprintf("  Host Version: %s\n", cluster.Status.HostVersion))
	}
	if cluster.Status.ClusterCIDR != "" {
		content.WriteString(fmt.Sprintf("  Actual Cluster CIDR: %s\n", cluster.Status.ClusterCIDR))
	}
	if cluster.Status.ServiceCIDR != "" {
		content.WriteString(fmt.Sprintf("  Actual Service CIDR: %s\n", cluster.Status.ServiceCIDR))
	}
	if cluster.Status.ClusterDNS != "" {
		content.WriteString(fmt.Sprintf("  Actual Cluster DNS: %s\n", cluster.Status.ClusterDNS))
	}
	if cluster.Status.PolicyName != "" {
		content.WriteString(fmt.Sprintf("  Policy Name: %s\n", cluster.Status.PolicyName))
	}

	// Conditions
	if len(cluster.Status.Conditions) > 0 {
		content.WriteString("  Conditions:\n")
		for _, condition := range cluster.Status.Conditions {
			content.WriteString(fmt.Sprintf("    %s: %s (%s)\n", 
				condition.Type, condition.Status, condition.Reason))
			if condition.Message != "" {
				content.WriteString(fmt.Sprintf("      %s\n", condition.Message))
			}
		}
	}

	return content.String()
}