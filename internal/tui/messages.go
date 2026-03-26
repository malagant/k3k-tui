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

type tickMsg struct{}

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

// launchK9s fetches the virtual cluster kubeconfig from the server pod
// and launches k9s with it. Falls back to host namespace view.
func (m Model) launchK9s(namespace, clusterName string) tea.Cmd {
	k9sPath, err := exec.LookPath("k9s")
	if err != nil {
		return func() tea.Msg {
			return k9sFinishedMsg{err: fmt.Errorf("k9s not found in PATH")}
		}
	}

	// Fetch kubeconfig from virtual cluster server pod
	ctx := context.Background()
	kubeconfigData, err := m.client.GetKubeconfig(ctx, namespace, clusterName)
	if err == nil && len(kubeconfigData) > 50 {
		tmpFile, tmpErr := os.CreateTemp("", fmt.Sprintf("k3k-tui-%s-%s-*.yaml", namespace, clusterName))
		if tmpErr == nil {
			tmpFile.Write(kubeconfigData)
			tmpFile.Close()
			c := exec.Command(k9sPath, "--kubeconfig", tmpFile.Name())
			return tea.ExecProcess(c, func(err error) tea.Msg {
				os.Remove(tmpFile.Name())
				return k9sFinishedMsg{err: err}
			})
		}
	}

	// Fallback: host kubeconfig scoped to namespace
	c := exec.Command(k9sPath, "-n", namespace)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return k9sFinishedMsg{err: err}
	})
}

