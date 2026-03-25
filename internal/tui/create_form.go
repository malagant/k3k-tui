package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/malagant/k3k-tui/internal/k8s"
	"github.com/malagant/k3k-tui/internal/types"
)

// FormStep represents a step in the form
type FormStep int

const (
	StepName FormStep = iota
	StepNamespace
	StepMode
	StepVersion
	StepServers
	StepAgents
	StepPersistence
	StepStorageClass
	StepConfirm
)

// CreateForm handles the cluster creation form
type CreateForm struct {
	step         FormStep
	inputs       []textinput.Model
	currentInput int
	
	// Form values
	name         string
	namespace    string
	mode         string // "shared" or "virtual"
	version      string
	servers      int32
	agents       int32
	persistence  string // "dynamic" or "ephemeral"
	storageClass string
	
	// State
	modeToggle       bool // false = shared, true = virtual
	persistenceToggle bool // false = dynamic, true = ephemeral
	
	// UI
	width  int
	height int
}

// NewCreateForm creates a new create form
func NewCreateForm() *CreateForm {
	f := &CreateForm{
		step:         StepName,
		servers:      types.DefaultServers,
		agents:       types.DefaultAgents,
		mode:         types.DefaultMode,
		persistence:  "dynamic",
		modeToggle:   false,
		persistenceToggle: false,
	}

	f.initInputs()
	return f
}

// initInputs initializes the text inputs
func (f *CreateForm) initInputs() {
	inputs := make([]textinput.Model, 5)

	// Name input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Enter cluster name"
	inputs[0].Focus()
	inputs[0].CharLimit = 253
	inputs[0].Width = 30

	// Namespace input
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Enter namespace"
	inputs[1].CharLimit = 253
	inputs[1].Width = 30

	// Version input
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "e.g., v1.31.3-k3s1 (optional)"
	inputs[2].CharLimit = 50
	inputs[2].Width = 30

	// Servers input
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "1"
	inputs[3].SetValue("1")
	inputs[3].CharLimit = 3
	inputs[3].Width = 10

	// Agents input
	inputs[4] = textinput.New()
	inputs[4].Placeholder = "0"
	inputs[4].SetValue("0")
	inputs[4].CharLimit = 3
	inputs[4].Width = 10

	f.inputs = inputs
}

// Update handles form updates
func (f *CreateForm) Update(msg tea.KeyMsg) {
	switch f.step {
	case StepName:
		f.inputs[0], _ = f.inputs[0].Update(msg)
		f.name = f.inputs[0].Value()
	case StepNamespace:
		f.inputs[1], _ = f.inputs[1].Update(msg)
		f.namespace = f.inputs[1].Value()
	case StepMode:
		if msg.String() == "space" {
			f.modeToggle = !f.modeToggle
			if f.modeToggle {
				f.mode = "virtual"
			} else {
				f.mode = "shared"
			}
		}
	case StepVersion:
		f.inputs[2], _ = f.inputs[2].Update(msg)
		f.version = f.inputs[2].Value()
	case StepServers:
		f.inputs[3], _ = f.inputs[3].Update(msg)
		if val, err := strconv.Atoi(f.inputs[3].Value()); err == nil && val >= 1 {
			f.servers = int32(val)
		}
	case StepAgents:
		f.inputs[4], _ = f.inputs[4].Update(msg)
		if val, err := strconv.Atoi(f.inputs[4].Value()); err == nil && val >= 0 {
			f.agents = int32(val)
		}
	case StepPersistence:
		if msg.String() == "space" {
			f.persistenceToggle = !f.persistenceToggle
			if f.persistenceToggle {
				f.persistence = "ephemeral"
			} else {
				f.persistence = "dynamic"
			}
		}
	case StepStorageClass:
		// Only show this step if persistence is dynamic
		if f.persistence == "dynamic" {
			// Storage class input (reuse an input)
			f.inputs[0].SetValue(f.storageClass)
			f.inputs[0], _ = f.inputs[0].Update(msg)
			f.storageClass = f.inputs[0].Value()
		}
	}
}

// Next moves to the next form step
func (f *CreateForm) Next() {
	switch f.step {
	case StepName:
		if f.name != "" {
			f.step = StepNamespace
			f.inputs[1].Focus()
		}
	case StepNamespace:
		if f.namespace != "" {
			f.step = StepMode
		}
	case StepMode:
		f.step = StepVersion
		f.inputs[2].Focus()
	case StepVersion:
		f.step = StepServers
		f.inputs[3].Focus()
	case StepServers:
		f.step = StepAgents
		f.inputs[4].Focus()
	case StepAgents:
		f.step = StepPersistence
	case StepPersistence:
		if f.persistence == "dynamic" {
			f.step = StepStorageClass
			f.inputs[0].SetValue(f.storageClass)
			f.inputs[0].Placeholder = "Storage class (optional)"
			f.inputs[0].Focus()
		} else {
			f.step = StepConfirm
		}
	case StepStorageClass:
		f.step = StepConfirm
	}
}

// Previous moves to the previous form step
func (f *CreateForm) Previous() {
	switch f.step {
	case StepNamespace:
		f.step = StepName
		f.inputs[0].Focus()
	case StepMode:
		f.step = StepNamespace
		f.inputs[1].Focus()
	case StepVersion:
		f.step = StepMode
	case StepServers:
		f.step = StepVersion
		f.inputs[2].Focus()
	case StepAgents:
		f.step = StepServers
		f.inputs[3].Focus()
	case StepPersistence:
		f.step = StepAgents
		f.inputs[4].Focus()
	case StepStorageClass:
		f.step = StepPersistence
	case StepConfirm:
		if f.persistence == "dynamic" {
			f.step = StepStorageClass
			f.inputs[0].Focus()
		} else {
			f.step = StepPersistence
		}
	}
}

// IsComplete returns whether the form is ready to submit
func (f *CreateForm) IsComplete() bool {
	return f.step == StepConfirm
}

// ToCluster converts the form data to a Cluster object
func (f *CreateForm) ToCluster() *types.Cluster {
	cluster := &types.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: types.APIVersion,
			Kind:       types.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.name,
			Namespace: f.namespace,
		},
		Spec: types.ClusterSpec{
			Mode:    f.mode,
			Servers: &f.servers,
			Agents:  &f.agents,
			Persistence: &types.PersistenceConfig{
				Type:               f.persistence,
				StorageRequestSize: types.DefaultStorageSize,
			},
		},
	}

	if f.version != "" {
		cluster.Spec.Version = f.version
	}

	if f.storageClass != "" && f.persistence == "dynamic" {
		cluster.Spec.Persistence.StorageClassName = f.storageClass
	}

	// Set default CIDRs based on mode
	if f.mode == "virtual" {
		cluster.Spec.ClusterCIDR = types.DefaultVirtualClusterCIDR
		cluster.Spec.ServiceCIDR = types.DefaultVirtualServiceCIDR
	} else {
		cluster.Spec.ClusterCIDR = types.DefaultSharedClusterCIDR
		cluster.Spec.ServiceCIDR = types.DefaultSharedServiceCIDR
	}
	cluster.Spec.ClusterDNS = types.DefaultClusterDNS

	return cluster
}

// View renders the form
func (f *CreateForm) View() string {
	var content strings.Builder
	
	content.WriteString("Create New k3k Cluster\n")
	content.WriteString(strings.Repeat("=", 30) + "\n\n")

	switch f.step {
	case StepName:
		content.WriteString("Step 1/8: Cluster Name\n\n")
		content.WriteString("Enter a name for your cluster:\n")
		content.WriteString(f.inputs[0].View() + "\n")
		
	case StepNamespace:
		content.WriteString("Step 2/8: Namespace\n\n")
		content.WriteString("Enter the namespace where the cluster will be created:\n")
		content.WriteString(f.inputs[1].View() + "\n")
		
	case StepMode:
		content.WriteString("Step 3/8: Cluster Mode\n\n")
		content.WriteString("Select cluster mode (use space to toggle):\n\n")
		
		sharedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		virtualStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		
		if !f.modeToggle {
			sharedStyle = sharedStyle.Foreground(lipgloss.Color("34")).Bold(true)
		} else {
			virtualStyle = virtualStyle.Foreground(lipgloss.Color("34")).Bold(true)
		}
		
		content.WriteString(fmt.Sprintf("[ ] %s\n", sharedStyle.Render("shared - Lightweight, shared control plane")))
		content.WriteString(fmt.Sprintf("[ ] %s\n", virtualStyle.Render("virtual - Full isolated virtual cluster")))
		
		if !f.modeToggle {
			content.WriteString("\n[x] shared")
		} else {
			content.WriteString("\n[x] virtual")
		}
		
	case StepVersion:
		content.WriteString("Step 4/8: K3s Version\n\n")
		content.WriteString("Enter K3s version (leave empty for default):\n")
		content.WriteString(f.inputs[2].View() + "\n")
		
	case StepServers:
		content.WriteString("Step 5/8: Server Nodes\n\n")
		content.WriteString("Number of server nodes (minimum 1):\n")
		content.WriteString(f.inputs[3].View() + "\n")
		
	case StepAgents:
		content.WriteString("Step 6/8: Agent Nodes\n\n")
		if f.mode == "shared" {
			content.WriteString("Agent nodes (ignored in shared mode, will be set to 0):\n")
		} else {
			content.WriteString("Number of agent nodes (minimum 0):\n")
		}
		content.WriteString(f.inputs[4].View() + "\n")
		
	case StepPersistence:
		totalSteps := "7/8"
		if f.persistence == "dynamic" {
			totalSteps = "7/9"
		}
		content.WriteString(fmt.Sprintf("Step %s: Persistence Type\n\n", totalSteps))
		content.WriteString("Select persistence type (use space to toggle):\n\n")
		
		dynamicStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		ephemeralStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		
		if !f.persistenceToggle {
			dynamicStyle = dynamicStyle.Foreground(lipgloss.Color("34")).Bold(true)
		} else {
			ephemeralStyle = ephemeralStyle.Foreground(lipgloss.Color("34")).Bold(true)
		}
		
		content.WriteString(fmt.Sprintf("[ ] %s\n", dynamicStyle.Render("dynamic - Persistent storage")))
		content.WriteString(fmt.Sprintf("[ ] %s\n", ephemeralStyle.Render("ephemeral - No persistent storage")))
		
		if !f.persistenceToggle {
			content.WriteString("\n[x] dynamic")
		} else {
			content.WriteString("\n[x] ephemeral")
		}
		
	case StepStorageClass:
		content.WriteString("Step 8/9: Storage Class\n\n")
		content.WriteString("Storage class name (leave empty for default):\n")
		content.WriteString(f.inputs[0].View() + "\n")
		
	case StepConfirm:
		totalSteps := "8/8"
		if f.persistence == "dynamic" {
			totalSteps = "9/9"
		}
		content.WriteString(fmt.Sprintf("Step %s: Confirm\n\n", totalSteps))
		content.WriteString("Review your cluster configuration:\n\n")
		
		content.WriteString(fmt.Sprintf("Name: %s\n", f.name))
		content.WriteString(fmt.Sprintf("Namespace: %s\n", f.namespace))
		content.WriteString(fmt.Sprintf("Mode: %s\n", f.mode))
		if f.version != "" {
			content.WriteString(fmt.Sprintf("Version: %s\n", f.version))
		}
		content.WriteString(fmt.Sprintf("Servers: %d\n", f.servers))
		if f.mode != "shared" {
			content.WriteString(fmt.Sprintf("Agents: %d\n", f.agents))
		}
		content.WriteString(fmt.Sprintf("Persistence: %s\n", f.persistence))
		if f.persistence == "dynamic" && f.storageClass != "" {
			content.WriteString(fmt.Sprintf("Storage Class: %s\n", f.storageClass))
		}
		
		content.WriteString("\nPress Enter to create the cluster, or Esc to cancel.\n")
		
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
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(80)

	return style.Render(content.String())
}