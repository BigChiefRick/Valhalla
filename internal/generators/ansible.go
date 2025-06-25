package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"valhalla/internal/logger"
	"valhalla/internal/models"
)

// AnsibleGenerator generates Ansible playbooks
type AnsibleGenerator struct {
	*BaseGenerator
}

// NewAnsibleGenerator creates a new Ansible generator
func NewAnsibleGenerator(log *logger.Logger) Generator {
	return &AnsibleGenerator{
		BaseGenerator: NewBaseGenerator("ansible", "ansible", log),
	}
}

// Generate creates Ansible playbooks from infrastructure models
func (g *AnsibleGenerator) Generate(infrastructures []*models.Infrastructure, opts GenerateOptions) ([]*GenerateResult, error) {
	g.Log().Info("Generating Ansible playbooks", "infrastructures", len(infrastructures))

	var results []*GenerateResult

	// Generate main playbook
	playbook := g.generateMainPlaybook(infrastructures)
	results = append(results, &GenerateResult{
		Path:      "site.yml",
		Content:   []byte(playbook),
		Size:      len(playbook),
		Type:      "playbook",
		Provider:  "ansible",
		Resources: []string{"playbook"},
	})

	// Generate inventory
	inventory := g.generateInventory(infrastructures)
	results = append(results, &GenerateResult{
		Path:      "inventory.yml",
		Content:   []byte(inventory),
		Size:      len(inventory),
		Type:      "inventory",
		Provider:  "ansible",
		Resources: []string{"inventory"},
	})

	// Generate group vars
	groupVars := g.generateGroupVars(infrastructures)
	results = append(results, &GenerateResult{
		Path:      "group_vars/all.yml",
		Content:   []byte(groupVars),
		Size:      len(groupVars),
		Type:      "variables",
		Provider:  "ansible",
		Resources: []string{},
	})

	// Generate provider-specific playbooks
	for _, infra := range infrastructures {
		providerResults, err := g.generateForProvider(infra, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to generate for provider %s: %w", infra.Provider, err)
		}
		results = append(results, providerResults...)
	}

	// Generate requirements
	requirements := g.generateRequirements()
	results = append(results, &GenerateResult{
		Path:      "requirements.yml",
		Content:   []byte(requirements),
		Size:      len(requirements),
		Type:      "requirements",
		Provider:  "ansible",
		Resources: []string{},
	})

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

// generateMainPlaybook generates the main Ansible playbook
func (g *AnsibleGenerator) generateMainPlaybook(infrastructures []*models.Infrastructure) string {
	playbook := `---
# Valhalla Generated Infrastructure Playbook
# This playbook recreates discovered infrastructure using Ansible

- name: Deploy Infrastructure
  hosts: localhost
  gather_facts: false
  vars:
    ansible_python_interpreter: "{{ ansible_playbook_python }}"
  
  tasks:
    - name: Include provider-specific playbooks
      include_tasks: "{{ item }}"
      loop:
`

	for _, infra := range infrastructures {
		providerFile := fmt.Sprintf("tasks/%s.yml", strings.ToLower(infra.Provider))
		playbook += fmt.Sprintf("        - %s\n", providerFile)
	}

	playbook += `
    - name: Display deployment summary
      debug:
        msg: |
          Infrastructure deployment completed successfully!
          
          Deployed resources:
`

	for _, infra := range infrastructures {
		playbook += fmt.Sprintf(`          - %s (%s): %d VMs, %d networks, %d storage volumes
`, strings.ToUpper(infra.Provider), infra.Server, 
			len(infra.VirtualMachines), len(infra.Networks), len(infra.Storage))
	}

	return playbook
}

// generateInventory generates the Ansible inventory
func (g *AnsibleGenerator) generateInventory(infrastructures []*models.Infrastructure) string {
	inventory := `---
# Valhalla Generated Inventory
# This inventory contains discovered infrastructure hosts

all:
  children:
`

	for _, infra := range infrastructures {
		groupName := fmt.Sprintf("%s_%s", strings.ToLower(infra.Provider), 
			strings.ReplaceAll(strings.ToLower(infra.Server), ".", "_"))
		
		inventory += fmt.Sprintf(`    %s:
      hosts:
`, groupName)

		for _, vm := range infra.VirtualMachines {
			if vm.Config.Template {
				continue
			}
			
			hostName := strings.ReplaceAll(strings.ToLower(vm.Name), " ", "_")
			inventory += fmt.Sprintf(`        %s:
          ansible_host: "{{ vm_ip_addresses['%s'] | default('pending') }}"
          vm_name: "%s"
          vm_cpus: %d
          vm_memory: %d
          vm_os: "%s"
          vm_state: "%s"
`, hostName, vm.Name, vm.Name, vm.CPUs, vm.Memory, vm.OperatingSystem, vm.State)
		}

		inventory += fmt.Sprintf(`      vars:
        provider: "%s"
        provider_server: "%s"
        datacenter: "%s"
        cluster: "%s"

`, infra.Provider, infra.Server, infra.Datacenter, infra.Cluster)
	}

	return inventory
}

// generateGroupVars generates group variables
func (g *AnsibleGenerator) generateGroupVars(infrastructures []*models.Infrastructure) string {
	groupVars := `---
# Valhalla Generated Group Variables
# Common variables for all infrastructure

# Connection settings
ansible_connection: local
ansible_python_interpreter: "{{ ansible_playbook_python }}"

# Deployment settings
deployment_mode: "recreate"  # recreate, validate, cleanup
wait_for_ip: true
wait_timeout: 300

# Default VM settings
default_vm_settings:
  cpu_hot_add: true
  memory_hot_add: true
  disk_provisioning: "thin"
  network_type: "vmxnet3"

# Provider configurations
providers:
`

	for _, infra := range infrastructures {
		provider := strings.ToLower(infra.Provider)
		groupVars += fmt.Sprintf(`  %s:
    server: "%s"
    datacenter: "%s"
    cluster: "%s"
    validate_certs: false
    connection_timeout: 30
`, provider, infra.Server, infra.Datacenter, infra.Cluster)

		if provider == "vmware" || provider == "vsphere" {
			groupVars += `    username: "{{ vsphere_username }}"
    password: "{{ vsphere_password }}"
`
		} else if provider == "proxmox" {
			groupVars += `    username: "{{ proxmox_username }}"
    password: "{{ proxmox_password }}"
    node: "{{ proxmox_node }}"
`
		}
	}

	groupVars += `
# Network mappings (customize as needed)
network_mappings:
`

	// Generate network mappings from discovered networks
	networks := make(map[string]bool)
	for _, infra := range infrastructures {
		for _, network := range infra.Networks {
			if !networks[network.Name] {
				groupVars += fmt.Sprintf(`  "%s": "%s"  # Original: %s
`, network.Name, network.Name, network.Name)
				networks[network.Name] = true
			}
		}
	}

	groupVars += `
# Datastore mappings (customize as needed)
datastore_mappings:
`

	// Generate datastore mappings from discovered storage
	datastores := make(map[string]bool)
	for _, infra := range infrastructures {
		for _, storage := range infra.Storage {
			if !datastores[storage.Name] {
				groupVars += fmt.Sprintf(`  "%s": "%s"  # Type: %s, Capacity: %dGB
`, storage.Name, storage.Name, storage.Type, storage.Capacity)
				datastores[storage.Name] = true
			}
		}
	}

	return groupVars
}

// generateForProvider generates provider-specific playbooks
func (g *AnsibleGenerator) generateForProvider(infra *models.Infrastructure, opts GenerateOptions) ([]*GenerateResult, error) {
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

// generateVMware generates VMware-specific Ansible tasks
func (g *AnsibleGenerator) generateVMware(infra *models.Infrastructure, opts GenerateOptions) ([]*GenerateResult, error) {
	content := fmt.Sprintf(`---
# VMware vSphere Tasks - Generated by Valhalla
# Server: %s
# Datacenter: %s

- name: Create VMware Virtual Machines
  community.vmware.vmware_guest:
    hostname: "{{ providers.vmware.server }}"
    username: "{{ providers.vmware.username }}"
    password: "{{ providers.vmware.password }}"
    validate_certs: "{{ providers.vmware.validate_certs }}"
    datacenter: "{{ providers.vmware.datacenter }}"
    cluster: "{{ providers.vmware.cluster }}"
    name: "{{ item.name }}"
    state: "{{ item.state | default('present') }}"
    guest_id: "{{ item.guest_id }}"
    hardware:
      num_cpus: "{{ item.cpus }}"
      memory_mb: "{{ item.memory }}"
      scsi: paravirtual
    disk: "{{ item.disks }}"
    networks: "{{ item.networks }}"
    wait_for_ip_address: "{{ wait_for_ip }}"
    wait_for_ip_address_timeout: "{{ wait_timeout }}"
  loop:
`, infra.Server, infra.Datacenter)

	// Generate VM list
	for _, vm := range infra.VirtualMachines {
		if vm.Config.Template {
			continue
		}

		content += fmt.Sprintf(`    - name: "%s"
      state: "%s"
      guest_id: "%s"
      cpus: %d
      memory: %d
      disks:
`, vm.Name, strings.ToLower(vm.State), vm.Config.GuestID, vm.CPUs, vm.Memory)

		// Add disks
		for i, disk := range vm.Disks {
			content += fmt.Sprintf(`        - size_gb: %d
          type: "%s"
          datastore: "{{ datastore_mappings['%s'] }}"
          scsi_controller: 0
          unit_number: %d
`, disk.Size, strings.ToLower(disk.Type), disk.Datastore, i)
		}

		content += "      networks:\n"
		// Add networks
		for _, nic := range vm.NetworkCards {
			content += fmt.Sprintf(`        - name: "{{ network_mappings['%s'] }}"
          device_type: "%s"
          start_connected: %t
`, nic.Network, nic.Type, nic.StartConnect)
		}
	}

	content += `  register: vm_deploy_result
  when: deployment_mode in ['recreate', 'create']

- name: Store VM IP addresses
  set_fact:
    vm_ip_addresses: "{{ vm_ip_addresses | default({}) | combine({item.item.name: item.instance.ipv4}) }}"
  loop: "{{ vm_deploy_result.results }}"
  when: 
    - vm_deploy_result is defined
    - item.instance is defined
    - item.instance.ipv4 is defined

- name: Display created VMs
  debug:
    msg: |
      Created VM: {{ item.item.name }}
      IP Address: {{ item.instance.ipv4 | default('Pending') }}
      State: {{ item.instance.hw_power_status }}
  loop: "{{ vm_deploy_result.results }}"
  when: vm_deploy_result is defined
`

	return []*GenerateResult{{
		Path:      "tasks/vmware.yml",
		Content:   []byte(content),
		Size:      len(content),
		Type:      "tasks",
		Provider:  "vmware",
		Resources: []string{"vmware_guest"},
	}}, nil
}

// generateProxmox generates Proxmox-specific Ansible tasks
func (g *AnsibleGenerator) generateProxmox(infra *models.Infrastructure, opts GenerateOptions) ([]*GenerateResult, error) {
	content := `---
# Proxmox Tasks - Generated by Valhalla
# TODO: Implement Proxmox task generation

- name: Proxmox infrastructure deployment
  debug:
    msg: "Proxmox Ansible tasks not yet implemented"
`

	return []*GenerateResult{{
		Path:      "tasks/proxmox.yml",
		Content:   []byte(content),
		Size:      len(content),
		Type:      "tasks",
		Provider:  "proxmox",
		Resources: []string{},
	}}, nil
}

// generateNutanix generates Nutanix-specific Ansible tasks
func (g *AnsibleGenerator) generateNutanix(infra *models.Infrastructure, opts GenerateOptions) ([]*GenerateResult, error) {
	content := `---
# Nutanix Tasks - Generated by Valhalla
# TODO: Implement Nutanix task generation

- name: Nutanix infrastructure deployment
  debug:
    msg: "Nutanix Ansible tasks not yet implemented"
`

	return []*GenerateResult{{
		Path:      "tasks/nutanix.yml",
		Content:   []byte(content),
		Size:      len(content),
		Type:      "tasks",
		Provider:  "nutanix",
		Resources: []string{},
	}}, nil
}

// generateRequirements generates Ansible requirements
func (g *AnsibleGenerator) generateRequirements() string {
	return `---
# Ansible Requirements - Generated by Valhalla
# Install with: ansible-galaxy install -r requirements.yml

collections:
  - name: community.vmware
    version: ">=3.0.0"
  - name: community.general
    version: ">=5.0.0"
  - name: ansible.posix
    version: ">=1.0.0"

roles: []
`
}

// writeFile writes a generate result to a file
func (g *AnsibleGenerator) writeFile(result *GenerateResult, outputDir string) error {
	// Ensure output directory exists
	dir := filepath.Dir(filepath.Join(outputDir, result.Path))
	if err := os.MkdirAll(dir, 0755); err != nil {
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
func (g *AnsibleGenerator) GetSupportedFormats() []string {
	return []string{"ansible"}
}

// Validate validates the generated templates
func (g *AnsibleGenerator) Validate(results []*GenerateResult) error {
	// TODO: Implement Ansible validation
	g.Log().Info("Ansible validation not yet implemented")
	return nil
}
