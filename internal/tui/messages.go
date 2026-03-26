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

// launchK9s launches k9s for the selected cluster.
// Strategy:
// 1. Try to find a real kubeconfig secret → launch k9s with it
// 2. Fallback: launch k9s with host kubeconfig, scoped to the cluster's namespace
func (m Model) launchK9s(namespace, clusterName string) tea.Cmd {
	return func() tea.Msg {
		// Check k9s is available
		k9sPath, err := exec.LookPath("k9s")
		if err != nil {
			return k9sFinishedMsg{err: fmt.Errorf("k9s not found in PATH")}
		}

		// Try to get a virtual cluster kubeconfig
		ctx := context.Background()
		kubeconfigData, err := m.client.GetKubeconfig(ctx, namespace, clusterName)

		if err == nil && len(kubeconfigData) > 100 && kubeconfigData[0] != '#' {
			// Got a real kubeconfig — write to temp file and use it
			tmpFile, err := os.CreateTemp("", fmt.Sprintf("k3k-tui-%s-%s-*.yaml", namespace, clusterName))
			if err == nil {
				tmpFile.Write(kubeconfigData)
				tmpFile.Close()
				return k9sExecMsg{
					k9sPath:        k9sPath,
					kubeconfigPath: tmpFile.Name(),
					namespace:      "",
					cleanup:        true,
				}
			}
		}

		// Fallback: use host kubeconfig, scoped to the cluster namespace
		return k9sExecMsg{
			k9sPath:   k9sPath,
			namespace: namespace,
			cleanup:   false,
		}
	}
}

// k9sExecMsg is an intermediate message to trigger tea.ExecProcess
type k9sExecMsg struct {
	k9sPath        string
	kubeconfigPath string // empty = use default/host kubeconfig
	namespace      string // if set, scope k9s to this namespace
	cleanup        bool   // whether to remove kubeconfigPath on exit
}

// execK9s creates the tea.ExecProcess command
func execK9s(msg k9sExecMsg) tea.Cmd {
	var args []string

	if msg.kubeconfigPath != "" {
		args = append(args, "--kubeconfig", msg.kubeconfigPath)
	}
	if msg.namespace != "" {
		args = append(args, "-n", msg.namespace)
	}

	c := exec.Command(msg.k9sPath, args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if msg.cleanup && msg.kubeconfigPath != "" {
			os.Remove(msg.kubeconfigPath)
		}
		return k9sFinishedMsg{err: err}
	})
}

