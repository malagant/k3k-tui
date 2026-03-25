# Contributing to k3k-tui

Thank you for your interest in contributing to k3k-tui! This document provides guidelines and information for contributors.

## Development Setup

### Prerequisites
- Go 1.22 or later
- A Kubernetes cluster with k3k CRDs installed
- kubectl configured with appropriate permissions

### Setting up the development environment
1. Clone the repository:
   ```bash
   git clone https://github.com/malagant/k3k-tui.git
   cd k3k-tui
   ```

2. Install dependencies:
   ```bash
   make deps
   ```

3. Build the application:
   ```bash
   make build
   ```

4. Run tests:
   ```bash
   make test
   ```

### Development Workflow

1. Create a new branch for your feature/fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and ensure tests pass:
   ```bash
   make test
   make lint
   ```

3. Build and test the application:
   ```bash
   make build
   ./k3k-tui --version
   ```

4. Commit your changes with a descriptive message:
   ```bash
   git commit -m "Add feature: description of your changes"
   ```

5. Push to your fork and create a pull request.

## Code Guidelines

### Go Code Style
- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions focused and single-purpose

### TUI Guidelines
- Follow Bubble Tea patterns for state management
- Use lipgloss for consistent styling
- Ensure keyboard navigation is intuitive
- Handle window resizing gracefully

### Project Structure
```
├── main.go              # Application entry point
├── internal/
│   ├── tui/             # TUI components and views
│   ├── k8s/             # Kubernetes client operations
│   └── types/           # k3k type definitions
└── examples/            # Example cluster configurations
```

## Testing

### Running Tests
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Test specific package
go test ./internal/tui/
```

### Testing with a k3k Cluster
1. Ensure you have k3k installed in your cluster
2. Create a test cluster:
   ```bash
   make init-test-cluster
   ```
3. Run the TUI:
   ```bash
   make demo
   ```

## Adding New Features

### Adding a New View
1. Add the view state to `ViewState` enum in `model.go`
2. Add the view handler to `Update()` method
3. Add the view renderer to `View()` method
4. Add keyboard shortcuts and help text

### Adding New Cluster Operations
1. Add the operation to `internal/k8s/client.go`
2. Add corresponding message types in `internal/tui/messages.go`
3. Add UI components in the appropriate view files
4. Update help text and keyboard shortcuts

### Modifying Forms
Form logic is in `create_form.go` and `edit_form.go`. When adding new fields:
1. Add the field to the form struct
2. Add input handling in `Update()` method
3. Add the field to the form view
4. Update the `ToCluster()` method

## Pull Request Process

1. Ensure your code builds without warnings
2. Add tests for new functionality
3. Update documentation if needed
4. Ensure all tests pass
5. Update CHANGELOG.md with your changes
6. Create a pull request with a clear description

### Pull Request Template
```markdown
## Description
Brief description of the changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] All tests pass
- [ ] Manual testing completed
- [ ] New tests added (if applicable)

## Screenshots (if applicable)
Include screenshots for UI changes
```

## Reporting Issues

When reporting bugs or requesting features, please include:

### Bug Reports
- Go version
- Operating system
- Kubernetes version and k3k version
- Steps to reproduce
- Expected vs actual behavior
- Terminal output/logs

### Feature Requests
- Use case description
- Proposed solution
- Alternative solutions considered
- Additional context

## Code of Conduct

- Be respectful and inclusive
- Focus on the code and ideas, not the person
- Accept constructive feedback gracefully
- Help others learn and grow

## Getting Help

- Check existing issues and discussions
- Read the documentation and examples
- Ask questions in GitHub discussions
- Contact maintainers for complex issues

## License

By contributing to k3k-tui, you agree that your contributions will be licensed under the MIT License.