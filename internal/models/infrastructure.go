package models

import (
	"time"
)

// Infrastructure represents discovered infrastructure from a hypervisor
type Infrastructure struct {
	Provider       string                 `json:"provider" yaml:"provider"`
	Server         string                 `json:"server" yaml:"server"`
	Datacenter     string                 `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	Cluster        string                 `json:"cluster,omitempty" yaml:"cluster,omitempty"`
	Node           string                 `json:"node,omitempty" yaml:"node,omitempty"`
	DiscoveryTime  time.Time             `json:"discovery_time" yaml:"discovery_time"`
	VirtualMachines []VirtualMachine       `json:"virtual_machines" yaml:"virtual_machines"`
	Networks       []Network             `json:"networks" yaml:"networks"`
	Storage        []Storage             `json:"storage" yaml:"storage"`
	ResourcePools  []ResourcePool        `json:"resource_pools,omitempty" yaml:"resource_pools,omitempty"`
	Templates      []Template            `json:"templates,omitempty" yaml:"templates,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// VirtualMachine represents a discovered virtual machine
type VirtualMachine struct {
	ID              string                 `json:"id" yaml:"id"`
	Name            string                 `json:"name" yaml:"name"`
	State           string                 `json:"state" yaml:"state"`
	PowerState      string                 `json:"power_state" yaml:"power_state"`
	OperatingSystem string                 `json:"operating_system,omitempty" yaml:"operating_system,omitempty"`
	CPUs            int                    `json:"cpus" yaml:"cpus"`
	Memory          int64                  `json:"memory" yaml:"memory"` // Memory in MB
	Disks           []Disk                 `json:"disks" yaml:"disks"`
	NetworkCards    []NetworkCard          `json:"network_cards" yaml:"network_cards"`
	Annotations     map[string]string      `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Tags            []string               `json:"tags,omitempty" yaml:"tags,omitempty"`
	ResourcePool    string                 `json:"resource_pool,omitempty" yaml:"resource_pool,omitempty"`
	Folder          string                 `json:"folder,omitempty" yaml:"folder,omitempty"`
	Host            string                 `json:"host,omitempty" yaml:"host,omitempty"`
	Tools           VMTools                `json:"tools,omitempty" yaml:"tools,omitempty"`
	Hardware        HardwareInfo           `json:"hardware" yaml:"hardware"`
	Config          VMConfig               `json:"config" yaml:"config"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Disk represents a virtual disk
type Disk struct {
	ID           string `json:"id" yaml:"id"`
	Name         string `json:"name,omitempty" yaml:"name,omitempty"`
	Size         int64  `json:"size" yaml:"size"` // Size in GB
	Type         string `json:"type" yaml:"type"` // thick, thin, etc.
	Datastore    string `json:"datastore" yaml:"datastore"`
	Path         string `json:"path,omitempty" yaml:"path,omitempty"`
	SCSI         string `json:"scsi,omitempty" yaml:"scsi,omitempty"`
	Controller   string `json:"controller,omitempty" yaml:"controller,omitempty"`
	Unit         int    `json:"unit,omitempty" yaml:"unit,omitempty"`
}

// NetworkCard represents a virtual network card
type NetworkCard struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Type        string `json:"type" yaml:"type"` // vmxnet3, e1000, etc.
	Network     string `json:"network" yaml:"network"`
	MACAddress  string `json:"mac_address,omitempty" yaml:"mac_address,omitempty"`
	Connected   bool   `json:"connected" yaml:"connected"`
	StartConnect bool   `json:"start_connect" yaml:"start_connect"`
}

// VMTools represents VMware Tools information
type VMTools struct {
	Status        string `json:"status" yaml:"status"`
	Version       string `json:"version,omitempty" yaml:"version,omitempty"`
	RunningStatus string `json:"running_status" yaml:"running_status"`
}

// HardwareInfo represents virtual machine hardware information
type HardwareInfo struct {
	Version         string `json:"version" yaml:"version"`
	NumCPU          int    `json:"num_cpu" yaml:"num_cpu"`
	NumCoresPerSocket int  `json:"num_cores_per_socket" yaml:"num_cores_per_socket"`
	MemoryMB        int64  `json:"memory_mb" yaml:"memory_mb"`
	Firmware        string `json:"firmware" yaml:"firmware"` // BIOS, EFI
}

// VMConfig represents virtual machine configuration
type VMConfig struct {
	Template         bool   `json:"template" yaml:"template"`
	GuestID          string `json:"guest_id" yaml:"guest_id"`
	UUID             string `json:"uuid" yaml:"uuid"`
	InstanceUUID     string `json:"instance_uuid,omitempty" yaml:"instance_uuid,omitempty"`
	ChangeVersion    string `json:"change_version,omitempty" yaml:"change_version,omitempty"`
	Modified         time.Time `json:"modified,omitempty" yaml:"modified,omitempty"`
}

// Network represents a discovered network
type Network struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Type        string                 `json:"type" yaml:"type"` // standard, distributed, bridge, etc.
	VLAN        int                    `json:"vlan,omitempty" yaml:"vlan,omitempty"`
	VSwitch     string                 `json:"vswitch,omitempty" yaml:"vswitch,omitempty"`
	Subnet      string                 `json:"subnet,omitempty" yaml:"subnet,omitempty"`
	Gateway     string                 `json:"gateway,omitempty" yaml:"gateway,omitempty"`
	DNS         []string               `json:"dns,omitempty" yaml:"dns,omitempty"`
	DHCP        bool                   `json:"dhcp" yaml:"dhcp"`
	Bridge      string                 `json:"bridge,omitempty" yaml:"bridge,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Storage represents discovered storage
type Storage struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Type        string                 `json:"type" yaml:"type"` // VMFS, NFS, local, etc.
	Capacity    int64                  `json:"capacity" yaml:"capacity"` // Capacity in GB
	FreeSpace   int64                  `json:"free_space" yaml:"free_space"` // Free space in GB
	UsedSpace   int64                  `json:"used_space" yaml:"used_space"` // Used space in GB
	URL         string                 `json:"url,omitempty" yaml:"url,omitempty"`
	Accessible  bool                   `json:"accessible" yaml:"accessible"`
	Multipath   bool                   `json:"multipath,omitempty" yaml:"multipath,omitempty"`
	SSD         bool                   `json:"ssd,omitempty" yaml:"ssd,omitempty"`
	Local       bool                   `json:"local,omitempty" yaml:"local,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ResourcePool represents a resource pool
type ResourcePool struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	CPU         ResourceAllocation     `json:"cpu" yaml:"cpu"`
	Memory      ResourceAllocation     `json:"memory" yaml:"memory"`
	Parent      string                 `json:"parent,omitempty" yaml:"parent,omitempty"`
	Children    []string               `json:"children,omitempty" yaml:"children,omitempty"`
	VMs         []string               `json:"vms,omitempty" yaml:"vms,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ResourceAllocation represents resource allocation settings
type ResourceAllocation struct {
	Reservation int64  `json:"reservation" yaml:"reservation"`
	Limit       int64  `json:"limit" yaml:"limit"`
	Shares      string `json:"shares" yaml:"shares"` // low, normal, high, custom
	SharesValue int32  `json:"shares_value,omitempty" yaml:"shares_value,omitempty"`
}

// Template represents a virtual machine template
type Template struct {
	ID              string                 `json:"id" yaml:"id"`
	Name            string                 `json:"name" yaml:"name"`
	OperatingSystem string                 `json:"operating_system,omitempty" yaml:"operating_system,omitempty"`
	CPUs            int                    `json:"cpus" yaml:"cpus"`
	Memory          int64                  `json:"memory" yaml:"memory"`
	Disks           []Disk                 `json:"disks" yaml:"disks"`
	NetworkCards    []NetworkCard          `json:"network_cards" yaml:"network_cards"`
	Annotations     map[string]string      `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Tags            []string               `json:"tags,omitempty" yaml:"tags,omitempty"`
	Folder          string                 `json:"folder,omitempty" yaml:"folder,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Host represents a hypervisor host
type Host struct {
	ID              string                 `json:"id" yaml:"id"`
	Name            string                 `json:"name" yaml:"name"`
	Type            string                 `json:"type" yaml:"type"` // ESXi, Proxmox, Nutanix
	Version         string                 `json:"version" yaml:"version"`
	State           string                 `json:"state" yaml:"state"`
	ConnectionState string                 `json:"connection_state" yaml:"connection_state"`
	CPU             HostResource           `json:"cpu" yaml:"cpu"`
	Memory          HostResource           `json:"memory" yaml:"memory"`
	Storage         []Storage              `json:"storage" yaml:"storage"`
	Networks        []Network              `json:"networks" yaml:"networks"`
	VMs             []string               `json:"vms" yaml:"vms"`
	Cluster         string                 `json:"cluster,omitempty" yaml:"cluster,omitempty"`
	Datacenter      string                 `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// HostResource represents host resource information
type HostResource struct {
	Total     int64 `json:"total" yaml:"total"`
	Used      int64 `json:"used" yaml:"used"`
	Available int64 `json:"available" yaml:"available"`
	Reserved  int64 `json:"reserved,omitempty" yaml:"reserved,omitempty"`
}

// Cluster represents a cluster of hosts
type Cluster struct {
	ID              string                 `json:"id" yaml:"id"`
	Name            string                 `json:"name" yaml:"name"`
	Hosts           []string               `json:"hosts" yaml:"hosts"`
	ResourcePools   []string               `json:"resource_pools,omitempty" yaml:"resource_pools,omitempty"`
	DRS             bool                   `json:"drs,omitempty" yaml:"drs,omitempty"`
	HA              bool                   `json:"ha,omitempty" yaml:"ha,omitempty"`
	VMs             []string               `json:"vms" yaml:"vms"`
	TotalCPU        int64                  `json:"total_cpu" yaml:"total_cpu"`
	TotalMemory     int64                  `json:"total_memory" yaml:"total_memory"`
	UsedCPU         int64                  `json:"used_cpu" yaml:"used_cpu"`
	UsedMemory      int64                  `json:"used_memory" yaml:"used_memory"`
	Datacenter      string                 `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
// Add these types to the end of internal/models/infrastructure.go

// Datacenter represents a hypervisor datacenter
type Datacenter struct {
	ID       string                 `json:"id" yaml:"id"`
	Name     string                 `json:"name" yaml:"name"`
	Provider string                 `json:"provider" yaml:"provider"`
	Clusters []string               `json:"clusters,omitempty" yaml:"clusters,omitempty"`
	Hosts    []string               `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	Networks []string               `json:"networks,omitempty" yaml:"networks,omitempty"`
	Storage  []string               `json:"datastores,omitempty" yaml:"datastores,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
