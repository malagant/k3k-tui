package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/malagant/k3k-tui/internal/k8s"
	"github.com/malagant/k3k-tui/internal/tui"
)

var (
	version    = "v0.1.0"
	buildTime  = "unknown"
	commitHash = "unknown"
)

func main() {
	var kubeconfig = flag.String("kubeconfig", "", "Path to kubeconfig file (defaults to ~/.kube/config)")
	var context = flag.String("context", "", "Kubernetes context to use")
	var showVersion = flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("k3k-tui %s\n", version)
		fmt.Printf("Build time: %s\n", buildTime)
		fmt.Printf("Commit: %s\n", commitHash)
		os.Exit(0)
	}

	// Initialize Kubernetes client
	client, err := k8s.NewClient(*kubeconfig, *context)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Initialize TUI
	model := tui.NewModel(client, version)
	
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}