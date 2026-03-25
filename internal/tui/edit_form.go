package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/malagant/k3k-tui/internal/k8s"
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

// EditForm handles the cluster edit form
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
	
	// State
	currentInput int
	
	// UI
	width  int
	height int
}

// NewEditForm creates a new edit form
func NewEditForm(cluster *types.Cluster) *EditForm {
	f := &EditForm{
		step:            EditStepServers,
		originalCluster: cluster,
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

// initInputs initializes the text inputs
func (f *EditForm) initInputs() {
	inputs := make([]textinput.Model, 4)

	// Servers input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Number of servers"
	inputs[0].SetValue(fmt.Sprintf("%d", f.servers))
	inputs[0].Focus()
	inputs[0].CharLimit = 3
	inputs[0].Width = 10

	// Agents input
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Number of agents"
	inputs[1].SetValue(fmt.Sprintf("%d", f.agents))
	inputs[1].CharLimit = 3
	inputs[1].Width = 10

	// Version input
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "K3s version (optional)"
	inputs[2].SetValue(f.version)
	inputs[2].CharLimit = 50
	inputs[2].Width = 30

	// Server args input (we'll use this for both server and agent args)
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "Additional arguments (space separated)"
	inputs[3].CharLimit = 500
	inputs[3].Width = 60

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
			f.inputs[1].Focus()
		} else {
			f.step = EditStepVersion
			f.inputs[2].Focus()
		}
	case EditStepAgents:
		f.step = EditStepVersion
		f.inputs[2].Focus()
	case EditStepVersion:
		f.step = EditStepServerArgs
		// Set up server args input
		f.inputs[3].SetValue(strings.Join(f.serverArgs, " "))
		f.inputs[3].Focus()
	case EditStepServerArgs:
		f.step = EditStepAgentArgs
		// Set up agent args input
		f.inputs[3].SetValue(strings.Join(f.agentArgs, " "))
		f.inputs[3].Focus()
	case EditStepAgentArgs:
		f.step = EditStepConfirm
	}
}

// Previous moves to the previous form step
func (f *EditForm) Previous() {
	switch f.step {
	case EditStepAgents:
		f.step = EditStepServers
		f.inputs[0].Focus()
	case EditStepVersion:
		if f.originalCluster.Spec.Mode != "shared" {
			f.step = EditStepAgents
			f.inputs[1].Focus()
		} else {
			f.step = EditStepServers
			f.inputs[0].Focus()
		}
	case EditStepServerArgs:
		f.step = EditStepVersion
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

// View renders the edit form
func (f *EditForm) View() string {
	var content strings.Builder
	
	content.WriteString(fmt.Sprintf("Edit Cluster: %s/%s\n", f.originalCluster.Namespace, f.originalCluster.Name))
	content.WriteString(strings.Repeat("=", 40) + "\n\n")

	totalSteps := 5
	if f.originalCluster.Spec.Mode == "shared" {
		totalSteps = 4
	}

	switch f.step {
	case EditStepServers:
		currentStep := 1
		content.WriteString(fmt.Sprintf("Step %d/%d: Server Nodes\n\n", currentStep, totalSteps))
		content.WriteString("Number of server nodes (minimum 1):\n")
		content.WriteString(fmt.Sprintf("Current: %d\n", *f.originalCluster.Spec.Servers))
		content.WriteString("New: " + f.inputs[0].View() + "\n")
		
	case EditStepAgents:
		currentStep := 2
		content.WriteString(fmt.Sprintf("Step %d/%d: Agent Nodes\n\n", currentStep, totalSteps))
		content.WriteString("Number of agent nodes (minimum 0):\n")
		if f.originalCluster.Spec.Agents != nil {
			content.WriteString(fmt.Sprintf("Current: %d\n", *f.originalCluster.Spec.Agents))
		} else {
			content.WriteString("Current: 0\n")
		}
		content.WriteString("New: " + f.inputs[1].View() + "\n")
		
	case EditStepVersion:
		currentStep := 2
		if f.originalCluster.Spec.Mode != "shared" {
			currentStep = 3
		}
		content.WriteString(fmt.Sprintf("Step %d/%d: K3s Version\n\n", currentStep, totalSteps))
		content.WriteString("K3s version (leave empty for no change):\n")
		if f.originalCluster.Spec.Version != "" {
			content.WriteString(fmt.Sprintf("Current: %s\n", f.originalCluster.Spec.Version))
		} else {
			content.WriteString("Current: (default)\n")
		}
		content.WriteString("New: " + f.inputs[2].View() + "\n")
		
	case EditStepServerArgs:
		currentStep := 3
		if f.originalCluster.Spec.Mode != "shared" {
			currentStep = 4
		}
		content.WriteString(fmt.Sprintf("Step %d/%d: Server Arguments\n\n", currentStep, totalSteps))
		content.WriteString("Additional arguments for K3s server:\n")
		if len(f.originalCluster.Spec.ServerArgs) > 0 {
			content.WriteString(fmt.Sprintf("Current: %s\n", strings.Join(f.originalCluster.Spec.ServerArgs, " ")))
		} else {
			content.WriteString("Current: (none)\n")
		}
		content.WriteString("New: " + f.inputs[3].View() + "\n")
		
	case EditStepAgentArgs:
		currentStep := 4
		if f.originalCluster.Spec.Mode != "shared" {
			currentStep = 5
		}
		content.WriteString(fmt.Sprintf("Step %d/%d: Agent Arguments\n\n", currentStep, totalSteps))
		content.WriteString("Additional arguments for K3s agent:\n")
		if len(f.originalCluster.Spec.AgentArgs) > 0 {
			content.WriteString(fmt.Sprintf("Current: %s\n", strings.Join(f.originalCluster.Spec.AgentArgs, " ")))
		} else {
			content.WriteString("Current: (none)\n")
		}
		content.WriteString("New: " + f.inputs[3].View() + "\n")
		
	case EditStepConfirm:
		content.WriteString(fmt.Sprintf("Step %d/%d: Confirm Changes\n\n", totalSteps, totalSteps))
		content.WriteString("Review your changes:\n\n")
		
		// Show changes
		content.WriteString("Changes to be applied:\n")
		
		if f.originalCluster.Spec.Servers == nil || *f.originalCluster.Spec.Servers != f.servers {
			content.WriteString(fmt.Sprintf("  Servers: %d → %d\n", 
				func() int32 {
					if f.originalCluster.Spec.Servers != nil {
						return *f.originalCluster.Spec.Servers
					}
					return 0
				}(), f.servers))
		}
		
		if f.originalCluster.Spec.Mode != "shared" {
			if f.originalCluster.Spec.Agents == nil || *f.originalCluster.Spec.Agents != f.agents {
				content.WriteString(fmt.Sprintf("  Agents: %d → %d\n", 
					func() int32 {
						if f.originalCluster.Spec.Agents != nil {
							return *f.originalCluster.Spec.Agents
						}
						return 0
					}(), f.agents))
			}
		}
		
		if f.originalCluster.Spec.Version != f.version {
			content.WriteString(fmt.Sprintf("  Version: \"%s\" → \"%s\"\n", f.originalCluster.Spec.Version, f.version))
		}
		
		if !stringSliceEqual(f.originalCluster.Spec.ServerArgs, f.serverArgs) {
			content.WriteString(fmt.Sprintf("  Server Args: [%s] → [%s]\n", 
				strings.Join(f.originalCluster.Spec.ServerArgs, " "), 
				strings.Join(f.serverArgs, " ")))
		}
		
		if !stringSliceEqual(f.originalCluster.Spec.AgentArgs, f.agentArgs) {
			content.WriteString(fmt.Sprintf("  Agent Args: [%s] → [%s]\n", 
				strings.Join(f.originalCluster.Spec.AgentArgs, " "), 
				strings.Join(f.agentArgs, " ")))
		}
		
		content.WriteString("\nPress Enter to apply changes, or Esc to cancel.\n")
		
		// Show YAML preview
		cluster := f.ToCluster()
		yaml, err := k8s.ClusterToYAML(cluster)
		if err == nil {
			content.WriteString("\nYAML Preview:\n")
			content.WriteString("-------------\n")
			content.WriteString(yaml)
		}
	}

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("214")).
		Padding(1, 2).
		Width(80)

	return style.Render(content.String())
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