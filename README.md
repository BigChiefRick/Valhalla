# Valhalla

<div align="center">
  <img src="docs/images/valhalla-logo.png" alt="Valhalla Logo" width="300"/>
  
  **Hypervisor Infrastructure Discovery and IaC Generation Tool**
  
  *The eternal hall where discovered infrastructure gains immortality through IaC transformation*
</div>

<div align="center">

[![Go 1.21+](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/dl/)
[![VMware vSphere](https://img.shields.io/badge/VMware-vSphere-blue.svg)]()
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-green.svg)]()

</div>

## ⚡ What Valhalla Does

Valhalla bridges the gap between existing hypervisor infrastructure and modern Infrastructure as Code practices. Transform your VMware vSphere, Proxmox, and Nutanix environments into battle-tested IaC templates for disaster recovery and infrastructure management.

- **🔍 Discover Hypervisor Infrastructure** - Connect to VMware vSphere environments to catalog VMs, networks, and storage
- **⚔️ Transform to IaC Warriors** - Convert discovered infrastructure into production-ready Infrastructure as Code
- **🏰 Multiple IaC Formats** - Generate Terraform, Pulumi, and Ansible templates
- **🌉 Disaster Recovery Ready** - Create deployable templates for infrastructure recreation

## 🎯 Why Valhalla Exists

**The Problem**: Organizations running VMware vSphere and other hypervisor infrastructure struggle to maintain Infrastructure as Code practices. Manual documentation becomes outdated, and disaster recovery planning lacks automation.

**The Solution**: Valhalla automatically discovers your existing hypervisor infrastructure and generates current, deployable Infrastructure as Code templates - perfect for disaster recovery, infrastructure migration, and compliance documentation.

## ✅ Current Status

**Production Ready Features:**
- ✅ **VMware vSphere Discovery** - Full VM, network, and storage discovery
- ✅ **Terraform Generation** - Complete HCL templates with data sources and variables
- ✅ **Pulumi Generation** - Python and TypeScript program generation
- ✅ **Ansible Generation** - Complete playbooks for infrastructure recreation
- ✅ **Multiple Output Formats** - Table, JSON, YAML, CSV for discovered data
- ✅ **Secure Authentication** - Environment variables and credential management

**In Development:**
- 🔧 **Proxmox Support** - Provider interface ready, implementation in progress
- 🔧 **Nutanix Support** - Provider interface ready, implementation in progress

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- Access to VMware vCenter environment
- Git

### Installation

```bash
# Clone the repository
git clone https://github.com/BigChiefRick/valhalla.git
cd valhalla

# Build the application
make deps
make build

# Verify installation
./bin/valhalla --help
```

### Docker Installation

```bash
# Build Docker image
make docker-build

# Run with Docker
docker run --rm -it valhalla:latest --help
```

## 🔧 Configuration

### Environment Variables

```bash
# VMware vSphere
export VSPHERE_SERVER="vcenter.example.com"
export VSPHERE_USER="administrator@vsphere.local"
export VSPHERE_PASSWORD="your-password"

# Proxmox (Coming Soon)
export PROXMOX_SERVER="proxmox.example.com"
export PROXMOX_USER="root@pam"
export PROXMOX_PASSWORD="your-password"

# Nutanix (Coming Soon)
export NUTANIX_SERVER="prism.example.com"
export NUTANIX_USER="admin"
export NUTANIX_PASSWORD="your-password"
```

### Configuration File

Create `~/.valhalla.yaml`:

```yaml
debug: false
log_format: text

providers:
  vmware:
    server: "vcenter.example.com"
    username: "administrator@vsphere.local"
    insecure: true
    datacenter: "Production DC"
    cluster: "Production Cluster"

output:
  format: table
  directory: ./output
```

## 📖 Usage Examples

### 1. Discover VMware Infrastructure

```bash
# Interactive authentication setup
./bin/valhalla auth vmware --server vcenter.example.com

# Discover infrastructure (dry run first)
./bin/valhalla discover --provider vmware --dry-run

# Full discovery with output to file
./bin/valhalla discover --provider vmware \
  --datacenter "Production DC" \
  --output-file infrastructure.json

# Table format output
./bin/valhalla discover --provider vmware \
  --datacenter "Production DC" \
  --format table
```

### 2. Generate Infrastructure as Code

```bash
# Generate Terraform templates
./bin/valhalla generate \
  --input infrastructure.json \
  --format terraform \
  --output-dir ./terraform

# Generate Pulumi Python program
./bin/valhalla generate \
  --input infrastructure.json \
  --format pulumi-python \
  --output-dir ./pulumi

# Generate Ansible playbooks
./bin/valhalla generate \
  --input infrastructure.json \
  --format ansible \
  --output-dir ./ansible
```

### 3. Validate Generated Templates

```bash
# Validate Terraform files
./bin/valhalla validate --path ./terraform --format terraform

# Validate all files recursively
./bin/valhalla validate --path ./output --recursive
```

## 🏗️ Generated IaC Structure

### Terraform Output
```
terraform/
├── provider.tf        # VMware provider configuration
├── variables.tf       # Input variables with defaults
├── data.tf           # Data sources for existing resources
├── virtual_machines.tf # VM resource definitions
└── outputs.tf        # Output values for created resources
```

### Pulumi Output
```
pulumi/
├── Pulumi.yaml       # Project configuration
├── requirements.txt  # Python dependencies
├── __main__.py       # Main program (Python)
└── package.json      # Node.js dependencies (TypeScript)
```

### Ansible Output
```
ansible/
├── site.yml          # Main playbook
├── inventory.yml     # Discovered hosts inventory
├── group_vars/       # Variables and mappings
├── tasks/            # Provider-specific tasks
└── requirements.yml  # Ansible collections
```

## 🔐 Authentication

### VMware vSphere

```bash
# Interactive setup
./bin/valhalla auth vmware --server vcenter.example.com

# Test existing credentials
./bin/valhalla auth vmware --test

# Environment variables
export VSPHERE_SERVER="vcenter.example.com"
export VSPHERE_USER="administrator@vsphere.local"
export VSPHERE_PASSWORD="password"
```

### Security Best Practices

- Use environment variables for credentials
- Never commit passwords to version control
- Test with non-production environments first
- Use service accounts with minimal required permissions

## 🛠️ Development

### Build from Source

```bash
# Development setup
make dev-setup

# Run with live reload
make dev

# Run tests
make test

# Build for all platforms
make build-all
```

### Available Make Targets

```bash
make build        # Build for current platform
make build-all    # Build for all platforms
make test         # Run tests
make test-coverage # Run tests with coverage
make lint         # Run linter
make clean        # Clean build artifacts
make deps         # Download dependencies
make dev          # Run with live reload
make docker-build # Build Docker image
```

## 📊 Example Output

### Discovery Results (Table Format)
```
=== VMWARE Infrastructure (vcenter.example.com) ===
Datacenter: Production DC
Cluster: Production Cluster
Discovery Time: 2025-06-25 17:00:00

Virtual Machines:
┌─────────────────┬───────────┬─────┬─────────────┬──────────────────┬──────────────┐
│ NAME            │ STATE     │ CPU │ MEMORY (MB) │ OS               │ HOST         │
├─────────────────┼───────────┼─────┼─────────────┼──────────────────┼──────────────┤
│ web-server-01   │ poweredOn │ 4   │ 8192        │ Ubuntu 20.04 LTS │ esxi-host-01 │
│ db-server-01    │ poweredOn │ 8   │ 16384       │ CentOS 7         │ esxi-host-02 │
│ app-server-01   │ poweredOn │ 2   │ 4096        │ Windows Server   │ esxi-host-01 │
└─────────────────┴───────────┴─────┴─────────────┴──────────────────┴──────────────┘

Networks:
┌──────────────┬─────────────┬──────┬──────────┬──────┐
│ NAME         │ TYPE        │ VLAN │ VSWITCH  │ DHCP │
├──────────────┼─────────────┼──────┼──────────┼──────┤
│ VM Network   │ standard    │ N/A  │ vSwitch0 │ No   │
│ DMZ Network  │ distributed │ 100  │ N/A      │ Yes  │
└──────────────┴─────────────┴──────┴──────────┴──────┘

Total Resources: 15
```

### Generated Terraform Example
```hcl
resource "vsphere_virtual_machine" "web_server_01" {
  name             = "web-server-01"
  resource_pool_id = data.vsphere_compute_cluster.cluster.resource_pool_id
  datastore_id     = data.vsphere_datastore.datastore1.id
  
  num_cpus = 4
  memory   = 8192
  guest_id = "ubuntu64Guest"
  firmware = "bios"
  
  network_interface {
    network_id   = data.vsphere_network.vm_network.id
    adapter_type = "vmxnet3"
  }
  
  disk {
    label            = "disk0"
    size             = 50
    thin_provisioned = true
    datastore_id     = data.vsphere_datastore.datastore1.id
  }
}
```

## 🗺️ Roadmap

### Version 1.1 (Next Release)
- [ ] Enhanced VM property discovery (resource pools, folders)
- [ ] Advanced filtering options
- [ ] Template and OVA discovery
- [ ] Cluster and host information

### Version 1.2 (Proxmox Support)
- [ ] Proxmox VE API integration
- [ ] Container discovery (LXC)
- [ ] Proxmox-specific IaC generation
- [ ] Multi-node cluster support

### Version 1.3 (Nutanix Support)
- [ ] Nutanix Prism API integration
- [ ] Category and policy discovery
- [ ] Nutanix-specific templates
- [ ] AHV virtual machine support

### Version 2.0 (Advanced Features)
- [ ] Web interface for discovery and generation
- [ ] CI/CD pipeline integration
- [ ] Change detection and drift analysis
- [ ] Incremental discovery and caching
- [ ] Cross-platform migration templates

## 🤝 Contributing

We welcome contributions from infrastructure engineers, DevOps practitioners, and anyone working with virtualization technology.

### Getting Started
1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature`
3. Make your changes and test: `make test`
4. Commit: `git commit -am "Add your feature"`
5. Push: `git push origin feature/your-feature`
6. Create a Pull Request

### Areas for Contribution
- **Hypervisor Providers** - Additional platform support (Hyper-V, KVM)
- **IaC Generators** - New template formats and optimizations
- **Testing** - Integration tests with real hypervisor environments
- **Documentation** - User guides and tutorials

## 🐛 Troubleshooting

### Common Issues

**Build Failures:**
```bash
# Clean and rebuild
make clean
make deps
make build
```

**Connection Issues:**
```bash
# Test credentials
./bin/valhalla auth vmware --test

# Enable debug logging
./bin/valhalla --debug discover --provider vmware
```

**Discovery Issues:**
```bash
# Use dry run mode
./bin/valhalla discover --provider vmware --dry-run

# Check permissions and network connectivity
# Ensure credentials have read access to vCenter
```

### Debug Mode
```bash
# Enable detailed logging
./bin/valhalla --debug --log-format json discover --provider vmware
```

## 📄 License

This project is licensed under the [MIT License](LICENSE) - see the LICENSE file for details.

## 🙏 Acknowledgments

- VMware govmomi library for vSphere API access
- Cobra CLI framework for command-line interface
- HashiCorp for Terraform ecosystem inspiration
- The entire Infrastructure as Code community

## 📞 Support

- **📚 Documentation**: [GitHub Wiki](https://github.com/BigChiefRick/valhalla/wiki)
- **🐛 Issues**: [GitHub Issues](https://github.com/BigChiefRick/valhalla/issues)
- **💬 Discussions**: [GitHub Discussions](https://github.com/BigChiefRick/valhalla/discussions)
- **💼 Enterprise Support**: Contact maintainers for commercial support options

---

<div align="center">

**⚔️ Join the Hall of Heroes ⚔️**

*Where infrastructure warriors earn eternal life through code*

*Built with ❤️ for the hypervisor community*

**Ready for disaster recovery. Ready for the future.**

</div>

## 🚀 Real-World Use Cases

### Disaster Recovery Planning
```bash
# Discover production environment
./bin/valhalla discover --provider vmware --datacenter "Production" --output-file prod-backup.json

# Generate recovery templates
./bin/valhalla generate --input prod-backup.json --format terraform --output-dir ./dr-terraform
./bin/valhalla generate --input prod-backup.json --format ansible --output-dir ./dr-ansible

# Now you have infrastructure-as-code for complete environment recreation
```

### Infrastructure Migration
```bash
# Document current state
./bin/valhalla discover --provider vmware --output-file current-state.json

# Generate templates for new environment
./bin/valhalla generate --input current-state.json --format pulumi-python --output-dir ./migration
```

### Compliance and Documentation
```bash
# Generate current infrastructure documentation
./bin/valhalla discover --provider vmware --format json > infrastructure-$(date +%Y%m%d).json

# Create human-readable reports
./bin/valhalla discover --provider vmware --format table > infrastructure-report.txt
```

---

**Version**: 1.0.0 ✅ **Status**: Production Ready for VMware vSphere
