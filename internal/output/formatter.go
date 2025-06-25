package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
	"valhalla/internal/models"
)

// Formatter handles output formatting for discovery results
type Formatter struct {
	format string
}

// NewFormatter creates a new output formatter
func NewFormatter(format string) *Formatter {
	return &Formatter{
		format: strings.ToLower(format),
	}
}

// Format formats the infrastructure results according to the specified format
func (f *Formatter) Format(infrastructures []*models.Infrastructure) ([]byte, error) {
	switch f.format {
	case "json":
		return f.formatJSON(infrastructures)
	case "yaml", "yml":
		return f.formatYAML(infrastructures)
	case "table":
		return f.formatTable(infrastructures)
	case "csv":
		return f.formatCSV(infrastructures)
	default:
		return nil, fmt.Errorf("unsupported output format: %s", f.format)
	}
}

// formatJSON formats output as JSON
func (f *Formatter) formatJSON(infrastructures []*models.Infrastructure) ([]byte, error) {
	return json.MarshalIndent(infrastructures, "", "  ")
}

// formatYAML formats output as YAML
func (f *Formatter) formatYAML(infrastructures []*models.Infrastructure) ([]byte, error) {
	return yaml.Marshal(infrastructures)
}

// formatTable formats output as a human-readable table
func (f *Formatter) formatTable(infrastructures []*models.Infrastructure) ([]byte, error) {
	var output strings.Builder

	for _, infra := range infrastructures {
		output.WriteString(fmt.Sprintf("\n=== %s Infrastructure (%s) ===\n", 
			strings.ToUpper(infra.Provider), infra.Server))
		
		if infra.Datacenter != "" {
			output.WriteString(fmt.Sprintf("Datacenter: %s\n", infra.Datacenter))
		}
		if infra.Cluster != "" {
			output.WriteString(fmt.Sprintf("Cluster: %s\n", infra.Cluster))
		}
		if infra.Node != "" {
			output.WriteString(fmt.Sprintf("Node: %s\n", infra.Node))
		}
		
		output.WriteString(fmt.Sprintf("Discovery Time: %s\n\n", 
			infra.DiscoveryTime.Format("2006-01-02 15:04:05")))

		// Virtual Machines Table
		if len(infra.VirtualMachines) > 0 {
			output.WriteString("Virtual Machines:\n")
			vmTable := f.createVMTable(infra.VirtualMachines)
			output.WriteString(vmTable)
			output.WriteString("\n")
		}

		// Networks Table
		if len(infra.Networks) > 0 {
			output.WriteString("Networks:\n")
			networkTable := f.createNetworkTable(infra.Networks)
			output.WriteString(networkTable)
			output.WriteString("\n")
		}

		// Storage Table
		if len(infra.Storage) > 0 {
			output.WriteString("Storage:\n")
			storageTable := f.createStorageTable(infra.Storage)
			output.WriteString(storageTable)
			output.WriteString("\n")
		}

		// Resource Pools Table
		if len(infra.ResourcePools) > 0 {
			output.WriteString("Resource Pools:\n")
			rpTable := f.createResourcePoolTable(infra.ResourcePools)
			output.WriteString(rpTable)
			output.WriteString("\n")
		}

		// Templates Table
		if len(infra.Templates) > 0 {
			output.WriteString("Templates:\n")
			templateTable := f.createTemplateTable(infra.Templates)
			output.WriteString(templateTable)
			output.WriteString("\n")
		}

		// Summary
		total := len(infra.VirtualMachines) + len(infra.Networks) + 
			len(infra.Storage) + len(infra.ResourcePools) + len(infra.Templates)
		output.WriteString(fmt.Sprintf("Total Resources: %d\n", total))
		output.WriteString(strings.Repeat("=", 80) + "\n")
	}

	return []byte(output.String()), nil
}

// createVMTable creates a table for virtual machines
func (f *Formatter) createVMTable(vms []models.VirtualMachine) string {
	var output strings.Builder
	
	table := tablewriter.NewWriter(&output)
	table.SetHeader([]string{"Name", "State", "CPU", "Memory (MB)", "OS", "Host"})
	table.SetBorder(true)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, vm := range vms {
		host := vm.Host
		if host == "" {
			host = "N/A"
		}
		
		os := vm.OperatingSystem
		if os == "" {
			os = "Unknown"
		}
		
		table.Append([]string{
			vm.Name,
			vm.State,
			strconv.Itoa(vm.CPUs),
			strconv.FormatInt(vm.Memory, 10),
			os,
			host,
		})
	}
	
	table.Render()
	return output.String()
}

// createNetworkTable creates a table for networks
func (f *Formatter) createNetworkTable(networks []models.Network) string {
	var output strings.Builder
	
	table := tablewriter.NewWriter(&output)
	table.SetHeader([]string{"Name", "Type", "VLAN", "VSwitch", "DHCP"})
	table.SetBorder(true)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, network := range networks {
		vlan := "N/A"
		if network.VLAN > 0 {
			vlan = strconv.Itoa(network.VLAN)
		}
		
		vswitch := network.VSwitch
		if vswitch == "" {
			vswitch = "N/A"
		}
		
		dhcp := "No"
		if network.DHCP {
			dhcp = "Yes"
		}
		
		table.Append([]string{
			network.Name,
			network.Type,
			vlan,
			vswitch,
			dhcp,
		})
	}
	
	table.Render()
	return output.String()
}

// createStorageTable creates a table for storage
func (f *Formatter) createStorageTable(storage []models.Storage) string {
	var output strings.Builder
	
	table := tablewriter.NewWriter(&output)
	table.SetHeader([]string{"Name", "Type", "Capacity (GB)", "Free (GB)", "Used (%)", "Accessible"})
	table.SetBorder(true)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, store := range storage {
		usedPercent := "N/A"
		if store.Capacity > 0 {
			pct := float64(store.UsedSpace) / float64(store.Capacity) * 100
			usedPercent = fmt.Sprintf("%.1f", pct)
		}
		
		accessible := "No"
		if store.Accessible {
			accessible = "Yes"
		}
		
		table.Append([]string{
			store.Name,
			store.Type,
			strconv.FormatInt(store.Capacity, 10),
			strconv.FormatInt(store.FreeSpace, 10),
			usedPercent,
			accessible,
		})
	}
	
	table.Render()
	return output.String()
}

// createResourcePoolTable creates a table for resource pools
func (f *Formatter) createResourcePoolTable(pools []models.ResourcePool) string {
	var output strings.Builder
	
	table := tablewriter.NewWriter(&output)
	table.SetHeader([]string{"Name", "CPU Limit", "Memory Limit", "CPU Shares", "Memory Shares"})
	table.SetBorder(true)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, pool := range pools {
		cpuLimit := "Unlimited"
		if pool.CPU.Limit > 0 {
			cpuLimit = strconv.FormatInt(pool.CPU.Limit, 10)
		}
		
		memLimit := "Unlimited"
		if pool.Memory.Limit > 0 {
			memLimit = strconv.FormatInt(pool.Memory.Limit, 10)
		}
		
		table.Append([]string{
			pool.Name,
			cpuLimit,
			memLimit,
			pool.CPU.Shares,
			pool.Memory.Shares,
		})
	}
	
	table.Render()
	return output.String()
}

// createTemplateTable creates a table for templates
func (f *Formatter) createTemplateTable(templates []models.Template) string {
	var output strings.Builder
	
	table := tablewriter.NewWriter(&output)
	table.SetHeader([]string{"Name", "OS", "CPU", "Memory (MB)", "Disks"})
	table.SetBorder(true)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, template := range templates {
		os := template.OperatingSystem
		if os == "" {
			os = "Unknown"
		}
		
		diskCount := strconv.Itoa(len(template.Disks))
		
		table.Append([]string{
			template.Name,
			os,
			strconv.Itoa(template.CPUs),
			strconv.FormatInt(template.Memory, 10),
			diskCount,
		})
	}
	
	table.Render()
	return output.String()
}

// formatCSV formats output as CSV
func (f *Formatter) formatCSV(infrastructures []*models.Infrastructure) ([]byte, error) {
	var output strings.Builder
	
	// CSV Header
	output.WriteString("Provider,Server,Datacenter,Cluster,Node,Resource_Type,Name,State,CPUs,Memory_MB,OS,Host,Type,Capacity_GB,Free_GB,VLAN,Network\n")
	
	for _, infra := range infrastructures {
		// Virtual Machines
		for _, vm := range infra.VirtualMachines {
			output.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,VM,%s,%s,%d,%d,%s,%s,,,,,%s\n",
				infra.Provider, infra.Server, infra.Datacenter, infra.Cluster, infra.Node,
				vm.Name, vm.State, vm.CPUs, vm.Memory, 
				strings.ReplaceAll(vm.OperatingSystem, ",", ";"), vm.Host,
				strings.Join(f.getVMNetworks(vm), ";")))
		}
		
		// Networks
		for _, network := range infra.Networks {
			output.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,Network,%s,,%s,,,,%s,,,%d,\n",
				infra.Provider, infra.Server, infra.Datacenter, infra.Cluster, infra.Node,
				network.Name, network.Type, network.VLAN))
		}
		
		// Storage
		for _, storage := range infra.Storage {
			output.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,Storage,%s,,,,,,%s,%d,%d,,\n",
				infra.Provider, infra.Server, infra.Datacenter, infra.Cluster, infra.Node,
				storage.Name, storage.Type, storage.Capacity, storage.FreeSpace))
		}
	}
	
	return []byte(output.String()), nil
}

// getVMNetworks extracts network names from a VM
func (f *Formatter) getVMNetworks(vm models.VirtualMachine) []string {
	var networks []string
	for _, nic := range vm.NetworkCards {
		if nic.Network != "" {
			networks = append(networks, nic.Network)
		}
	}
	return networks
}

// FormatSummary creates a summary of the discovery results
func (f *Formatter) FormatSummary(infrastructures []*models.Infrastructure) string {
	var output strings.Builder
	
	totalVMs := 0
	totalNetworks := 0
	totalStorage := 0
	totalTemplates := 0
	
	output.WriteString("=== Discovery Summary ===\n\n")
	
	for _, infra := range infrastructures {
		totalVMs += len(infra.VirtualMachines)
		totalNetworks += len(infra.Networks)
		totalStorage += len(infra.Storage)
		totalTemplates += len(infra.Templates)
		
		output.WriteString(fmt.Sprintf("%s (%s):\n", 
			strings.ToUpper(infra.Provider), infra.Server))
		output.WriteString(fmt.Sprintf("  Virtual Machines: %d\n", len(infra.VirtualMachines)))
		output.WriteString(fmt.Sprintf("  Networks: %d\n", len(infra.Networks)))
		output.WriteString(fmt.Sprintf("  Storage: %d\n", len(infra.Storage)))
		output.WriteString(fmt.Sprintf("  Templates: %d\n", len(infra.Templates)))
		output.WriteString("\n")
	}
	
	output.WriteString("Total Resources:\n")
	output.WriteString(fmt.Sprintf("  Virtual Machines: %d\n", totalVMs))
	output.WriteString(fmt.Sprintf("  Networks: %d\n", totalNetworks))
	output.WriteString(fmt.Sprintf("  Storage: %d\n", totalStorage))
	output.WriteString(fmt.Sprintf("  Templates: %d\n", totalTemplates))
	output.WriteString(fmt.Sprintf("  Grand Total: %d\n", totalVMs+totalNetworks+totalStorage+totalTemplates))
	
	return output.String()
}
