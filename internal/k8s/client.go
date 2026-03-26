package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

// GetKubeconfig retrieves the kubeconfig for a k3k virtual cluster.
// It reads the kubeconfig from the k3s server pod and rewrites the server
// endpoint to point to the cluster's Service (accessible from the host).
func (c *Client) GetKubeconfig(ctx context.Context, namespace, clusterName string) ([]byte, error) {
	// 1. Find the server pod for this cluster
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("cluster=%s,role=server", clusterName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list server pods: %w", err)
	}

	// Try alternative label selectors if first one didn't work
	if len(pods.Items) == 0 {
		pods, err = c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("k3k.io/cluster=%s", clusterName),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list server pods: %w", err)
		}
	}

	// Last resort: find pods matching the cluster name pattern
	if len(pods.Items) == 0 {
		allPods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list pods: %w", err)
		}
		for _, pod := range allPods.Items {
			if strings.Contains(pod.Name, clusterName) && strings.Contains(pod.Name, "server") {
				pods.Items = append(pods.Items, pod)
			}
		}
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no server pods found for cluster %s/%s", namespace, clusterName)
	}

	// Find a running server pod
	var serverPod *corev1.Pod
	for i := range pods.Items {
		if pods.Items[i].Status.Phase == corev1.PodRunning {
			serverPod = &pods.Items[i]
			break
		}
	}
	if serverPod == nil {
		serverPod = &pods.Items[0] // try first pod even if not running
	}

	// 2. Exec into the pod to read the kubeconfig
	kubeconfigData, err := c.execInPod(ctx, namespace, serverPod.Name, "cat /etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig from pod %s: %w", serverPod.Name, err)
	}

	// 3. Find the service endpoint for this cluster
	serverEndpoint := fmt.Sprintf("https://%s-server.%s.svc:6443", clusterName, namespace)

	// Try to find the actual service
	svcNames := []string{
		clusterName + "-server",
		clusterName + "-k3k-server",
		"k3k-" + clusterName + "-server",
	}
	for _, svcName := range svcNames {
		svc, err := c.clientset.CoreV1().Services(namespace).Get(ctx, svcName, metav1.GetOptions{})
		if err == nil {
			for _, port := range svc.Spec.Ports {
				if port.Port == 6443 || port.Name == "https" || port.Name == "api" {
					serverEndpoint = fmt.Sprintf("https://%s.%s.svc:%d", svc.Name, namespace, port.Port)
					break
				}
			}
			break
		}
	}

	// 4. Rewrite the server URL in the kubeconfig
	kubeconfig := strings.ReplaceAll(string(kubeconfigData), "https://127.0.0.1:6443", serverEndpoint)
	kubeconfig = strings.ReplaceAll(kubeconfig, "https://localhost:6443", serverEndpoint)

	return []byte(kubeconfig), nil
}

// execInPod executes a command in a pod via kubectl and returns stdout
func (c *Client) execInPod(ctx context.Context, namespace, podName, command string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "exec", "-n", namespace, podName, "--", "sh", "-c", command)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl exec failed: %w", err)
	}
	return out, nil
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