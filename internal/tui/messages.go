package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/malagant/k3k-tui/internal/types"
)

// Message types for async operations

type clustersLoadedMsg struct {
	clusters []types.Cluster
	err      error
}

type clusterDetailLoadedMsg struct {
	cluster *types.Cluster
	err     error
}

type clusterCreatedMsg struct {
	cluster *types.Cluster
	err     error
}

type clusterUpdatedMsg struct {
	cluster *types.Cluster
	err     error
}

type clusterDeletedMsg struct {
	err error
}

type kubeconfigLoadedMsg struct {
	content string
	err     error
}

type k9sFinishedMsg struct {
	err error
}

type errorMsg struct {
	error string
}

// Commands for async operations

func (m Model) loadClusters() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		clusters, err := m.client.ListClusters(ctx, m.namespace)
		if err != nil {
			return clustersLoadedMsg{err: err}
		}
		return clustersLoadedMsg{clusters: clusters.Items}
	})
}

func (m Model) loadClusterDetail(namespace, name string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		cluster, err := m.client.GetCluster(ctx, namespace, name)
		if err != nil {
			return clusterDetailLoadedMsg{err: err}
		}
		return clusterDetailLoadedMsg{cluster: cluster}
	})
}

func (m Model) createCluster(cluster *types.Cluster) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		created, err := m.client.CreateCluster(ctx, cluster)
		if err != nil {
			return clusterCreatedMsg{err: err}
		}
		return clusterCreatedMsg{cluster: created}
	})
}

func (m Model) updateCluster(cluster *types.Cluster) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		updated, err := m.client.UpdateCluster(ctx, cluster)
		if err != nil {
			return clusterUpdatedMsg{err: err}
		}
		return clusterUpdatedMsg{cluster: updated}
	})
}

func (m Model) deleteCluster(namespace, name string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		err := m.client.DeleteCluster(ctx, namespace, name)
		return clusterDeletedMsg{err: err}
	})
}

func (m Model) loadKubeconfig(namespace, name string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		kubeconfig, err := m.client.GetKubeconfig(ctx, namespace, name)
		if err != nil {
			return kubeconfigLoadedMsg{err: err}
		}
		return kubeconfigLoadedMsg{content: string(kubeconfig)}
	})
}

// launchK9s writes the kubeconfig to a temp file and launches k9s with it.
// Uses tea.ExecProcess to suspend the TUI while k9s runs.
func (m Model) launchK9s(namespace, clusterName string) tea.Cmd {
	return func() tea.Msg {
		// First fetch the kubeconfig
		ctx := context.Background()
		kubeconfigData, err := m.client.GetKubeconfig(ctx, namespace, clusterName)
		if err != nil {
			return k9sFinishedMsg{err: fmt.Errorf("failed to get kubeconfig: %w", err)}
		}

		// Check if it's a real kubeconfig (not just info text)
		if len(kubeconfigData) < 100 || kubeconfigData[0] == '#' {
			return k9sFinishedMsg{err: fmt.Errorf("no kubeconfig available for %s/%s — use k3kcli to generate one first", namespace, clusterName)}
		}

		// Write to temp file
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("k3k-tui-%s-%s-*.yaml", namespace, clusterName))
		if err != nil {
			return k9sFinishedMsg{err: fmt.Errorf("failed to create temp file: %w", err)}
		}

		if _, err := tmpFile.Write(kubeconfigData); err != nil {
			os.Remove(tmpFile.Name())
			return k9sFinishedMsg{err: fmt.Errorf("failed to write kubeconfig: %w", err)}
		}
		tmpFile.Close()

		// Return an exec command that will suspend the TUI
		return k9sExecMsg{kubeconfigPath: tmpFile.Name()}
	}
}

// k9sExecMsg is an intermediate message to trigger tea.ExecProcess
type k9sExecMsg struct {
	kubeconfigPath string
}

// execK9s creates the tea.ExecProcess command
func execK9s(kubeconfigPath string) tea.Cmd {
	k9sPath, err := exec.LookPath("k9s")
	if err != nil {
		return func() tea.Msg {
			os.Remove(kubeconfigPath)
			return k9sFinishedMsg{err: fmt.Errorf("k9s not found in PATH — install it first")}
		}
	}

	c := exec.Command(k9sPath, "--kubeconfig", kubeconfigPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		os.Remove(kubeconfigPath) // cleanup temp file
		return k9sFinishedMsg{err: err}
	})
}

