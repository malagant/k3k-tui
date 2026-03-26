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

// View renders the edit form as a centered Catppuccin Mocha modal
func (f *EditForm) View() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(catPeach).
		Bold(true).
		Align(lipgloss.Center).
		Width(68)

	clusterTitle := fmt.Sprintf("✎ EDIT: %s/%s", f.originalCluster.Namespace, f.originalCluster.Name)
	content.WriteString(titleStyle.Render(clusterTitle) + "\n\n")

	// Progress bar
	totalSteps := 5
	if f.originalCluster.Spec.Mode == "shared" {
		totalSteps = 4
	}
	currentStep := int(f.step) + 1
	if f.originalCluster.Spec.Mode == "shared" && f.step >= EditStepVersion {
		currentStep = int(f.step)
	}
	content.WriteString(f.renderProgressBar(currentStep, totalSteps) + "\n\n")

	descStyle := lipgloss.NewStyle().Foreground(catSubtext0)

	switch f.step {
	case EditStepServers:
		content.WriteString(f.renderStepTitle("Server Nodes"))
		content.WriteString(descStyle.Render("Number of server nodes (min 1):") + "\n\n")
		cv := "1"
		if f.originalCluster.Spec.Servers != nil {
			cv = fmt.Sprintf("%d", *f.originalCluster.Spec.Servers)
		}
		content.WriteString(f.renderCurrentVsNew("Current", cv, "New", f.inputs[0].View()))

	case EditStepAgents:
		content.WriteString(f.renderStepTitle("Agent Nodes"))
		content.WriteString(descStyle.Render("Number of agent nodes (min 0):") + "\n\n")
		cv := "0"
		if f.originalCluster.Spec.Agents != nil {
			cv = fmt.Sprintf("%d", *f.originalCluster.Spec.Agents)
		}
		content.WriteString(f.renderCurrentVsNew("Current", cv, "New", f.inputs[1].View()))

	case EditStepVersion:
		content.WriteString(f.renderStepTitle("K3s Version"))
		content.WriteString(descStyle.Render("Leave empty for no change:") + "\n\n")
		cv := f.originalCluster.Spec.Version
		if cv == "" {
			cv = "(default)"
		}
		content.WriteString(f.renderCurrentVsNew("Current", cv, "New", f.inputs[2].View()))

	case EditStepServerArgs:
		content.WriteString(f.renderStepTitle("Server Arguments"))
		content.WriteString(descStyle.Render("Additional K3s server arguments:") + "\n\n")
		cv := strings.Join(f.originalCluster.Spec.ServerArgs, " ")
		if cv == "" {
			cv = "(none)"
		}
		content.WriteString(f.renderCurrentVsNew("Current", cv, "New", f.inputs[3].View()))

	case EditStepAgentArgs:
		content.WriteString(f.renderStepTitle("Agent Arguments"))
		content.WriteString(descStyle.Render("Additional K3s agent arguments:") + "\n\n")
		cv := strings.Join(f.originalCluster.Spec.AgentArgs, " ")
		if cv == "" {
			cv = "(none)"
		}
		content.WriteString(f.renderCurrentVsNew("Current", cv, "New", f.inputs[3].View()))

	case EditStepConfirm:
		content.WriteString(f.renderStepTitle("Confirm Changes"))

		hasChanges := false
		kS := lipgloss.NewStyle().Foreground(catBlue).Bold(true).Width(14)
		oldS := lipgloss.NewStyle().Foreground(catRed)
		newS := lipgloss.NewStyle().Foreground(catGreen)
		arrowS := lipgloss.NewStyle().Foreground(catOverlay1)

		diffLine := func(label, old, new string) string {
			return fmt.Sprintf("%s%s %s %s\n", kS.Render(label), oldS.Render(old), arrowS.Render("→"), newS.Render(new))
		}

		if f.originalCluster.Spec.Servers == nil || *f.originalCluster.Spec.Servers != f.servers {
			ov := "0"
			if f.originalCluster.Spec.Servers != nil {
				ov = fmt.Sprintf("%d", *f.originalCluster.Spec.Servers)
			}
			content.WriteString(diffLine("Servers:", ov, fmt.Sprintf("%d", f.servers)))
			hasChanges = true
		}

		if f.originalCluster.Spec.Mode != "shared" {
			if f.originalCluster.Spec.Agents == nil || *f.originalCluster.Spec.Agents != f.agents {
				ov := "0"
				if f.originalCluster.Spec.Agents != nil {
					ov = fmt.Sprintf("%d", *f.originalCluster.Spec.Agents)
				}
				content.WriteString(diffLine("Agents:", ov, fmt.Sprintf("%d", f.agents)))
				hasChanges = true
			}
		}

		if f.originalCluster.Spec.Version != f.version {
			ov, nv := f.originalCluster.Spec.Version, f.version
			if ov == "" {
				ov = "(default)"
			}
			if nv == "" {
				nv = "(default)"
			}
			content.WriteString(diffLine("Version:", ov, nv))
			hasChanges = true
		}

		if !stringSliceEqual(f.originalCluster.Spec.ServerArgs, f.serverArgs) {
			ov := strings.Join(f.originalCluster.Spec.ServerArgs, " ")
			nv := strings.Join(f.serverArgs, " ")
			if ov == "" {
				ov = "(none)"
			}
			if nv == "" {
				nv = "(none)"
			}
			content.WriteString(diffLine("Server Args:", ov, nv))
			hasChanges = true
		}

		if !stringSliceEqual(f.originalCluster.Spec.AgentArgs, f.agentArgs) {
			ov := strings.Join(f.originalCluster.Spec.AgentArgs, " ")
			nv := strings.Join(f.agentArgs, " ")
			if ov == "" {
				ov = "(none)"
			}
			if nv == "" {
				nv = "(none)"
			}
			content.WriteString(diffLine("Agent Args:", ov, nv))
			hasChanges = true
		}

		if !hasChanges {
			content.WriteString(lipgloss.NewStyle().Foreground(catOverlay0).Render("No changes detected.") + "\n")
		}

		content.WriteString("\n")
		if hasChanges {
			content.WriteString(lipgloss.NewStyle().Foreground(catGreen).Render("Press Enter to apply changes") + "\n")
		} else {
			content.WriteString(lipgloss.NewStyle().Foreground(catOverlay0).Render("Press Esc to cancel") + "\n")
		}
	}

	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(catOverlay0).Render(
		"Tab: Next • Shift+Tab: Back • Enter: Continue • Esc: Cancel"))

	modalStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(catYellow).
		Background(catMantle).
		Padding(1, 3).
		Width(74)

	modal := modalStyle.Render(content.String())

	return lipgloss.Place(
		f.width, f.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// renderStepTitle renders a step title (Catppuccin)
func (f *EditForm) renderStepTitle(title string) string {
	return lipgloss.NewStyle().
		Foreground(catBlue).
		Bold(true).
		Render(title) + "\n"
}

// renderProgressBar renders a Catppuccin-styled progress bar
func (f *EditForm) renderProgressBar(current, total int) string {
	barWidth := 56
	filled := int(float64(current) / float64(total) * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	doneStyle := lipgloss.NewStyle().Foreground(catYellow)
	todoStyle := lipgloss.NewStyle().Foreground(catSurface1)
	labelStyle := lipgloss.NewStyle().Foreground(catOverlay1)

	bar := doneStyle.Render(strings.Repeat("━", filled)) +
		todoStyle.Render(strings.Repeat("━", barWidth-filled))

	return bar + " " + labelStyle.Render(fmt.Sprintf("%d/%d", current, total))
}

// renderCurrentVsNew renders a current vs new value comparison
func (f *EditForm) renderCurrentVsNew(currentLabel, currentValue, newLabel, newInput string) string {
	kS := lipgloss.NewStyle().Foreground(catBlue).Bold(true)
	vS := lipgloss.NewStyle().Foreground(catOverlay1)

	return fmt.Sprintf("%s %s\n%s %s\n", kS.Render(currentLabel+":"), vS.Render(currentValue), kS.Render(newLabel+":"), newInput)
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