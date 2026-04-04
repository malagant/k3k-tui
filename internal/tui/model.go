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

// Catppuccin Mocha color palette
// https://github.com/catppuccin/catppuccin
var (
	// Base colors
	catBase      = lipgloss.Color("#1e1e2e") // Base background
	catMantle    = lipgloss.Color("#181825") // Darker background
	catCrust     = lipgloss.Color("#11111b") // Darkest background
	catSurface0  = lipgloss.Color("#313244") // Surface
	catSurface1  = lipgloss.Color("#45475a") // Surface highlight
	catSurface2  = lipgloss.Color("#585b70") // Surface brighter
	catOverlay0  = lipgloss.Color("#6c7086") // Overlay muted
	catOverlay1  = lipgloss.Color("#7f849c") // Overlay
	catSubtext0  = lipgloss.Color("#a6adc8") // Subtext
	catSubtext1  = lipgloss.Color("#bac2de") // Subtext bright
	catText      = lipgloss.Color("#cdd6f4") // Main text
	
	// Accent colors
	catMauve     = lipgloss.Color("#cba6f7")
	catRed       = lipgloss.Color("#f38ba8")
	catPeach     = lipgloss.Color("#fab387")
	catYellow    = lipgloss.Color("#f9e2af")
	catGreen     = lipgloss.Color("#a6e3a1")
	catTeal      = lipgloss.Color("#94e2d5")
	catSapphire  = lipgloss.Color("#74c7ec")
	catBlue      = lipgloss.Color("#89b4fa")
	catLavender  = lipgloss.Color("#b4befe")

	// Semantic aliases (k9s-style mapping onto Catppuccin)
	colorBg           = catBase
	colorHeaderText   = catText
	colorTableHeader  = catBlue
	colorSelectedBg   = catSurface1
	colorSelectedText = catText
	colorRunning      = catGreen
	colorPending      = catYellow
	colorFailed       = catRed
	colorAge          = catOverlay1
	colorNamespace    = catSapphire
	colorModeShared   = catTeal
	colorModeVirtual  = catMauve
	colorHelp         = catOverlay0
	colorCommand      = catYellow
	colorYamlKey      = catBlue
	colorYamlValue    = catText
	colorYamlStatus   = catGreen
	colorYamlHeader   = catPeach
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
	runningK9s   bool
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

	// Catppuccin Mocha styled table (borderless, k9s-like)
	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		Foreground(catBlue).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(catSurface2).
		Align(lipgloss.Left)
	
	s.Selected = s.Selected.
		Foreground(catText).
		Background(catSurface1).
		Bold(false)
	
	s.Cell = s.Cell.
		Foreground(catSubtext1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(false).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false)
	
	t.SetStyles(s)

	// Initialize spinner for loading states
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(catMauve)

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
		m.autoRefreshTick(),
	)
}

// autoRefreshTick returns a command that sends a tickMsg after the refresh interval
func (m Model) autoRefreshTick() tea.Cmd {
	return tea.Tick(m.refreshInterval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
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
		// ctrl+c quits — but not while k9s is running (it needs ctrl+c itself)
		if msg.String() == "ctrl+c" && !m.runningK9s {
			return m, tea.Quit
		}

		// Global commands (work in any view except when typing)
		if !m.isInputFocused() {
			switch msg.String() {
			case "q":
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
				m.error = ""
				switch m.state {
				case HelpView, CommandView, FilterView:
					m.state = m.lastState
					return m, nil
				case ClusterDetailView, KubeconfigView:
					m.state = ClusterListView
					return m, nil
				case ClusterListView:
					// just clear error, stay in list
					return m, nil
				}
				// For other views (Create/Edit/Delete) — fall through to their handlers
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

	case tickMsg:
		// Don't do anything while k9s is running
		if m.runningK9s {
			return m, nil
		}
		// Auto-refresh: only reload when on the list view and not loading
		if m.state == ClusterListView && !m.loading {
			return m, tea.Batch(m.loadClusters(), m.autoRefreshTick())
		}
		// Keep ticking even if not refreshing now
		return m, m.autoRefreshTick()

	case k9sFinishedMsg:
		m.loading = false
		m.runningK9s = false
		if msg.err != nil {
			m.error = fmt.Sprintf("k9s: %v", msg.err)
		}
		// Refresh clusters and restart auto-refresh after returning from k9s
		cmds = append(cmds, m.loadClusters(), m.autoRefreshTick())

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
		Foreground(catYellow).
		Background(catCrust)
	
	return commandStyle.Render(prompt + input)
}

// renderHeader renders the k9s-style header with logo (Catppuccin Mocha)
func (m Model) renderHeader() string {
	keyStyle := lipgloss.NewStyle().Foreground(catOverlay1)
	valStyle := lipgloss.NewStyle().Foreground(catText).Bold(true)

	leftInfo := fmt.Sprintf("%s %s\n%s %s\n%s %s\n%s %s",
		keyStyle.Render("Context:"), valStyle.Render(m.contextName),
		keyStyle.Render("Cluster:"), valStyle.Render(m.clusterName),
		keyStyle.Render("K8s:"), valStyle.Render(m.k8sVersion),
		keyStyle.Render("k3k-tui:"), valStyle.Foreground(catMauve).Render(m.version))

	leftStyle := lipgloss.NewStyle().
		Width(35).
		Height(4).
		Align(lipgloss.Left, lipgloss.Top)

	logoStyle := lipgloss.NewStyle().
		Foreground(catMauve).
		Bold(true)

	rightStyle := lipgloss.NewStyle().
		Width(20).
		Height(4).
		Align(lipgloss.Right, lipgloss.Top)

	headerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(4)

	left := leftStyle.Render(leftInfo)
	right := rightStyle.Render(logoStyle.Render(k3kLogo))

	return headerStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			left,
			lipgloss.NewStyle().Width(m.width-55).Render(""),
			right,
		),
	)
}

// renderBreadcrumb renders the k9s-style breadcrumb bar (Catppuccin Mocha)
func (m Model) renderBreadcrumb() string {
	namespaceText := "all"
	if m.namespace != "" {
		namespaceText = m.namespace
	}

	count := len(m.filteredClusters)

	labelStyle := lipgloss.NewStyle().
		Foreground(catCrust).
		Background(catBlue).
		Bold(true).
		Padding(0, 1)

	countStyle := lipgloss.NewStyle().
		Foreground(catText).
		Background(catSurface0).
		Padding(0, 1)

	text := labelStyle.Render(fmt.Sprintf(" Clusters(%s) ", namespaceText))
	text += countStyle.Render(fmt.Sprintf(" %d ", count))

	// Filter indicator
	if m.filter != "" {
		filterStyle := lipgloss.NewStyle().
			Foreground(catYellow).
			Background(catSurface0).
			Padding(0, 1)
		text += filterStyle.Render(fmt.Sprintf("/%s", m.filter))
	}

	// Loading indicator
	if m.loading {
		text += " " + m.spinner.View()
	}

	barStyle := lipgloss.NewStyle().
		Background(catSurface0).
		Width(m.width).
		Padding(0, 0)

	return barStyle.Render(text)
}

// renderFooter renders the k9s-style footer with keybindings and status (Catppuccin Mocha)
func (m Model) renderFooter() string {
	keyStyle := lipgloss.NewStyle().Foreground(catMauve).Bold(true)
	actionStyle := lipgloss.NewStyle().Foreground(catSubtext0)

	// Build keybinding help with colored keys
	var helpParts []string

	switch m.state {
	case ClusterListView:
		helpParts = []string{
			keyStyle.Render("<c>") + actionStyle.Render("Create"),
			keyStyle.Render("<d>") + actionStyle.Render("Describe"),
			keyStyle.Render("<e>") + actionStyle.Render("Edit"),
			keyStyle.Render("<x>") + actionStyle.Render("Delete"),
			keyStyle.Render("<k>") + actionStyle.Render("Kubeconfig"),
			keyStyle.Render("<9>") + actionStyle.Render("k9s"),
			keyStyle.Render("</>") + actionStyle.Render("Filter"),
			keyStyle.Render("<?>") + actionStyle.Render("Help"),
		}
	case ClusterDetailView, KubeconfigView:
		helpParts = []string{
			keyStyle.Render("<esc>") + actionStyle.Render("Back"),
			keyStyle.Render("<?>") + actionStyle.Render("Help"),
		}
	case CreateClusterView, EditClusterView:
		helpParts = []string{
			keyStyle.Render("<tab>") + actionStyle.Render("Next"),
			keyStyle.Render("<S-tab>") + actionStyle.Render("Prev"),
			keyStyle.Render("<enter>") + actionStyle.Render("Submit"),
			keyStyle.Render("<esc>") + actionStyle.Render("Cancel"),
		}
	case DeleteConfirmView:
		helpParts = []string{
			keyStyle.Render("<enter>") + actionStyle.Render("Confirm"),
			keyStyle.Render("<esc>") + actionStyle.Render("Cancel"),
		}
	case HelpView:
		helpParts = []string{
			keyStyle.Render("<esc>") + actionStyle.Render("Back"),
		}
	}

	help := strings.Join(helpParts, "  ")

	// Status info (right side)
	statusParts := []string{m.contextName, "k3k.io/v1beta1"}
	if !m.lastRefresh.IsZero() {
		elapsed := time.Since(m.lastRefresh)
		statusParts = append(statusParts, fmt.Sprintf("⟳ %ds", int(elapsed.Seconds())))
		statusParts = append(statusParts, m.lastRefresh.Format("15:04"))
	}

	statusText := lipgloss.NewStyle().Foreground(catOverlay0).Render(strings.Join(statusParts, " │ "))

	footerStyle := lipgloss.NewStyle().
		Width(m.width).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(catSurface2).
		Padding(0, 1)

	helpWidth := m.width - lipgloss.Width(statusText) - 4
	if helpWidth < 20 {
		helpWidth = 20
	}

	return footerStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			lipgloss.NewStyle().Width(helpWidth).Render(help),
			statusText,
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