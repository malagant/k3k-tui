package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/malagant/k3k-tui/internal/types"
)

var (
	k3kGVR = schema.GroupVersionResource{
		Group:    "k3k.io",
		Version:  "v1beta1",
		Resource: "clusters",
	}
)

// Client wraps Kubernetes clients for k3k operations
type Client struct {
	dynamic    dynamic.Interface
	clientset  kubernetes.Interface
	restConfig *rest.Config
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfigPath, context string) (*Client, error) {
	config, err := buildConfig(kubeconfigPath, context)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Client{
		dynamic:    dynamicClient,
		clientset:  clientset,
		restConfig: config,
	}, nil
}

// buildConfig constructs a Kubernetes client configuration
func buildConfig(kubeconfigPath, context string) (*rest.Config, error) {
	if kubeconfigPath == "" {
		if home := homeDir(); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	// Try in-cluster config first
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}

	// Fall back to kubeconfig
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{CurrentContext: context},
	).ClientConfig()
}

// homeDir returns the home directory for the current user
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// ListClusters retrieves all k3k clusters
func (c *Client) ListClusters(ctx context.Context, namespace string) (*types.ClusterList, error) {
	var opts metav1.ListOptions
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	unstructuredList, err := c.dynamic.Resource(k3kGVR).Namespace(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	clusterList := &types.ClusterList{}
	for _, item := range unstructuredList.Items {
		cluster := types.Cluster{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &cluster); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to cluster: %w", err)
		}
		clusterList.Items = append(clusterList.Items, cluster)
	}

	return clusterList, nil
}

// GetCluster retrieves a specific k3k cluster
func (c *Client) GetCluster(ctx context.Context, namespace, name string) (*types.Cluster, error) {
	unstructuredObj, err := c.dynamic.Resource(k3kGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster %s/%s: %w", namespace, name, err)
	}

	cluster := &types.Cluster{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, cluster); err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to cluster: %w", err)
	}

	return cluster, nil
}

// ensureNamespace creates the namespace if it doesn't exist
func (c *Client) ensureNamespace(ctx context.Context, namespace string) error {
	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil // already exists
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err = c.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
	}
	return nil
}

// CreateCluster creates a new k3k cluster, creating the namespace if needed
func (c *Client) CreateCluster(ctx context.Context, cluster *types.Cluster) (*types.Cluster, error) {
	// Ensure namespace exists
	if err := c.ensureNamespace(ctx, cluster.Namespace); err != nil {
		return nil, err
	}

	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to convert cluster to unstructured: %w", err)
	}

	unstructuredCluster := &unstructured.Unstructured{Object: unstructuredObj}
	unstructuredCluster.SetAPIVersion(types.APIVersion)
	unstructuredCluster.SetKind(types.Kind)

	created, err := c.dynamic.Resource(k3kGVR).Namespace(cluster.Namespace).Create(ctx, unstructuredCluster, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	createdCluster := &types.Cluster{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(created.Object, createdCluster); err != nil {
		return nil, fmt.Errorf("failed to convert created cluster: %w", err)
	}

	return createdCluster, nil
}

// UpdateCluster updates an existing k3k cluster
func (c *Client) UpdateCluster(ctx context.Context, cluster *types.Cluster) (*types.Cluster, error) {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to convert cluster to unstructured: %w", err)
	}

	unstructuredCluster := &unstructured.Unstructured{Object: unstructuredObj}

	updated, err := c.dynamic.Resource(k3kGVR).Namespace(cluster.Namespace).Update(ctx, unstructuredCluster, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update cluster: %w", err)
	}

	updatedCluster := &types.Cluster{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(updated.Object, updatedCluster); err != nil {
		return nil, fmt.Errorf("failed to convert updated cluster: %w", err)
	}

	return updatedCluster, nil
}

// DeleteCluster deletes a k3k cluster
func (c *Client) DeleteCluster(ctx context.Context, namespace, name string) error {
	err := c.dynamic.Resource(k3kGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete cluster %s/%s: %w", namespace, name, err)
	}
	return nil
}

// GetClusterEvents retrieves events for a specific cluster
func (c *Client) GetClusterEvents(ctx context.Context, namespace, name string) (*corev1.EventList, error) {
	events, err := c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Cluster", name),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get events for cluster %s/%s: %w", namespace, name, err)
	}
	return events, nil
}

// GetClusterPods retrieves pods related to a specific cluster
func (c *Client) GetClusterPods(ctx context.Context, namespace, clusterName string) (*corev1.PodList, error) {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("k3k.io/cluster=%s", clusterName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pods for cluster %s/%s: %w", namespace, clusterName, err)
	}
	return pods, nil
}

// GetKubeconfig retrieves the kubeconfig for a k3k cluster.
// k3k stores the server CA in a secret named "<cluster>-server-ca" and the
// API server is exposed via a service. We look for common secret patterns
// and build connection info from available data.
func (c *Client) GetKubeconfig(ctx context.Context, namespace, clusterName string) ([]byte, error) {
	// Try common kubeconfig secret name patterns used by k3k
	secretPatterns := []struct {
		name string
		keys []string
	}{
		{clusterName + "-kubeconfig", []string{"config", "kubeconfig", "value"}},
		{clusterName + "-kubeconfig", []string{"kubeconfig.yaml"}},
		{"k3k-" + clusterName + "-kubeconfig", []string{"config", "kubeconfig", "value"}},
	}

	for _, p := range secretPatterns {
		secret, err := c.clientset.CoreV1().Secrets(namespace).Get(ctx, p.name, metav1.GetOptions{})
		if err != nil {
			continue
		}
		for _, key := range p.keys {
			if data, ok := secret.Data[key]; ok && len(data) > 0 {
				return data, nil
			}
		}
	}

	// Fallback: list all secrets in namespace and look for kubeconfig-related data
	secrets, err := c.clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets in %s: %w", namespace, err)
	}

	for _, secret := range secrets.Items {
		for key, data := range secret.Data {
			if (key == "config" || key == "kubeconfig" || key == "kubeconfig.yaml" || key == "value") &&
				len(data) > 100 && containsKubeconfigMarker(data) {
				return data, nil
			}
		}
	}

	// Build info from available k3k secrets (server-ca, service)
	var info strings.Builder
	info.WriteString(fmt.Sprintf("# Kubeconfig for k3k cluster %s/%s\n", namespace, clusterName))
	info.WriteString("# No pre-generated kubeconfig secret found.\n")
	info.WriteString("# Use k3kcli to generate one:\n")
	info.WriteString(fmt.Sprintf("#   k3kcli kubeconfig generate --name %s --namespace %s\n\n", clusterName, namespace))

	// Show available secrets for debugging
	info.WriteString("# Available secrets in namespace:\n")
	for _, secret := range secrets.Items {
		info.WriteString(fmt.Sprintf("#   %s (type: %s, keys: %s)\n", secret.Name, secret.Type, secretKeys(&secret)))
	}

	// Show service endpoint if available
	svcName := clusterName + "-server"
	svc, err := c.clientset.CoreV1().Services(namespace).Get(ctx, svcName, metav1.GetOptions{})
	if err == nil {
		for _, port := range svc.Spec.Ports {
			if port.Name == "https" || port.Port == 6443 {
				info.WriteString(fmt.Sprintf("\n# API Server: %s.%s.svc:%d\n", svc.Name, namespace, port.Port))
			}
		}
	}

	return []byte(info.String()), nil
}

// containsKubeconfigMarker checks if data looks like a kubeconfig
func containsKubeconfigMarker(data []byte) bool {
	s := string(data)
	return strings.Contains(s, "apiVersion") &&
		(strings.Contains(s, "clusters:") || strings.Contains(s, "kind: Config"))
}

// secretKeys returns a comma-separated list of keys in a secret
func secretKeys(secret *corev1.Secret) string {
	keys := make([]string, 0, len(secret.Data))
	for k := range secret.Data {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

// ListNamespaces retrieves all namespaces
func (c *Client) ListNamespaces(ctx context.Context) (*corev1.NamespaceList, error) {
	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	return namespaces, nil
}

// ClusterToYAML converts a cluster to YAML representation
func ClusterToYAML(cluster *types.Cluster) (string, error) {
	// Convert to unstructured for proper YAML output
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cluster)
	if err != nil {
		return "", fmt.Errorf("failed to convert to unstructured: %w", err)
	}

	unstructured := &unstructured.Unstructured{Object: unstructuredObj}
	unstructured.SetAPIVersion(types.APIVersion)
	unstructured.SetKind(types.Kind)

	// Convert to JSON first, then we can format it nicely
	jsonData, err := json.MarshalIndent(unstructured.Object, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	return string(jsonData), nil
}

// Age calculates the age of a cluster
func Age(creationTime metav1.Time) string {
	duration := time.Since(creationTime.Time)
	
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	
	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}