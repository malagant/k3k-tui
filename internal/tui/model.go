package tui

import (
	"fmt"
	"strings"
	"time"

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
	CommandView
	HelpView
)

// k9s-like color scheme
var (
	// Background colors
	colorBg          = lipgloss.Color("#000000")        // Terminal default
	colorBreadcrumbBg = lipgloss.Color("#008B8B")      // Dark cyan for breadcrumb
	
	// Text colors
	colorHeaderText   = lipgloss.Color("#FFFFFF")       // Bold white for headers
	colorTableHeader  = lipgloss.Color("#00FFFF")       // Bright cyan/teal for table headers
	colorSelectedBg   = lipgloss.Color("#008080")       // Teal background for selected row
	colorSelectedText = lipgloss.Color("#FFFFFF")       // White text on selected row
	colorRunning      = lipgloss.Color("#00FF00")       // Green for running/ready
	colorPending      = lipgloss.Color("#FFA500")       // Orange for pending
	colorFailed       = lipgloss.Color("#FF0000")       // Red for failed/error
	colorAge          = lipgloss.Color("#808080")       // Gray for age
	colorNamespace    = lipgloss.Color("#87CEEB")       // Light blue for namespace
	colorModeShared   = lipgloss.Color("#00FFFF")       // Cyan for shared mode
	colorModeVirtual  = lipgloss.Color("#FF00FF")       // Magenta for virtual mode
	colorHelp         = lipgloss.Color("#696969")       // Dark gray for help
	colorCommand      = lipgloss.Color("#FFFF00")       // Yellow for command bar
	
	// YAML colors
	colorYamlKey      = lipgloss.Color("#00FFFF")       // Cyan for YAML keys
	colorYamlValue    = lipgloss.Color("#FFFFFF")       // White for YAML values
	colorYamlStatus   = lipgloss.Color("#00FF00")       // Green for status fields
	colorYamlHeader   = lipgloss.Color("#FFFF00")       // Bold yellow for section headers
)

// Model represents the main TUI model
type Model struct {
	client   *k8s.Client
	version  string
	state    ViewState
	lastState ViewState // For returning from help/command views
	
	// UI components
	table        table.Model
	spinner      spinner.Model
	viewport     viewport.Model
	textInput    textinput.Model
	commandInput textinput.Model
	
	// Data
	clusters     []types.Cluster
	selectedCluster *types.Cluster
	filteredClusters []types.Cluster
	
	// State
	loading      bool
	error        string
	filter       string
	namespace    string
	commandMode  string // ":", "/", "?"
	
	// Create/edit form
	createForm   *CreateForm
	editForm     *EditForm
	
	// Delete confirmation
	deleteTarget string
	deleteInput  string
	
	// Kubeconfig
	kubeconfigContent string
	
	// k9s context info
	contextName      string
	clusterName      string
	k8sVersion       string
	refreshInterval  time.Duration
	lastRefresh      time.Time
	
	// Dimensions
	width        int
	height       int
}

// ASCII art logo for k3k
const k3kLogo = ` _    ___  _    
| | _|_  )| | __
| |/ / / / | |/ /
|   < / /_ |   < 
|_|\_\____||_|\_\`

// NewModel creates a new TUI model
func NewModel(client *k8s.Client, version string) Model {
	// Initialize borderless table with k9s-like styling
	columns := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "NAMESPACE", Width: 15},
		{Title: "MODE", Width: 8},
		{Title: "VERSION", Width: 15},
		{Title: "S", Width: 3},  // Servers
		{Title: "A", Width: 3},  // Agents
		{Title: "STATUS", Width: 10},
		{Title: "AGE", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	// k9s-style table (borderless)
	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		Foreground(colorTableHeader).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(false).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		Align(lipgloss.Left)
	
	s.Selected = s.Selected.
		Foreground(colorSelectedText).
		Background(colorSelectedBg).
		Bold(false)
	
	s.Cell = s.Cell.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(false).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false)
	
	t.SetStyles(s)

	// Initialize spinner for loading states
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorCommand)

	// Initialize viewport for detail views
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		Margin(1, 2)

	// Initialize text input for filtering
	ti := textinput.New()
	ti.Placeholder = "Filter..."
	ti.Width = 30

	// Initialize command input
	ci := textinput.New()
	ci.Width = 50

	return Model{
		client:          client,
		version:         version,
		state:           ClusterListView,
		table:           t,
		spinner:         sp,
		viewport:        vp,
		textInput:       ti,
		commandInput:    ci,
		loading:         true,
		refreshInterval: 5 * time.Second,
		contextName:     "default", // TODO: get from kubeconfig
		clusterName:     "unknown", // TODO: get from kubeconfig  
		k8sVersion:      "unknown", // TODO: get from cluster
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
		// Global commands (work in any view except when typing)
		if !m.isInputFocused() {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case ":":
				return m.enterCommandMode(":")
			case "/":
				return m.enterCommandMode("/")
			case "?":
				m.lastState = m.state
				m.state = HelpView
				return m, nil
			case "esc":
				if m.state == HelpView || m.state == CommandView || m.state == FilterView {
					m.state = m.lastState
					m.error = ""
					return m, nil
				}
				m.error = ""
				return m, nil
			}
		}

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
		case CommandView:
			return m.updateCommandView(msg)
		case HelpView:
			return m.updateHelpView(msg)
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case clustersLoadedMsg:
		m.loading = false
		m.lastRefresh = time.Now()
		if msg.err != nil {
			m.error = fmt.Sprintf("Failed to load clusters: %v", msg.err)
		} else {
			m.clusters = msg.clusters
			m.filteredClusters = msg.clusters
			m.updateTable()
		}

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

// isInputFocused returns true if any text input is currently focused
func (m Model) isInputFocused() bool {
	switch m.state {
	case FilterView:
		return m.textInput.Focused()
	case CommandView:
		return m.commandInput.Focused()
	case CreateClusterView:
		return m.createForm != nil
	case EditClusterView:
		return m.editForm != nil
	case DeleteConfirmView:
		return true
	}
	return false
}

// enterCommandMode switches to command mode
func (m Model) enterCommandMode(mode string) (Model, tea.Cmd) {
	m.lastState = m.state
	m.state = CommandView
	m.commandMode = mode
	m.commandInput.SetValue("")
	
	switch mode {
	case ":":
		m.commandInput.Placeholder = "Enter command..."
	case "/":
		m.commandInput.Placeholder = "Enter filter..."
		m.commandInput.SetValue(m.filter)
	}
	
	m.commandInput.Focus()
	return m, nil
}

// updateComponentSizes updates component sizes based on window dimensions
func (m *Model) updateComponentSizes() {
	headerHeight := 7  // Logo area + breadcrumb + command bar
	footerHeight := 3  // Status bar + help
	availableHeight := m.height - headerHeight - footerHeight

	if availableHeight < 5 {
		availableHeight = 5
	}

	m.table.SetHeight(availableHeight)
	m.viewport.Width = m.width - 4
	m.viewport.Height = availableHeight
}

// View renders the current view
func (m Model) View() string {
	// Command bar (top line) - only show when active
	commandBar := ""
	if m.state == CommandView {
		commandBar = m.renderCommandBar()
	}

	// Header with logo and info
	header := m.renderHeader()
	
	// Breadcrumb bar
	breadcrumb := m.renderBreadcrumb()

	// Main content
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
	case CommandView:
		content = m.viewClusterList() // Show list in background
	case HelpView:
		content = m.viewHelp()
	default:
		content = "Unknown view"
	}

	// Footer with keybindings and status
	footer := m.renderFooter()

	// Error display
	errorDisplay := ""
	if m.error != "" {
		errorDisplay = "\n" + lipgloss.NewStyle().
			Foreground(colorFailed).
			Bold(true).
			Render(fmt.Sprintf("● %s", m.error))
	}

	// Build the layout
	var sections []string
	
	if commandBar != "" {
		sections = append(sections, commandBar)
	}
	
	sections = append(sections, 
		header, 
		breadcrumb,
	)
	
	if errorDisplay != "" {
		sections = append(sections, errorDisplay)
	}
	
	sections = append(sections, 
		content, 
		footer,
	)
	
	return strings.Join(sections, "\n")
}

// renderCommandBar renders the vim-style command bar
func (m Model) renderCommandBar() string {
	if m.state != CommandView {
		return ""
	}
	
	prompt := m.commandMode
	input := m.commandInput.View()
	
	commandStyle := lipgloss.NewStyle().
		Foreground(colorCommand).
		Background(colorBg)
	
	return commandStyle.Render(prompt + input)
}

// renderHeader renders the k9s-style header with logo
func (m Model) renderHeader() string {
	leftInfo := fmt.Sprintf("Context: %s\nCluster: %s\nK8s: %s\nk3k-tui: %s", 
		m.contextName, 
		m.clusterName, 
		m.k8sVersion, 
		m.version)

	leftStyle := lipgloss.NewStyle().
		Foreground(colorHeaderText).
		Width(30).
		Height(4).
		Align(lipgloss.Left, lipgloss.Top)

	rightStyle := lipgloss.NewStyle().
		Foreground(colorHeaderText).
		Width(20).
		Height(4).
		Align(lipgloss.Right, lipgloss.Top)

	headerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(4)

	left := leftStyle.Render(leftInfo)
	right := rightStyle.Render(k3kLogo)
	
	return headerStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			left,
			lipgloss.NewStyle().Width(m.width-50).Render(""),
			right,
		),
	)
}

// renderBreadcrumb renders the k9s-style breadcrumb bar
func (m Model) renderBreadcrumb() string {
	var text string
	count := len(m.filteredClusters)
	
	namespaceText := "all"
	if m.namespace != "" {
		namespaceText = m.namespace
	}
	
	text = fmt.Sprintf("Clusters(%s) [%d]", namespaceText, count)
	
	// Add filter indicator
	if m.filter != "" {
		text += fmt.Sprintf(" /%s", m.filter)
	}
	
	// Add loading indicator
	if m.loading {
		text += " " + m.spinner.View()
	}
	
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(colorBreadcrumbBg).
		Bold(false).
		Width(m.width).
		Padding(0, 1).
		Align(lipgloss.Left)
	
	return style.Render(text)
}

// renderFooter renders the k9s-style footer with keybindings and status  
func (m Model) renderFooter() string {
	// Key bindings (left side)
	var help string
	
	switch m.state {
	case ClusterListView:
		help = "<c>Create <d>Describe <e>Edit <x>Delete <k>Kubeconfig </>Filter <?> Help"
	case ClusterDetailView, KubeconfigView:
		help = "<esc>Back <?> Help"
	case CreateClusterView, EditClusterView:
		help = "<tab>Next <shift+tab>Previous <enter>Submit <esc>Cancel"
	case DeleteConfirmView:
		help = "<enter>Delete <esc>Cancel"
	case HelpView:
		help = "<esc>Back"
	}

	// Status info (right side)
	refreshStatus := ""
	if !m.lastRefresh.IsZero() {
		elapsed := time.Since(m.lastRefresh)
		refreshStatus = fmt.Sprintf("⟳ %ds", int(elapsed.Seconds()))
	}
	
	statusText := fmt.Sprintf("%s | k3k.io/v1beta1 | %s", m.contextName, refreshStatus)
	if !m.lastRefresh.IsZero() {
		statusText += fmt.Sprintf("    %s", m.lastRefresh.Format("2006-01-02 15:04"))
	}

	// Layout
	helpStyle := lipgloss.NewStyle().
		Foreground(colorHelp).
		Align(lipgloss.Left)

	statusStyle := lipgloss.NewStyle().
		Foreground(colorHelp).
		Align(lipgloss.Right)

	footerStyle := lipgloss.NewStyle().
		Width(m.width).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(colorHelp).
		Padding(0, 1)

	return footerStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			helpStyle.Width(m.width-len(statusText)-4).Render(help),
			statusStyle.Render(statusText),
		),
	)
}

// updateTable updates the cluster table with k9s-style formatting
func (m *Model) updateTable() {
	rows := make([]table.Row, len(m.filteredClusters))
	
	for i, cluster := range m.filteredClusters {
		// Status with color coding
		status := cluster.Status.Phase
		if status == "" {
			status = "Unknown"
		}
		
		// Mode with color coding
		mode := cluster.Spec.Mode
		
		// Format numbers (right-aligned)
		servers := "0"
		if cluster.Spec.Servers != nil {
			servers = fmt.Sprintf("%d", *cluster.Spec.Servers)
		}

		agents := "0"
		if cluster.Spec.Agents != nil {
			agents = fmt.Sprintf("%d", *cluster.Spec.Agents)
		}

		// Age in k9s format
		age := k8s.Age(cluster.ObjectMeta.CreationTimestamp)

		rows[i] = table.Row{
			cluster.Name,
			cluster.Namespace,
			mode,
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