package providers

import (
	"context"

	"valhalla/internal/config"
	"valhalla/internal/models"
)

// Provider defines the interface for infrastructure discovery providers
type Provider interface {
	// Connect establishes connection to the provider
	Connect(ctx context.Context) error
	
	// Disconnect closes the connection to the provider
	Disconnect() error
	
	// Discover performs infrastructure discovery
	Discover(ctx context.Context) (*models.Infrastructure, error)
	
	// GetName returns the provider name
	GetName() string
	
	// IsConnected returns true if the provider is connected
	IsConnected() bool
}

// VMwareProvider defines the interface for VMware vSphere discovery
type VMwareProvider interface {
	Provider
	
	// Connect with VMware-specific configuration
	Connect(ctx context.Context, cfg config.VMwareConfig) error
	
	// DiscoverDatacenters discovers all datacenters
	DiscoverDatacenters(ctx context.Context) ([]models.Datacenter, error)
	
	// DiscoverClusters discovers clusters in a datacenter
	DiscoverClusters(ctx context.Context, datacenter string) ([]models.Cluster, error)
	
	// DiscoverHosts discovers hosts in a cluster
	DiscoverHosts(ctx context.Context, cluster string) ([]models.Host, error)
	
	// DiscoverVMs discovers virtual machines
	DiscoverVMs(ctx context.Context, filters VMDiscoveryFilters) ([]models.VirtualMachine, error)
	
	// DiscoverNetworks discovers networks
	DiscoverNetworks(ctx context.Context) ([]models.Network, error)
	
	// DiscoverStorage discovers storage
	DiscoverStorage(ctx context.Context) ([]models.Storage, error)
	
	// DiscoverResourcePools discovers resource pools
	DiscoverResourcePools(ctx context.Context) ([]models.ResourcePool, error)
	
	// DiscoverTemplates discovers VM templates
	DiscoverTemplates(ctx context.Context) ([]models.Template, error)
}

// ProxmoxProvider defines the interface for Proxmox discovery
type ProxmoxProvider interface {
	Provider
	
	// Connect with Proxmox-specific configuration
	Connect(ctx context.Context, cfg config.ProxmoxConfig) error
	
	// DiscoverNodes discovers Proxmox nodes
	DiscoverNodes(ctx context.Context) ([]models.Host, error)
	
	// DiscoverVMs discovers virtual machines and containers
	DiscoverVMs(ctx context.Context, filters VMDiscoveryFilters) ([]models.VirtualMachine, error)
	
	// DiscoverNetworks discovers networks
	DiscoverNetworks(ctx context.Context) ([]models.Network, error)
	
	// DiscoverStorage discovers storage
	DiscoverStorage(ctx context.Context) ([]models.Storage, error)
	
	// DiscoverTemplates discovers VM templates
	DiscoverTemplates(ctx context.Context) ([]models.Template, error)
}

// NutanixProvider defines the interface for Nutanix discovery
type NutanixProvider interface {
	Provider
	
	// Connect with Nutanix-specific configuration
	Connect(ctx context.Context, cfg config.NutanixConfig) error
	
	// DiscoverClusters discovers Nutanix clusters
	DiscoverClusters(ctx context.Context) ([]models.Cluster, error)
	
	// DiscoverHosts discovers hosts in a cluster
	DiscoverHosts(ctx context.Context, cluster string) ([]models.Host, error)
	
	// DiscoverVMs discovers virtual machines
	DiscoverVMs(ctx context.Context, filters VMDiscoveryFilters) ([]models.VirtualMachine, error)
	
	// DiscoverNetworks discovers networks
	DiscoverNetworks(ctx context.Context) ([]models.Network, error)
	
	// DiscoverStorage discovers storage
	DiscoverStorage(ctx context.Context) ([]models.Storage, error)
	
	// DiscoverCategories discovers Nutanix categories
	DiscoverCategories(ctx context.Context) (map[string][]string, error)
}

// VMDiscoveryFilters defines filters for VM discovery
type VMDiscoveryFilters struct {
	Datacenter    string   `json:"datacenter,omitempty"`
	Cluster       string   `json:"cluster,omitempty"`
	Host          string   `json:"host,omitempty"`
	Node          string   `json:"node,omitempty"`
	ResourcePool  string   `json:"resource_pool,omitempty"`
	Folder        string   `json:"folder,omitempty"`
	PowerState    string   `json:"power_state,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Names         []string `json:"names,omitempty"`
	IncludeTemplates bool  `json:"include_templates"`
}

// DiscoveryResult represents the result of a discovery operation
type DiscoveryResult struct {
	Provider      string                 `json:"provider"`
	Success       bool                   `json:"success"`
	Error         string                 `json:"error,omitempty"`
	Duration      string                 `json:"duration"`
	ResourceCount int                    `json:"resource_count"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// DiscoveryOptions defines options for discovery operations
type DiscoveryOptions struct {
	Concurrent       int    `json:"concurrent"`
	Timeout          string `json:"timeout"`
	IncludeMetadata  bool   `json:"include_metadata"`
	FollowReferences bool   `json:"follow_references"`
	DetailLevel      string `json:"detail_level"` // basic, detailed, full
}

// ConnectionInfo represents connection information for a provider
type ConnectionInfo struct {
	Server      string                 `json:"server"`
	Port        int                    `json:"port,omitempty"`
	Username    string                 `json:"username"`
	Connected   bool                   `json:"connected"`
	LastConnect string                 `json:"last_connect,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
