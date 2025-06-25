package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"valhalla/internal/logger"
	"valhalla/internal/models"
)

// TerraformGenerator generates Terraform HCL files
type TerraformGenerator struct {
	*BaseGenerator
}

// NewTerraformGenerator creates a new Terraform generator
func NewTerraformGenerator(log *logger.Logger) Generator {
	return &TerraformGenerator{
		BaseGenerator: NewBaseGenerator("terraform", "terraform", log),
	}
}

// Generate creates Terraform templates from infrastructure models
func (g *TerraformGenerator) Generate(infrastructures []*models.Infrastructure, opts GenerateOptions) ([]*GenerateResult, error) {
	g.Log().Info("Generating Terraform templates", "infrastructures", len(infrastructures))

	var results []*GenerateResult

	for _, infra := range infrastructures {
		providerResults, err := g.generateForProvider(infra, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to generate for provider %s: %w", infra.Provider, err)
		}
		results = append(results, providerResults...)
	}

	// Write files if not dry run
	if !opts.DryRun {
		for _, result := range results {
			if err := g.writeFile(result, opts.OutputDir); err != nil {
				return nil, fmt.Errorf("failed to write file %s: %w", result.Path, err)
			}
		}
	}

	return results, nil
}

// generateForProvider generates Terraform files for a specific provider
func (g *TerraformGenerator) generateForProvider(infra *models.Infrastructure, opts GenerateOptions) ([]*GenerateResult, error) {
	switch strings.ToLower(infra.Provider) {
	case "vmware", "vsphere":
		return g.generateVMware(infra, opts)
	case "proxmox":
		return g.generateProxmox(infra, opts)
	case "nutanix":
		return g.generateNutanix(infra, opts)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", infra.Provider)
	}
}

// generateVMware generates Terraform files for VMware infrastructure
func (g *TerraformGenerator) generateVMware(infra *models.Infrastructure, opts GenerateOptions) ([]*GenerateResult, error) {
	var results []*GenerateResult

	// Generate provider configuration
	providerConfig := g.generateVMwareProvider(infra)
	results = append(results, &GenerateResult{
		Path:      "provider.tf",
		Content:   []byte(providerConfig),
		Size:      len(providerConfig),
		Type:      "provider",
		Provider:  "vmware",
		Resources: []string{"vsphere"},
	})

	// Generate variables
	variables := g.generateVMwareVariables(infra)
	results = append(results, &GenerateResult{
		Path:      "variables.tf",
		Content:   []byte(variables),
		Size:      len(variables),
		Type:      "variables",
		Provider:  "vmware",
		Resources: []string{},
	})

	// Generate data sources
	dataSources := g.generateVMwareDataSources(infra)
	results = append(results, &GenerateResult{
		Path:      "data.tf",
		Content:   []byte(dataSources),
		Size:      len(dataSources),
		Type:      "data",
		Provider:  "vmware",
		Resources: []string{},
	})

	// Generate VMs
	if len(infra.VirtualMachines) > 0 {
		vms := g.generateVMwareVMs(infra.VirtualMachines)
		results = append(results, &GenerateResult{
			Path:      "virtual_machines.tf",
			Content:   []byte(vms),
			Size:      len(vms),
			Type:      "resources",
			Provider:  "vmware",
			Resources: []string{"vsphere_virtual_machine"},
		})
	}

	// Generate outputs
	outputs := g.generateVMwareOutputs(infra)
	results = append(results, &GenerateResult{
		Path:      "outputs.tf",
		Content:   []byte(outputs),
		Size:      len(outputs),
		Type:      "outputs",
		Provider:  "vmware",
		Resources: []string{},
	})

	return results, nil
}

// generateVMwareProvider generates VMware provider configuration
func (g *TerraformGenerator) generateVMwareProvider(infra *models.Infrastructure) string {
	return fmt.Sprintf(`terraform {
  required_providers {
    vsphere = {
      source  = "hashicorp/vsphere"
      version = "~> 2.0"
    }
  }
  required_version = ">= 1.0"
}

provider "vsphere" {
  user                 = var.vsphere_user
  password             = var.vsphere_password
  vsphere_server       = var.vsphere_server
  allow_unverified_ssl = var.vsphere_insecure
}
`)
}

// generateVMwareVariables generates variable definitions
func (g *TerraformGenerator) generateVMwareVariables(infra *models.Infrastructure) string {
	return fmt.Sprintf(`variable "vsphere_server" {
  description = "vSphere server address"
  type        = string
  default     = "%s"
}

variable "vsphere_user" {
  description = "vSphere username"
  type        = string
  sensitive   = true
}

variable "vsphere_password" {
  description = "vSphere password"
  type        = string
  sensitive   = true
}

variable "vsphere_insecure" {
  description = "Allow unverified SSL certificates"
  type        = bool
  default     = true
}

variable "datacenter" {
  description = "vSphere datacenter"
  type        = string
  default     = "%s"
}
`, infra.Server, infra.Datacenter)
}

// generateVMwareDataSources generates data source definitions
func (g *TerraformGenerator) generateVMwareDataSources(infra *models.Infrastructure) string {
	dataConfig := `data "vsphere_datacenter" "dc" {
  name = var.datacenter
}
`

	// Add data sources for discovered resources
	if infra.Cluster != "" {
		dataConfig += fmt.Sprintf(`
data "vsphere_compute_cluster" "cluster" {
  name          = "%s"
  datacenter_id = data.vsphere_datacenter.dc.id
}
`, infra.Cluster)
	}

	// Add common data sources for networks and datastores
	networks := make(map[string]bool)
	datastores := make(map[string]bool)

	for _, vm := range infra.VirtualMachines {
		for _, nic := range vm.NetworkCards {
			if nic.Network != "" {
				networks[nic.Network] = true
			}
		}
		for _, disk := range vm.Disks {
			if disk.Datastore != "" {
				datastores[disk.Datastore] = true
			}
		}
	}

	for network := range networks {
		resourceName := g.GenerateResourceName(network)
		dataConfig += fmt.Sprintf(`
data "vsphere_network" "%s" {
  name          = "%s"
  datacenter_id = data.vsphere_datacenter.dc.id
}
`, resourceName, network)
	}

	for datastore := range datastores {
		resourceName := g.GenerateResourceName(datastore)
		dataConfig += fmt.Sprintf(`
data "vsphere_datastore" "%s" {
  name          = "%s"
  datacenter_id = data.vsphere_datacenter.dc.id
}
`, resourceName, datastore)
	}

	return dataConfig
}

// generateVMwareVMs generates VM resource definitions
func (g *TerraformGenerator) generateVMwareVMs(vms []models.VirtualMachine) string {
	var vmConfigs []string

	for _, vm := range vms {
		// Skip templates
		if vm.Config.Template {
			continue
		}

		resourceName := g.GenerateResourceName(vm.Name)
		
		config := fmt.Sprintf(`resource "vsphere_virtual_machine" "%s" {
  name             = "%s"
  resource_pool_id = data.vsphere_compute_cluster.cluster.resource_pool_id
  datastore_id     = data.vsphere_datastore.%s.id
  
  num_cpus = %d
  memory   = %d
  
  guest_id = "%s"
  
  firmware = "%s"
`, resourceName, vm.Name, g.GenerateResourceName(vm.Disks[0].Datastore), 
   vm.CPUs, vm.Memory, vm.Config.GuestID, strings.ToLower(vm.Hardware.Firmware))

		// Add network interfaces
		for _, nic := range vm.NetworkCards {
			networkResourceName := g.GenerateResourceName(nic.Network)
			config += fmt.Sprintf(`
  network_interface {
    network_id   = data.vsphere_network.%s.id
    adapter_type = "%s"
  }
`, networkResourceName, nic.Type)
		}

		// Add disks
		for i, disk := range vm.Disks {
			datastoreResourceName := g.GenerateResourceName(disk.Datastore)
			config += fmt.Sprintf(`
  disk {
    label            = "disk%d"
    size             = %d
    thin_provisioned = %t
    datastore_id     = data.vsphere_datastore.%s.id
  }
`, i, disk.Size, strings.Contains(disk.Type, "thin"), datastoreResourceName)
		}

		config += "}\n"
		vmConfigs = append(vmConfigs, config)
	}

	return strings.Join(vmConfigs, "\n")
}

// generateVMwareOutputs generates output definitions
func (g *TerraformGenerator) generateVMwareOutputs(infra *models.Infrastructure) string {
	outputs := `output "virtual_machines" {
  description = "Information about created virtual machines"
  value = {
`

	for _, vm := range infra.VirtualMachines {
		if vm.Config.Template {
			continue
		}
		resourceName := g.GenerateResourceName(vm.Name)
		outputs += fmt.Sprintf(`    "%s" = {
      id   = vsphere_virtual_machine.%s.id
      name = vsphere_virtual_machine.%s.name
      ip   = vsphere_virtual_machine.%s.default_ip_address
    }
`, vm.Name, resourceName, resourceName, resourceName)
	}

	outputs += `  }
}
`

	return outputs
}

// generateProxmox generates Terraform files for Proxmox infrastructure
func (g *TerraformGenerator) generateProxmox(infra *models.Infrastructure, opts GenerateOptions) ([]*GenerateResult, error) {
	// TODO: Implement Proxmox Terraform generation
	g.Log().Info("Proxmox Terraform generation not yet implemented")
	return []*GenerateResult{}, nil
}

// generateNutanix generates Terraform files for Nutanix infrastructure
func (g *TerraformGenerator) generateNutanix(infra *models.Infrastructure, opts GenerateOptions) ([]*GenerateResult, error) {
	// TODO: Implement Nutanix Terraform generation
	g.Log().Info("Nutanix Terraform generation not yet implemented")
	return []*GenerateResult{}, nil
}

// writeFile writes a generate result to a file
func (g *TerraformGenerator) writeFile(result *GenerateResult, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write file
	filePath := filepath.Join(outputDir, result.Path)
	if err := os.WriteFile(filePath, result.Content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	result.Path = filePath
	return nil
}

// GetSupportedFormats returns supported output formats
func (g *TerraformGenerator) GetSupportedFormats() []string {
	return []string{"terraform", "tf"}
}

// Validate validates the generated templates
func (g *TerraformGenerator) Validate(results []*GenerateResult) error {
	// TODO: Implement Terraform validation
	g.Log().Info("Terraform validation not yet implemented")
	return nil
}
