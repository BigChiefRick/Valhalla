# Valhalla Project Structure

This document outlines the project structure and development guidelines for Valhalla.

## Directory Structure

```
valhalla/
├── cmd/                           # CLI command implementations
│   ├── auth.go                   # Authentication command
│   ├── discover.go               # Discovery command
│   ├── generate.go               # IaC generation command
│   └── validate.go               # Validation command
├── internal/                     # Internal packages (not for external use)
│   ├── config/                   # Configuration management
│   │   └── config.go
│   ├── discovery/                # Infrastructure discovery engine
│   │   ├── engine.go            # Main discovery engine
│   │   └── providers/           # Provider implementations
│   │       ├── interfaces.go    # Provider interfaces
│   │       ├── vmware.go        # VMware vSphere provider
│   │       ├── proxmox.go       # Proxmox provider (to be implemented)
│   │       └── nutanix.go       # Nutanix provider (to be implemented)
│   ├── generators/              # IaC generators
│   │   ├── generator.go         # Generator interface and factory
│   │   ├── terraform.go         # Terraform generator (to be implemented)
│   │   ├── pulumi.go           # Pulumi generator (to be implemented)
│   │   └── ansible.go          # Ansible generator (to be implemented)
│   ├── logger/                  # Structured logging
│   │   └── logger.go
│   ├── models/                  # Data models
│   │   └── infrastructure.go    # Infrastructure data structures
│   ├── output/                  # Output formatting
│   │   └── formatter.go
│   └── validation/              # Template validation
│       └── validator.go
├── docs/                        # Documentation
├── examples/                    # Example configurations and outputs
├── scripts/                     # Build and development scripts
├── .gitignore                   # Git ignore rules
├── Dockerfile                   # Container build
├── LICENSE                      # MIT license
├── Makefile                     # Build configuration
├── README.md                    # Project documentation
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
└── main.go                      # Application entry point
```

## Development Workflow

### Prerequisites

- Go 1.21 or higher
- Make
- Docker (optional)
- Access to hypervisor environments for testing

### Getting Started

1. **Clone the repository**:
   ```bash
   git clone https://github.com/BigChiefRick/valhalla.git
   cd valhalla
   ```

2. **Install dependencies**:
   ```bash
   make deps
   ```

3. **Build the application**:
   ```bash
   make build
   ```

4. **Run tests**:
   ```bash
   make test
   ```

5. **Set up development environment**:
   ```bash
   make dev-setup
   ```

### Development Commands

| Command | Description |
|---------|-------------|
| `make build` | Build binary for current platform |
| `make build-all` | Build for all supported platforms |
| `make test` | Run tests |
| `make test-coverage` | Run tests with coverage |
| `make fmt` | Format code |
| `make lint` | Run linter |
| `make dev` | Run with live reload |
| `make clean` | Clean build artifacts |

### Code Organization

#### Commands (`cmd/`)
- Each command is in its own file
- Commands use Cobra for CLI functionality
- Commands should be focused and single-purpose
- Use structured logging throughout

#### Internal Packages (`internal/`)
- **config**: Configuration management with Viper
- **discovery**: Core discovery engine and provider implementations
- **generators**: IaC template generation
- **logger**: Structured logging with slog
- **models**: Data structures for infrastructure representation
- **output**: Output formatting (table, JSON, YAML, CSV)
- **validation**: Template and configuration validation

#### Provider Implementation
Providers must implement the provider interfaces defined in `internal/discovery/providers/interfaces.go`:

- `VMwareProvider` - VMware vSphere discovery
- `ProxmoxProvider` - Proxmox VE discovery
- `NutanixProvider` - Nutanix discovery

Each provider should:
- Handle connection management
- Implement discovery methods for all resource types
- Provide proper error handling and logging
- Support filtering and configuration options

#### Generator Implementation
Generators must implement the `Generator` interface in `internal/generators/generator.go`:

- Support multiple output formats
- Generate syntactically correct templates
- Include validation capabilities
- Support modular output structures

### Configuration

Valhalla uses a hierarchical configuration system:

1. **Command-line flags** (highest priority)
2. **Environment variables** 
3. **Configuration file** (lowest priority)

#### Configuration File Example
```yaml
debug: false
log_format: text
providers:
  vmware:
    server: vcenter.example.com
    username: administrator@vsphere.local
    insecure: true
    datacenter: "Production DC"
  proxmox:
    server: proxmox.example.com
    username: root@pam
    insecure: true
  nutanix:
    server: prism.example.com
    username: admin
    port: 9440
    insecure: true
output:
  format: table
  directory: ./output
```

#### Environment Variables
- `VALHALLA_DEBUG=true` - Enable debug logging
- `VSPHERE_SERVER=vcenter.example.com` - VMware vCenter server
- `VSPHERE_USER=admin` - VMware username
- `VSPHERE_PASSWORD=password` - VMware password
- `PROXMOX_SERVER=proxmox.example.com` - Proxmox server
- `PROXMOX_USER=root@pam` - Proxmox username
- `PROXMOX_PASSWORD=password` - Proxmox password
- `NUTANIX_SERVER=prism.example.com` - Nutanix server
- `NUTANIX_USER=admin` - Nutanix username
- `NUTANIX_PASSWORD=password` - Nutanix password

### Testing Strategy

#### Unit Tests
- Test individual functions and methods
- Mock external dependencies
- Test error conditions
- Aim for high coverage

#### Integration Tests
- Test provider connections
- Test end-to-end workflows
- Use test environments when possible
- Test with real hypervisor APIs

#### Test Commands
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test package
go test ./internal/discovery/providers -v

# Run benchmarks
make bench
```

### Security Considerations

#### Credential Management
- Never log credentials
- Support multiple authentication methods
- Use secure credential storage
- Implement credential testing

#### Network Security
- Support TLS verification
- Allow insecure connections for testing
- Implement connection timeouts
- Handle network failures gracefully

#### Code Security
- Regular dependency updates
- Vulnerability scanning with `make security`
- Input validation and sanitization
- Secure defaults

### Release Process

#### Version Tagging
```bash
# Create and push a new tag
git tag v1.0.0
git push origin v1.0.0
```

#### Building Releases
```bash
# Build all platforms
make build-all VERSION=1.0.0

# Create release archives
make release VERSION=1.0.0
```

#### Docker Images
```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run
```

### Contributing Guidelines

#### Code Style
- Use `gofmt` for formatting
- Follow Go naming conventions
- Add comments for exported functions
- Use structured logging

#### Pull Request Process
1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Run `make lint` and `make test`
5. Submit pull request
6. Address review feedback

#### Commit Messages
Follow conventional commit format:
- `feat: add Proxmox provider support`
- `fix: handle connection timeouts`
- `docs: update configuration examples`
- `test: add unit tests for discovery engine`

### Troubleshooting

#### Common Issues

**Build Failures**:
- Ensure Go 1.21+ is installed
- Run `make deps` to update dependencies
- Check for syntax errors with `make lint`

**Connection Issues**:
- Verify credentials with `valhalla auth <provider> --test`
- Check network connectivity
- Validate SSL certificates

**Discovery Issues**:
- Enable debug logging with `--debug`
- Check provider-specific configuration
- Verify permissions in hypervisor environment

#### Debug Mode
Enable detailed logging:
```bash
valhalla discover --provider vmware --debug
```

#### Log Analysis
Logs include structured fields for easy analysis:
```bash
# Filter by provider
grep '"provider":"vmware"' valhalla.log

# Filter by operation
grep '"operation":"discovery"' valhalla.log
```

### Future Enhancements

#### Planned Features
- Additional hypervisor support (Hyper-V, KVM)
- Web interface for discovery and generation
- CI/CD pipeline integration
- Advanced template customization
- Resource dependency mapping
- Change detection and drift analysis

#### Architecture Improvements
- Plugin system for providers
- Distributed discovery for large environments
- Caching and incremental discovery
- Advanced filtering and querying
- Export to external systems

This structure provides a solid foundation for developing and maintaining Valhalla while ensuring code quality, security, and extensibility.
