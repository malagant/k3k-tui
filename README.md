# k3k-tui

A terminal user interface (TUI) for managing k3k (Kubernetes-in-Kubernetes) virtual clusters built with [Charmbracelet's Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

### Cluster Management
- **List View**: Browse all k3k clusters across namespaces with color-coded status
- **Create**: Interactive form to create new clusters with all configuration options
- **Edit**: Modify mutable cluster properties (servers, agents, version, args)
- **Delete**: Safe deletion with cluster name confirmation
- **Details**: View complete cluster specification and status
- **Kubeconfig**: Access and export cluster kubeconfig

### UI Features
- **Keyboard Navigation**: Full keyboard-driven interface
- **Filtering**: Search/filter clusters by name, namespace, mode, or status
- **Namespace Filtering**: Focus on specific namespaces
- **Color Coding**: Visual status indicators (green=Running, yellow=Provisioning, red=Failed)
- **Responsive Layout**: Adapts to terminal size

## Installation

### Prerequisites
- Go 1.22 or later
- Access to a Kubernetes cluster with k3k CRDs installed
- kubectl configured with appropriate permissions

### Build from Source
```bash
git clone https://github.com/malagant/k3k-tui.git
cd k3k-tui
go build -o k3k-tui .
```

## Usage

### Basic Usage
```bash
# Use default kubeconfig
./k3k-tui

# Specify kubeconfig file
./k3k-tui --kubeconfig /path/to/kubeconfig

# Use specific context
./k3k-tui --context my-context

# Show version
./k3k-tui --version
```

### Keyboard Shortcuts

#### Cluster List View
- `↑/↓` or `j/k`: Navigate clusters
- `c`: Create new cluster  
- `d` or `Enter`: View cluster details
- `e`: Edit cluster
- `x` or `Delete`: Delete cluster
- `k`: Get kubeconfig
- `/`: Filter clusters
- `n`: Namespace filter
- `r` or `F5`: Refresh
- `q` or `Ctrl+C`: Quit

#### Create/Edit Forms
- `Tab/Shift+Tab`: Navigate form fields
- `Space`: Toggle options (mode, persistence)
- `Enter`: Next step / Submit
- `Esc`: Cancel

#### Detail/Kubeconfig Views  
- `↑/↓`: Scroll content
- `Esc`: Return to list

#### Delete Confirmation
- Type cluster name exactly to confirm
- `Enter`: Delete (when confirmed)
- `Esc`: Cancel

## k3k Cluster Configuration

The TUI supports all k3k cluster configuration options:

### Basic Settings
- **Name**: Cluster identifier
- **Namespace**: Kubernetes namespace for the cluster
- **Mode**: `shared` (lightweight) or `virtual` (isolated)
- **Version**: K3s version (e.g., "v1.31.3-k3s1")

### Scaling
- **Servers**: Number of control plane nodes (min: 1)
- **Agents**: Number of worker nodes (min: 0, ignored in shared mode)

### Networking  
- **Cluster CIDR**: Pod network CIDR (defaults: 10.42.0.0/16 shared, 10.52.0.0/16 virtual)
- **Service CIDR**: Service network CIDR (defaults: 10.43.0.0/16 shared, 10.53.0.0/16 virtual)
- **Cluster DNS**: DNS server IP (default: 10.43.0.10)

### Storage
- **Persistence**: `dynamic` (persistent) or `ephemeral` (temporary)
- **Storage Class**: Kubernetes storage class for persistent volumes
- **Storage Size**: Volume size (default: 2G)

### Advanced Options
- **Node Selector**: Pod placement constraints
- **TLS SANs**: Additional certificate Subject Alternative Names
- **Server Args**: Additional K3s server arguments
- **Agent Args**: Additional K3s agent arguments
- **Resource Limits**: CPU/memory limits for server and worker pods

## API Reference

The TUI works with k3k CRDs:

- **API Group**: `k3k.io/v1beta1`
- **Kind**: `Cluster`
- **Resource**: `clusters`

Required RBAC permissions:
```yaml
- apiGroups: ["k3k.io"]
  resources: ["clusters"]
  verbs: ["get", "list", "create", "update", "delete"]
- apiGroups: [""]
  resources: ["secrets", "events", "pods", "namespaces"]
  verbs: ["get", "list"]
```

## Development

### Project Structure
```
├── main.go              # Application entry point
├── internal/
│   ├── tui/             # TUI components and views
│   │   ├── model.go     # Main application model
│   │   ├── views.go     # View handlers
│   │   ├── messages.go  # Async operations
│   │   ├── create_form.go # Cluster creation form
│   │   └── edit_form.go   # Cluster editing form
│   ├── k8s/             # Kubernetes client operations
│   │   └── client.go    # k3k cluster CRUD operations
│   └── types/           # k3k type definitions
│       ├── cluster.go   # Cluster types and constants
│       └── deepcopy.go  # Deep copy implementations
```

### Building
```bash
# Build
make build

# Cross-compile for different platforms
make build-all

# Clean build artifacts
make clean
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes and add tests
4. Submit a pull request

## Acknowledgments

- [Charmbracelet](https://github.com/charmbracelet) for the excellent TUI libraries
- [Rancher](https://github.com/rancher/k3k) for the k3k project
- [Kubernetes](https://kubernetes.io/) community for client-go