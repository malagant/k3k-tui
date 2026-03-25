package types

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ClusterSpec defines the desired state of a k3k Cluster
type ClusterSpec struct {
	// Version is the K3s version (e.g., "v1.31.3-k3s1")
	Version string `json:"version,omitempty"`
	
	// Mode specifies the cluster mode: "shared" or "virtual"
	Mode string `json:"mode,omitempty"`
	
	// Servers is the number of server nodes (min 1, default 1)
	Servers *int32 `json:"servers,omitempty"`
	
	// Agents is the number of agent nodes (min 0, default 0, ignored in shared mode)
	Agents *int32 `json:"agents,omitempty"`
	
	// ClusterCIDR is the CIDR for the cluster network
	ClusterCIDR string `json:"clusterCIDR,omitempty"`
	
	// ServiceCIDR is the CIDR for services
	ServiceCIDR string `json:"serviceCIDR,omitempty"`
	
	// ClusterDNS is the DNS server IP
	ClusterDNS string `json:"clusterDNS,omitempty"`
	
	// Persistence defines storage configuration
	Persistence *PersistenceConfig `json:"persistence,omitempty"`
	
	// Expose defines how the cluster is exposed (mutually exclusive)
	Expose *ExposeConfig `json:"expose,omitempty"`
	
	// NodeSelector for pod placement
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	
	// TLSSANs are additional Subject Alternative Names for TLS certificates
	TLSSANs []string `json:"tlsSANs,omitempty"`
	
	// ServerArgs are additional arguments for K3s server
	ServerArgs []string `json:"serverArgs,omitempty"`
	
	// AgentArgs are additional arguments for K3s agent
	AgentArgs []string `json:"agentArgs,omitempty"`
	
	// ServerLimit defines resource limits for server pods
	ServerLimit corev1.ResourceList `json:"serverLimit,omitempty"`
	
	// WorkerLimit defines resource limits for worker pods
	WorkerLimit corev1.ResourceList `json:"workerLimit,omitempty"`
}

// PersistenceConfig defines storage configuration
type PersistenceConfig struct {
	Type               string `json:"type"` // "dynamic" or "ephemeral"
	StorageClassName   string `json:"storageClassName,omitempty"`
	StorageRequestSize string `json:"storageRequestSize,omitempty"`
}

// ExposeConfig defines how the cluster is exposed
type ExposeConfig struct {
	Ingress      *runtime.RawExtension `json:"ingress,omitempty"`
	LoadBalancer *runtime.RawExtension `json:"loadBalancer,omitempty"`
	NodePort     *runtime.RawExtension `json:"nodePort,omitempty"`
}

// ClusterStatus defines the observed state of a k3k Cluster
type ClusterStatus struct {
	// Phase represents the current phase of the cluster
	Phase string `json:"phase,omitempty"`
	
	// HostVersion is the version of the host cluster
	HostVersion string `json:"hostVersion,omitempty"`
	
	// ClusterCIDR is the actual CIDR being used
	ClusterCIDR string `json:"clusterCIDR,omitempty"`
	
	// ServiceCIDR is the actual service CIDR being used
	ServiceCIDR string `json:"serviceCIDR,omitempty"`
	
	// ClusterDNS is the actual DNS server IP being used
	ClusterDNS string `json:"clusterDNS,omitempty"`
	
	// PolicyName is the name of the network policy
	PolicyName string `json:"policyName,omitempty"`
	
	// Conditions represents the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Cluster represents a k3k cluster resource
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	
	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// ClusterList contains a list of Cluster resources
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

const (
	// API constants
	APIVersion = "k3k.io/v1beta1"
	Kind       = "Cluster"
	Resource   = "clusters"
	
	// Default values
	DefaultMode              = "shared"
	DefaultServers           = 1
	DefaultAgents            = 0
	DefaultSharedClusterCIDR = "10.42.0.0/16"
	DefaultVirtualClusterCIDR = "10.52.0.0/16"
	DefaultSharedServiceCIDR = "10.43.0.0/16"
	DefaultVirtualServiceCIDR = "10.53.0.0/16"
	DefaultClusterDNS        = "10.43.0.10"
	DefaultStorageSize       = "2G"
	
	// Cluster phases
	PhaseRunning      = "Running"
	PhaseProvisioning = "Provisioning"
	PhaseFailed       = "Failed"
	PhaseDeleting     = "Deleting"
)