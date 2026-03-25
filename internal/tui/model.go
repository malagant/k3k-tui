package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/malagant/k3k-tui/internal/k8s"
	"github.com/malagant/k3k-tui/internal/types"
)

// ViewState represents the current view
type ViewState int

const (
	ClusterListView ViewState = iota
	ClusterDetailView
	CreateClusterView
	EditClusterView
	DeleteConfirmView
	KubeconfigView
	FilterView
)

// Model represents the main TUI model
type Model struct {
	client   *k8s.Client
	version  string
	state    ViewState
	
	// UI components
	table        table.Model
	spinner      spinner.Model
	viewport     viewport.Model
	textInput    textinput.Model
	
	// Data
	clusters     []types.Cluster
	selectedCluster *types.Cluster
	filteredClusters []types.Cluster
	
	// State
	loading      bool
	error        string
	filter       string
	namespace    string
	
	// Create/edit form
	createForm   *CreateForm
	editForm     *EditForm
	
	// Delete confirmation
	deleteTarget string
	deleteInput  string
	
	// Kubeconfig
	kubeconfigContent string
	
	// Dimensions
	width        int
	height       int
}

// NewModel creates a new TUI model
func NewModel(client *k8s.Client, version string) Model {
	// Initialize table
	columns := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Namespace", Width: 15},
		{Title: "Mode", Width: 10},
		{Title: "Version", Width: 15},
		{Title: "Servers", Width: 8},
		{Title: "Agents", Width: 7},
		{Title: "Status", Width: 12},
		{Title: "Age", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
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
	t.SetStyles(s)

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize viewport for detail view
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)

	// Initialize text input for filtering
	ti := textinput.New()
	ti.Placeholder = "Filter clusters..."
	ti.Focus()

	return Model{
		client:    client,
		version:   version,
		state:     ClusterListView,
		table:     t,
		spinner:   sp,
		viewport:  vp,
		textInput: ti,
		loading:   true,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadClusters(),
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateComponentSizes()

	case tea.KeyMsg:
		switch m.state {
		case ClusterListView:
			return m.updateClusterList(msg)
		case ClusterDetailView:
			return m.updateDetailView(msg)
		case CreateClusterView:
			return m.updateCreateView(msg)
		case EditClusterView:
			return m.updateEditView(msg)
		case DeleteConfirmView:
			return m.updateDeleteView(msg)
		case KubeconfigView:
			return m.updateKubeconfigView(msg)
		case FilterView:
			return m.updateFilterView(msg)
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case clustersLoadedMsg:
		m.loading = false
		m.clusters = msg.clusters
		m.filteredClusters = msg.clusters
		m.updateTable()

	case clusterCreatedMsg:
		m.loading = false
		if msg.err != nil {
			m.error = fmt.Sprintf("Failed to create cluster: %v", msg.err)
		} else {
			m.state = ClusterListView
			cmds = append(cmds, m.loadClusters())
		}

	case clusterUpdatedMsg:
		m.loading = false
		if msg.err != nil {
			m.error = fmt.Sprintf("Failed to update cluster: %v", msg.err)
		} else {
			m.state = ClusterListView
			cmds = append(cmds, m.loadClusters())
		}

	case clusterDeletedMsg:
		m.loading = false
		if msg.err != nil {
			m.error = fmt.Sprintf("Failed to delete cluster: %v", msg.err)
		} else {
			m.state = ClusterListView
			cmds = append(cmds, m.loadClusters())
		}

	case clusterDetailLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.error = fmt.Sprintf("Failed to load cluster details: %v", msg.err)
		} else {
			m.selectedCluster = msg.cluster
			m.viewport.SetContent(m.formatClusterDetails())
			m.state = ClusterDetailView
		}

	case kubeconfigLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.error = fmt.Sprintf("Failed to load kubeconfig: %v", msg.err)
		} else {
			m.kubeconfigContent = msg.content
			m.viewport.SetContent(m.kubeconfigContent)
			m.state = KubeconfigView
		}

	case errorMsg:
		m.loading = false
		m.error = msg.error
	}

	return m, tea.Batch(cmds...)
}

// updateComponentSizes updates component sizes based on window dimensions
func (m *Model) updateComponentSizes() {
	headerHeight := 3
	footerHeight := 3
	availableHeight := m.height - headerHeight - footerHeight

	m.table.SetHeight(availableHeight - 4)
	m.viewport.Width = m.width - 4
	m.viewport.Height = availableHeight - 4
}

// View renders the current view
func (m Model) View() string {
	var content string

	switch m.state {
	case ClusterListView:
		content = m.viewClusterList()
	case ClusterDetailView:
		content = m.viewClusterDetail()
	case CreateClusterView:
		content = m.viewCreateCluster()
	case EditClusterView:
		content = m.viewEditCluster()
	case DeleteConfirmView:
		content = m.viewDeleteConfirm()
	case KubeconfigView:
		content = m.viewKubeconfig()
	case FilterView:
		content = m.viewFilter()
	default:
		content = "Unknown view"
	}

	// Header
	header := m.renderHeader()
	
	// Footer
	footer := m.renderFooter()

	// Error display
	errorDisplay := ""
	if m.error != "" {
		errorDisplay = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("Error: %s", m.error)) + "\n"
	}

	return fmt.Sprintf("%s\n%s%s\n%s", header, errorDisplay, content, footer)
}

// renderHeader renders the header
func (m Model) renderHeader() string {
	title := fmt.Sprintf("k3k TUI - Virtual Cluster Manager %s", m.version)
	if m.namespace != "" {
		title += fmt.Sprintf(" (namespace: %s)", m.namespace)
	}
	
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		BorderStyle(lipgloss.DoubleBorder()).
		BorderBottom(true).
		Width(m.width).
		Align(lipgloss.Center).
		Render(title)
}

// renderFooter renders the footer with keybindings
func (m Model) renderFooter() string {
	var help string
	
	switch m.state {
	case ClusterListView:
		help = "↑/↓: navigate • c: create • d/enter: details • e: edit • x: delete • k: kubeconfig • /: filter • n: namespace • q: quit"
	case ClusterDetailView, KubeconfigView:
		help = "↑/↓: scroll • esc: back • q: quit"
	case CreateClusterView, EditClusterView:
		help = "tab/shift+tab: navigate • enter: next/submit • esc: cancel • q: quit"
	case DeleteConfirmView:
		help = "type cluster name to confirm • enter: delete • esc: cancel • q: quit"
	case FilterView:
		help = "type to filter • enter: apply • esc: cancel • q: quit"
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		Width(m.width).
		Align(lipgloss.Center).
		Render(help)
}

// updateTable updates the cluster table
func (m *Model) updateTable() {
	rows := make([]table.Row, len(m.filteredClusters))
	
	for i, cluster := range m.filteredClusters {
		status := cluster.Status.Phase
		if status == "" {
			status = "Unknown"
		}
		
		// Color-code status
		switch status {
		case types.PhaseRunning:
			status = lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Render(status)
		case types.PhaseProvisioning:
			status = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(status)
		case types.PhaseFailed:
			status = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(status)
		}

		servers := "0"
		if cluster.Spec.Servers != nil {
			servers = fmt.Sprintf("%d", *cluster.Spec.Servers)
		}

		agents := "0"
		if cluster.Spec.Agents != nil {
			agents = fmt.Sprintf("%d", *cluster.Spec.Agents)
		}

		age := k8s.Age(cluster.ObjectMeta.CreationTimestamp)

		rows[i] = table.Row{
			cluster.Name,
			cluster.Namespace,
			cluster.Spec.Mode,
			cluster.Spec.Version,
			servers,
			agents,
			status,
			age,
		}
	}

	m.table.SetRows(rows)
}

// getCurrentCluster returns the currently selected cluster
func (m Model) getCurrentCluster() *types.Cluster {
	if len(m.filteredClusters) == 0 {
		return nil
	}
	
	selected := m.table.Cursor()
	if selected >= 0 && selected < len(m.filteredClusters) {
		return &m.filteredClusters[selected]
	}
	
	return nil
}

// applyFilter applies the current filter to clusters
func (m *Model) applyFilter() {
	if m.filter == "" {
		m.filteredClusters = m.clusters
	} else {
		m.filteredClusters = []types.Cluster{}
		filter := strings.ToLower(m.filter)
		
		for _, cluster := range m.clusters {
			if strings.Contains(strings.ToLower(cluster.Name), filter) ||
				strings.Contains(strings.ToLower(cluster.Namespace), filter) ||
				strings.Contains(strings.ToLower(cluster.Spec.Mode), filter) ||
				strings.Contains(strings.ToLower(cluster.Status.Phase), filter) {
				m.filteredClusters = append(m.filteredClusters, cluster)
			}
		}
	}
	
	m.updateTable()
}