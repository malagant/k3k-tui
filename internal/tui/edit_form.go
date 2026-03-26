package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"


	"github.com/malagant/k3k-tui/internal/types"
)

// EditFormStep represents a step in the edit form
type EditFormStep int

const (
	EditStepServers EditFormStep = iota
	EditStepAgents
	EditStepVersion
	EditStepServerArgs
	EditStepAgentArgs
	EditStepConfirm
)

// EditForm handles the cluster edit form with k9s modal styling
type EditForm struct {
	step         EditFormStep
	inputs       []textinput.Model
	originalCluster *types.Cluster
	
	// Form values
	servers    int32
	agents     int32
	version    string
	serverArgs []string
	agentArgs  []string
	
	// UI dimensions
	width  int
	height int
}

// NewEditForm creates a new edit form
func NewEditForm(cluster *types.Cluster) *EditForm {
	f := &EditForm{
		step:            EditStepServers,
		originalCluster: cluster,
		width:           80,
		height:         25,
	}

	// Initialize with current values
	if cluster.Spec.Servers != nil {
		f.servers = *cluster.Spec.Servers
	} else {
		f.servers = types.DefaultServers
	}

	if cluster.Spec.Agents != nil {
		f.agents = *cluster.Spec.Agents
	} else {
		f.agents = types.DefaultAgents
	}

	f.version = cluster.Spec.Version
	f.serverArgs = cluster.Spec.ServerArgs
	f.agentArgs = cluster.Spec.AgentArgs

	f.initInputs()
	return f
}

// initInputs initializes the text inputs with k9s styling
func (f *EditForm) initInputs() {
	inputs := make([]textinput.Model, 4)

	inputStyle := lipgloss.NewStyle().
		Foreground(colorHeaderText).
		Background(colorBg)

	focusedStyle := lipgloss.NewStyle().
		Foreground(colorCommand).
		Background(colorBg)

	// Servers input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Number of servers"
	inputs[0].SetValue(fmt.Sprintf("%d", f.servers))
	inputs[0].Focus()
	inputs[0].CharLimit = 3
	inputs[0].Width = 10
	inputs[0].TextStyle = inputStyle
	inputs[0].PlaceholderStyle = lipgloss.NewStyle().Foreground(colorHelp)
	inputs[0].Cursor.Style = focusedStyle

	// Agents input
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Number of agents"
	inputs[1].SetValue(fmt.Sprintf("%d", f.agents))
	inputs[1].CharLimit = 3
	inputs[1].Width = 10
	inputs[1].TextStyle = inputStyle
	inputs[1].PlaceholderStyle = lipgloss.NewStyle().Foreground(colorHelp)
	inputs[1].Cursor.Style = focusedStyle

	// Version input
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "K3s version (optional)"
	inputs[2].SetValue(f.version)
	inputs[2].CharLimit = 50
	inputs[2].Width = 30
	inputs[2].TextStyle = inputStyle
	inputs[2].PlaceholderStyle = lipgloss.NewStyle().Foreground(colorHelp)
	inputs[2].Cursor.Style = focusedStyle

	// Args input (we'll use this for both server and agent args)
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "Additional arguments (space separated)"
	inputs[3].CharLimit = 500
	inputs[3].Width = 60
	inputs[3].TextStyle = inputStyle
	inputs[3].PlaceholderStyle = lipgloss.NewStyle().Foreground(colorHelp)
	inputs[3].Cursor.Style = focusedStyle

	f.inputs = inputs
}

// Update handles form updates
func (f *EditForm) Update(msg tea.KeyMsg) {
	switch f.step {
	case EditStepServers:
		f.inputs[0], _ = f.inputs[0].Update(msg)
		if val, err := strconv.Atoi(f.inputs[0].Value()); err == nil && val >= 1 {
			f.servers = int32(val)
		}
	case EditStepAgents:
		f.inputs[1], _ = f.inputs[1].Update(msg)
		if val, err := strconv.Atoi(f.inputs[1].Value()); err == nil && val >= 0 {
			f.agents = int32(val)
		}
	case EditStepVersion:
		f.inputs[2], _ = f.inputs[2].Update(msg)
		f.version = f.inputs[2].Value()
	case EditStepServerArgs:
		f.inputs[3], _ = f.inputs[3].Update(msg)
		if f.inputs[3].Value() != "" {
			f.serverArgs = strings.Fields(f.inputs[3].Value())
		} else {
			f.serverArgs = nil
		}
	case EditStepAgentArgs:
		f.inputs[3], _ = f.inputs[3].Update(msg)
		if f.inputs[3].Value() != "" {
			f.agentArgs = strings.Fields(f.inputs[3].Value())
		} else {
			f.agentArgs = nil
		}
	}
}

// Next moves to the next form step
func (f *EditForm) Next() {
	switch f.step {
	case EditStepServers:
		if f.originalCluster.Spec.Mode != "shared" {
			f.step = EditStepAgents
			f.inputs[0].Blur()
			f.inputs[1].Focus()
		} else {
			f.step = EditStepVersion
			f.inputs[0].Blur()
			f.inputs[2].Focus()
		}
	case EditStepAgents:
		f.step = EditStepVersion
		f.inputs[1].Blur()
		f.inputs[2].Focus()
	case EditStepVersion:
		f.step = EditStepServerArgs
		// Set up server args input
		f.inputs[3].SetValue(strings.Join(f.serverArgs, " "))
		f.inputs[2].Blur()
		f.inputs[3].Focus()
	case EditStepServerArgs:
		f.step = EditStepAgentArgs
		// Set up agent args input
		f.inputs[3].SetValue(strings.Join(f.agentArgs, " "))
		f.inputs[3].Focus() // Already focused but ensure it
	case EditStepAgentArgs:
		f.step = EditStepConfirm
		f.inputs[3].Blur()
	}
}

// Previous moves to the previous form step
func (f *EditForm) Previous() {
	switch f.step {
	case EditStepAgents:
		f.step = EditStepServers
		f.inputs[1].Blur()
		f.inputs[0].Focus()
	case EditStepVersion:
		if f.originalCluster.Spec.Mode != "shared" {
			f.step = EditStepAgents
			f.inputs[2].Blur()
			f.inputs[1].Focus()
		} else {
			f.step = EditStepServers
			f.inputs[2].Blur()
			f.inputs[0].Focus()
		}
	case EditStepServerArgs:
		f.step = EditStepVersion
		f.inputs[3].Blur()
		f.inputs[2].Focus()
	case EditStepAgentArgs:
		f.step = EditStepServerArgs
		f.inputs[3].SetValue(strings.Join(f.serverArgs, " "))
		f.inputs[3].Focus()
	case EditStepConfirm:
		f.step = EditStepAgentArgs
		f.inputs[3].SetValue(strings.Join(f.agentArgs, " "))
		f.inputs[3].Focus()
	}
}

// IsComplete returns whether the form is ready to submit
func (f *EditForm) IsComplete() bool {
	return f.step == EditStepConfirm
}

// ToCluster converts the form data to an updated Cluster object
func (f *EditForm) ToCluster() *types.Cluster {
	// Create a copy of the original cluster
	cluster := f.originalCluster.DeepCopy()

	// Update mutable fields
	cluster.Spec.Servers = &f.servers
	if f.originalCluster.Spec.Mode != "shared" {
		cluster.Spec.Agents = &f.agents
	}
	cluster.Spec.Version = f.version
	cluster.Spec.ServerArgs = f.serverArgs
	cluster.Spec.AgentArgs = f.agentArgs

	return cluster
}

// View renders the edit form in k9s modal style
func (f *EditForm) View() string {
	var content strings.Builder
	
	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(colorYamlHeader).
		Bold(true).
		Align(lipgloss.Center).
		Width(74)
	
	clusterTitle := fmt.Sprintf("EDIT CLUSTER: %s/%s", f.originalCluster.Namespace, f.originalCluster.Name)
	content.WriteString(titleStyle.Render(clusterTitle) + "\n\n")

	// Progress indicator
	totalSteps := 5
	if f.originalCluster.Spec.Mode == "shared" {
		totalSteps = 4
	}
	
	currentStep := int(f.step) + 1
	if f.originalCluster.Spec.Mode == "shared" && f.step >= EditStepVersion {
		currentStep = int(f.step) // Adjust for skipped agent step
	}
	
	progressBar := f.renderProgressBar(currentStep, totalSteps)
	content.WriteString(progressBar + "\n\n")

	// Step content
	switch f.step {
	case EditStepServers:
		content.WriteString(f.renderStepTitle("Server Nodes"))
		content.WriteString("Number of server nodes (minimum 1):\n\n")
		
		// Show current vs new
		currentValue := "1"
		if f.originalCluster.Spec.Servers != nil {
			currentValue = fmt.Sprintf("%d", *f.originalCluster.Spec.Servers)
		}
		
		content.WriteString(f.renderCurrentVsNew("Current", currentValue, "New", f.inputs[0].View()))
		
	case EditStepAgents:
		content.WriteString(f.renderStepTitle("Agent Nodes"))
		content.WriteString("Number of agent nodes (minimum 0):\n\n")
		
		currentValue := "0"
		if f.originalCluster.Spec.Agents != nil {
			currentValue = fmt.Sprintf("%d", *f.originalCluster.Spec.Agents)
		}
		
		content.WriteString(f.renderCurrentVsNew("Current", currentValue, "New", f.inputs[1].View()))
		
	case EditStepVersion:
		content.WriteString(f.renderStepTitle("K3s Version"))
		content.WriteString("K3s version (leave empty for no change):\n\n")
		
		currentValue := f.originalCluster.Spec.Version
		if currentValue == "" {
			currentValue = "(default)"
		}
		
		content.WriteString(f.renderCurrentVsNew("Current", currentValue, "New", f.inputs[2].View()))
		
	case EditStepServerArgs:
		content.WriteString(f.renderStepTitle("Server Arguments"))
		content.WriteString("Additional arguments for K3s server:\n\n")
		
		currentValue := strings.Join(f.originalCluster.Spec.ServerArgs, " ")
		if currentValue == "" {
			currentValue = "(none)"
		}
		
		content.WriteString(f.renderCurrentVsNew("Current", currentValue, "New", f.inputs[3].View()))
		
	case EditStepAgentArgs:
		content.WriteString(f.renderStepTitle("Agent Arguments"))
		content.WriteString("Additional arguments for K3s agent:\n\n")
		
		currentValue := strings.Join(f.originalCluster.Spec.AgentArgs, " ")
		if currentValue == "" {
			currentValue = "(none)"
		}
		
		content.WriteString(f.renderCurrentVsNew("Current", currentValue, "New", f.inputs[3].View()))
		
	case EditStepConfirm:
		content.WriteString(f.renderStepTitle("Confirm Changes"))
		content.WriteString("Review the changes to be applied:\n\n")
		
		// Show only fields that changed
		hasChanges := false
		
		keyStyle := lipgloss.NewStyle().Foreground(colorYamlKey).Bold(true)
		oldStyle := lipgloss.NewStyle().Foreground(colorFailed)
		newStyle := lipgloss.NewStyle().Foreground(colorRunning)
		arrowStyle := lipgloss.NewStyle().Foreground(colorTableHeader)
		
		if f.originalCluster.Spec.Servers == nil || *f.originalCluster.Spec.Servers != f.servers {
			oldVal := "0"
			if f.originalCluster.Spec.Servers != nil {
				oldVal = fmt.Sprintf("%d", *f.originalCluster.Spec.Servers)
			}
			content.WriteString(fmt.Sprintf("%s %s %s %s\n", 
				keyStyle.Render("Servers:"), 
				oldStyle.Render(oldVal), 
				arrowStyle.Render("→"), 
				newStyle.Render(fmt.Sprintf("%d", f.servers))))
			hasChanges = true
		}
		
		if f.originalCluster.Spec.Mode != "shared" {
			if f.originalCluster.Spec.Agents == nil || *f.originalCluster.Spec.Agents != f.agents {
				oldVal := "0"
				if f.originalCluster.Spec.Agents != nil {
					oldVal = fmt.Sprintf("%d", *f.originalCluster.Spec.Agents)
				}
				content.WriteString(fmt.Sprintf("%s %s %s %s\n", 
					keyStyle.Render("Agents:"), 
					oldStyle.Render(oldVal), 
					arrowStyle.Render("→"), 
					newStyle.Render(fmt.Sprintf("%d", f.agents))))
				hasChanges = true
			}
		}
		
		if f.originalCluster.Spec.Version != f.version {
			oldVal := f.originalCluster.Spec.Version
			if oldVal == "" {
				oldVal = "(default)"
			}
			newVal := f.version
			if newVal == "" {
				newVal = "(default)"
			}
			content.WriteString(fmt.Sprintf("%s %s %s %s\n", 
				keyStyle.Render("Version:"), 
				oldStyle.Render(oldVal), 
				arrowStyle.Render("→"), 
				newStyle.Render(newVal)))
			hasChanges = true
		}
		
		if !stringSliceEqual(f.originalCluster.Spec.ServerArgs, f.serverArgs) {
			oldVal := strings.Join(f.originalCluster.Spec.ServerArgs, " ")
			if oldVal == "" {
				oldVal = "(none)"
			}
			newVal := strings.Join(f.serverArgs, " ")
			if newVal == "" {
				newVal = "(none)"
			}
			content.WriteString(fmt.Sprintf("%s %s %s %s\n", 
				keyStyle.Render("Server Args:"), 
				oldStyle.Render(oldVal), 
				arrowStyle.Render("→"), 
				newStyle.Render(newVal)))
			hasChanges = true
		}
		
		if !stringSliceEqual(f.originalCluster.Spec.AgentArgs, f.agentArgs) {
			oldVal := strings.Join(f.originalCluster.Spec.AgentArgs, " ")
			if oldVal == "" {
				oldVal = "(none)"
			}
			newVal := strings.Join(f.agentArgs, " ")
			if newVal == "" {
				newVal = "(none)"
			}
			content.WriteString(fmt.Sprintf("%s %s %s %s\n", 
				keyStyle.Render("Agent Args:"), 
				oldStyle.Render(oldVal), 
				arrowStyle.Render("→"), 
				newStyle.Render(newVal)))
			hasChanges = true
		}
		
		if !hasChanges {
			content.WriteString(lipgloss.NewStyle().Foreground(colorHelp).Render("No changes detected.\n"))
		}
		
		content.WriteString("\n")
		if hasChanges {
			content.WriteString(lipgloss.NewStyle().Foreground(colorCommand).Render("Press Enter to apply changes") + "\n")
		} else {
			content.WriteString(lipgloss.NewStyle().Foreground(colorHelp).Render("Press Esc to cancel") + "\n")
		}
	}
	
	// Instructions
	content.WriteString("\n")
	instrStyle := lipgloss.NewStyle().Foreground(colorHelp)
	content.WriteString(instrStyle.Render("Tab: Next • Shift+Tab: Previous • Enter: Continue • Esc: Cancel"))

	// Modal style with rounded border like k9s but in orange for edit
	modalStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorPending).
		Background(colorBg).
		Padding(2, 3).
		Width(80).
		Height(25).
		Align(lipgloss.Left, lipgloss.Top)

	return modalStyle.Render(content.String())
}

// renderStepTitle renders a step title with k9s styling
func (f *EditForm) renderStepTitle(title string) string {
	style := lipgloss.NewStyle().
		Foreground(colorTableHeader).
		Bold(true).
		Margin(0, 0, 1, 0)
	return style.Render(title) + "\n"
}

// renderProgressBar renders a k9s-style progress bar
func (f *EditForm) renderProgressBar(current, total int) string {
	barWidth := 60
	filled := int(float64(current) / float64(total) * float64(barWidth))
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	
	progressStyle := lipgloss.NewStyle().
		Foreground(colorPending) // Orange for edit form
	
	labelStyle := lipgloss.NewStyle().
		Foreground(colorHelp)
	
	return progressStyle.Render(bar) + " " + labelStyle.Render(fmt.Sprintf("%d/%d", current, total))
}

// renderCurrentVsNew renders a current vs new value comparison
func (f *EditForm) renderCurrentVsNew(currentLabel, currentValue, newLabel, newInput string) string {
	keyStyle := lipgloss.NewStyle().Foreground(colorYamlKey).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(colorHelp)
	
	current := fmt.Sprintf("%s: %s", keyStyle.Render(currentLabel), valueStyle.Render(currentValue))
	new := fmt.Sprintf("%s: %s", keyStyle.Render(newLabel), newInput)
	
	return current + "\n" + new + "\n"
}

// stringSliceEqual compares two string slices for equality
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}