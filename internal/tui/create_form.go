package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"


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

// CreateForm handles the cluster creation form with k9s modal styling
type CreateForm struct {
	step         FormStep
	inputs       []textinput.Model
	
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
	
	// UI dimensions
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
		width:        80,
		height:       25,
	}

	f.initInputs()
	return f
}

// initInputs initializes the text inputs with k9s styling
func (f *CreateForm) initInputs() {
	inputs := make([]textinput.Model, 5)

	inputStyle := lipgloss.NewStyle().
		Foreground(colorHeaderText).
		Background(colorBg)

	focusedStyle := lipgloss.NewStyle().
		Foreground(colorCommand).
		Background(colorBg)

	// Name input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Enter cluster name"
	inputs[0].Focus()
	inputs[0].CharLimit = 253
	inputs[0].Width = 30
	inputs[0].TextStyle = inputStyle
	inputs[0].PlaceholderStyle = lipgloss.NewStyle().Foreground(colorHelp)
	inputs[0].Cursor.Style = focusedStyle

	// Namespace input
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Enter namespace"
	inputs[1].CharLimit = 253
	inputs[1].Width = 30
	inputs[1].TextStyle = inputStyle
	inputs[1].PlaceholderStyle = lipgloss.NewStyle().Foreground(colorHelp)
	inputs[1].Cursor.Style = focusedStyle

	// Version input
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "e.g., v1.31.3-k3s1 (optional)"
	inputs[2].CharLimit = 50
	inputs[2].Width = 30
	inputs[2].TextStyle = inputStyle
	inputs[2].PlaceholderStyle = lipgloss.NewStyle().Foreground(colorHelp)
	inputs[2].Cursor.Style = focusedStyle

	// Servers input
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "1"
	inputs[3].SetValue("1")
	inputs[3].CharLimit = 3
	inputs[3].Width = 10
	inputs[3].TextStyle = inputStyle
	inputs[3].PlaceholderStyle = lipgloss.NewStyle().Foreground(colorHelp)
	inputs[3].Cursor.Style = focusedStyle

	// Agents input
	inputs[4] = textinput.New()
	inputs[4].Placeholder = "0"
	inputs[4].SetValue("0")
	inputs[4].CharLimit = 3
	inputs[4].Width = 10
	inputs[4].TextStyle = inputStyle
	inputs[4].PlaceholderStyle = lipgloss.NewStyle().Foreground(colorHelp)
	inputs[4].Cursor.Style = focusedStyle

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
		if msg.String() == " " || msg.String() == "space" {
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
		if msg.String() == " " || msg.String() == "space" {
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
			// Storage class input (reuse name input)
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
			f.inputs[0].Blur()
		}
	case StepNamespace:
		if f.namespace != "" {
			f.step = StepMode
			f.inputs[1].Blur()
		}
	case StepMode:
		f.step = StepVersion
		f.inputs[2].Focus()
	case StepVersion:
		f.step = StepServers
		f.inputs[2].Blur()
		f.inputs[3].Focus()
	case StepServers:
		f.step = StepAgents
		f.inputs[3].Blur()
		f.inputs[4].Focus()
	case StepAgents:
		f.step = StepPersistence
		f.inputs[4].Blur()
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
		f.inputs[0].Blur()
	}
}

// Previous moves to the previous form step
func (f *CreateForm) Previous() {
	switch f.step {
	case StepNamespace:
		f.step = StepName
		f.inputs[1].Blur()
		f.inputs[0].Focus()
	case StepMode:
		f.step = StepNamespace
		f.inputs[1].Focus()
	case StepVersion:
		f.step = StepMode
		f.inputs[2].Blur()
	case StepServers:
		f.step = StepVersion
		f.inputs[3].Blur()
		f.inputs[2].Focus()
	case StepAgents:
		f.step = StepServers
		f.inputs[4].Blur()
		f.inputs[3].Focus()
	case StepPersistence:
		f.step = StepAgents
		f.inputs[4].Focus()
	case StepStorageClass:
		f.step = StepPersistence
		f.inputs[0].Blur()
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

// View renders the form as a centered Catppuccin Mocha modal
func (f *CreateForm) View() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(catPeach).
		Bold(true).
		Align(lipgloss.Center).
		Width(68)

	content.WriteString(titleStyle.Render("✦ CREATE K3K CLUSTER") + "\n\n")

	// Progress bar
	totalSteps := 8
	if f.persistence == "dynamic" && f.step >= StepStorageClass {
		totalSteps = 9
	}
	currentStep := int(f.step) + 1
	if f.step == StepStorageClass {
		currentStep = 8
	} else if f.step == StepConfirm && f.persistence == "dynamic" {
		currentStep = 9
	}
	content.WriteString(f.renderProgressBar(currentStep, totalSteps) + "\n\n")

	descStyle := lipgloss.NewStyle().Foreground(catSubtext0)

	switch f.step {
	case StepName:
		content.WriteString(f.renderStepTitle("Cluster Name"))
		content.WriteString(descStyle.Render("Enter a unique name for your cluster:") + "\n\n")
		content.WriteString(f.inputs[0].View() + "\n")

	case StepNamespace:
		content.WriteString(f.renderStepTitle("Namespace"))
		content.WriteString(descStyle.Render("Namespace where the cluster will be created:") + "\n\n")
		content.WriteString(f.inputs[1].View() + "\n")

	case StepMode:
		content.WriteString(f.renderStepTitle("Cluster Mode"))
		content.WriteString(descStyle.Render("Select cluster mode (space to toggle):") + "\n\n")

		sharedIcon, virtualIcon := "○", "○"
		if !f.modeToggle {
			sharedIcon = lipgloss.NewStyle().Foreground(catGreen).Render("●")
		} else {
			virtualIcon = lipgloss.NewStyle().Foreground(catGreen).Render("●")
		}

		content.WriteString(fmt.Sprintf("%s %s  Lightweight, shared control plane\n",
			sharedIcon, lipgloss.NewStyle().Foreground(catTeal).Bold(true).Render("shared")))
		content.WriteString(fmt.Sprintf("%s %s  Full isolated virtual cluster\n",
			virtualIcon, lipgloss.NewStyle().Foreground(catMauve).Bold(true).Render("virtual")))

	case StepVersion:
		content.WriteString(f.renderStepTitle("K3s Version"))
		content.WriteString(descStyle.Render("K3s version (leave empty for host default):") + "\n\n")
		content.WriteString(f.inputs[2].View() + "\n")

	case StepServers:
		content.WriteString(f.renderStepTitle("Server Nodes"))
		content.WriteString(descStyle.Render("Number of server (control plane) nodes:") + "\n\n")
		content.WriteString(f.inputs[3].View() + "\n")

	case StepAgents:
		content.WriteString(f.renderStepTitle("Agent Nodes"))
		if f.mode == "shared" {
			content.WriteString(descStyle.Render("Ignored in shared mode (set to 0):") + "\n\n")
		} else {
			content.WriteString(descStyle.Render("Number of agent (worker) nodes:") + "\n\n")
		}
		content.WriteString(f.inputs[4].View() + "\n")

	case StepPersistence:
		content.WriteString(f.renderStepTitle("Persistence"))
		content.WriteString(descStyle.Render("Select persistence type (space to toggle):") + "\n\n")

		dynIcon, ephIcon := "○", "○"
		if !f.persistenceToggle {
			dynIcon = lipgloss.NewStyle().Foreground(catGreen).Render("●")
		} else {
			ephIcon = lipgloss.NewStyle().Foreground(catGreen).Render("●")
		}

		content.WriteString(fmt.Sprintf("%s %s  Persistent PVC storage\n",
			dynIcon, lipgloss.NewStyle().Foreground(catGreen).Bold(true).Render("dynamic")))
		content.WriteString(fmt.Sprintf("%s %s  No persistent storage\n",
			ephIcon, lipgloss.NewStyle().Foreground(catYellow).Bold(true).Render("ephemeral")))

	case StepStorageClass:
		content.WriteString(f.renderStepTitle("Storage Class"))
		content.WriteString(descStyle.Render("Storage class name (leave empty for default):") + "\n\n")
		content.WriteString(f.inputs[0].View() + "\n")

	case StepConfirm:
		content.WriteString(f.renderStepTitle("Confirm"))

		kS := lipgloss.NewStyle().Foreground(catBlue).Width(16)
		vS := lipgloss.NewStyle().Foreground(catText)

		content.WriteString(fmt.Sprintf("%s%s\n", kS.Render("Name:"), vS.Render(f.name)))
		content.WriteString(fmt.Sprintf("%s%s\n", kS.Render("Namespace:"), vS.Render(f.namespace)))

		mC := catTeal
		if f.mode == "virtual" {
			mC = catMauve
		}
		content.WriteString(fmt.Sprintf("%s%s\n", kS.Render("Mode:"), lipgloss.NewStyle().Foreground(mC).Render(f.mode)))

		if f.version != "" {
			content.WriteString(fmt.Sprintf("%s%s\n", kS.Render("Version:"), vS.Render(f.version)))
		}
		content.WriteString(fmt.Sprintf("%s%s\n", kS.Render("Servers:"), vS.Render(fmt.Sprintf("%d", f.servers))))
		if f.mode != "shared" {
			content.WriteString(fmt.Sprintf("%s%s\n", kS.Render("Agents:"), vS.Render(fmt.Sprintf("%d", f.agents))))
		}

		pC := catGreen
		if f.persistence == "ephemeral" {
			pC = catYellow
		}
		content.WriteString(fmt.Sprintf("%s%s\n", kS.Render("Persistence:"), lipgloss.NewStyle().Foreground(pC).Render(f.persistence)))

		if f.persistence == "dynamic" && f.storageClass != "" {
			content.WriteString(fmt.Sprintf("%s%s\n", kS.Render("Storage Class:"), vS.Render(f.storageClass)))
		}

		content.WriteString("\n" + lipgloss.NewStyle().Foreground(catGreen).Render("Press Enter to create the cluster") + "\n")
	}

	// Footer instructions
	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(catOverlay0).Render(
		"Tab: Next • Shift+Tab: Back • Enter: Continue • Esc: Cancel"))

	modalStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(catGreen).
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
func (f *CreateForm) renderStepTitle(title string) string {
	return lipgloss.NewStyle().
		Foreground(catBlue).
		Bold(true).
		Render(title) + "\n"
}

// renderProgressBar renders a Catppuccin-styled progress bar
func (f *CreateForm) renderProgressBar(current, total int) string {
	barWidth := 56
	filled := int(float64(current) / float64(total) * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	doneStyle := lipgloss.NewStyle().Foreground(catGreen)
	todoStyle := lipgloss.NewStyle().Foreground(catSurface1)
	labelStyle := lipgloss.NewStyle().Foreground(catOverlay1)

	bar := doneStyle.Render(strings.Repeat("━", filled)) +
		todoStyle.Render(strings.Repeat("━", barWidth-filled))

	return bar + " " + labelStyle.Render(fmt.Sprintf("%d/%d", current, total))
}