package providers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"

	"valhalla/internal/config"
	"valhalla/internal/logger"
	"valhalla/internal/models"
)

// vmwareProvider implements the VMwareProvider interface
type vmwareProvider struct {
	log       *logger.Logger
	client    *govmomi.Client
	finder    *find.Finder
	config    config.VMwareConfig
	connected bool
}

// NewVMwareProvider creates a new VMware provider
func NewVMwareProvider(log *logger.Logger) VMwareProvider {
	return &vmwareProvider{
		log: log,
	}
}

// Connect establishes connection to vCenter with VMware-specific configuration
func (p *vmwareProvider) ConnectVMware(ctx context.Context, cfg config.VMwareConfig) error {
	p.config = cfg
	
	// Parse server URL
	u, err := soap.ParseURL(cfg.Server)
	if err != nil {
		return fmt.Errorf("failed to parse vCenter URL: %w", err)
	}

	// Set credentials
	u.User = url.UserPassword(cfg.Username, cfg.Password)

	// Create SOAP client with TLS configuration
	soapClient := soap.NewClient(u, cfg.Insecure)
	if cfg.Insecure {
		soapClient.DefaultTransport().TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Create vim25 client
	vimClient, err := vim25.NewClient(ctx, soapClient)
	if err != nil {
		return fmt.Errorf("failed to create vim25 client: %w", err)
	}

	// Create govmomi client
	p.client = &govmomi.Client{
		Client:         vimClient,
		SessionManager: session.NewManager(vimClient),
	}

	// Login to vCenter
	p.log.Info("Authenticating to vCenter", "server", cfg.Server, "username", cfg.Username)
	err = p.client.Login(ctx, u.User)
	if err != nil {
		return fmt.Errorf("failed to login to vCenter: %w", err)
	}

	// Create finder
	p.finder = find.NewFinder(p.client.Client, true)
	
	// Set datacenter if specified
	if cfg.Datacenter != "" {
		dc, err := p.finder.Datacenter(ctx, cfg.Datacenter)
		if err != nil {
			return fmt.Errorf("failed to find datacenter %s: %w", cfg.Datacenter, err)
		}
		p.finder.SetDatacenter(dc)
		p.log.Info("Set datacenter context", "datacenter", cfg.Datacenter)
	}

	p.connected = true
	p.log.Info("Successfully connected to vCenter", "server", cfg.Server)
	
	return nil
}

// Disconnect closes the vCenter connection
func (p *vmwareProvider) Disconnect() error {
	if p.client != nil && p.connected {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		err := p.client.Logout(ctx)
		p.connected = false
		
		if err != nil {
			p.log.Error("Error during logout", "error", err)
			return err
		}
		
		p.log.Info("Disconnected from vCenter")
	}
	return nil
}

// Discover performs complete infrastructure discovery
func (p *vmwareProvider) Discover(ctx context.Context) (*models.Infrastructure, error) {
	if !p.connected {
		return nil, fmt.Errorf("not connected to vCenter")
	}

	infrastructure := &models.Infrastructure{
		Provider:      "vmware",
		Server:        p.config.Server,
		Datacenter:    p.config.Datacenter,
		Cluster:       p.config.Cluster,
		DiscoveryTime: time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	// Discover VMs
	p.log.Info("Discovering virtual machines")
	vms, err := p.DiscoverVMs(ctx, VMDiscoveryFilters{
		Datacenter: p.config.Datacenter,
		Cluster:    p.config.Cluster,
	})
	if err != nil {
		p.log.Error("Failed to discover VMs", "error", err)
		// Don't fail completely, just log and continue
	} else {
		infrastructure.VirtualMachines = vms
		p.log.Info("Discovered virtual machines", "count", len(vms))
	}

	// Discover Networks
	p.log.Info("Discovering networks")
	networks, err := p.DiscoverNetworks(ctx)
	if err != nil {
		p.log.Error("Failed to discover networks", "error", err)
	} else {
		infrastructure.Networks = networks
		p.log.Info("Discovered networks", "count", len(networks))
	}

	// Discover Storage
	p.log.Info("Discovering storage")
	storage, err := p.DiscoverStorage(ctx)
	if err != nil {
		p.log.Error("Failed to discover storage", "error", err)
	} else {
		infrastructure.Storage = storage
		p.log.Info("Discovered storage", "count", len(storage))
	}

	// Add basic metadata
	totalResources := len(infrastructure.VirtualMachines) + len(infrastructure.Networks) + len(infrastructure.Storage)
	infrastructure.Metadata["total_resources"] = totalResources
	infrastructure.Metadata["discovery_duration"] = time.Since(infrastructure.DiscoveryTime).String()

	return infrastructure, nil
}

// DiscoverVMs discovers virtual machines
func (p *vmwareProvider) DiscoverVMs(ctx context.Context, filters VMDiscoveryFilters) ([]models.VirtualMachine, error) {
	// Find all VMs
	vms, err := p.finder.VirtualMachineList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	var vmList []models.VirtualMachine
	
	// Simple approach - get basic properties for each VM
	for _, vm := range vms {
		var moVM mo.VirtualMachine
		err := vm.Properties(ctx, vm.Reference(), []string{"name", "runtime", "config", "summary"}, &moVM)
		if err != nil {
			p.log.Error("Failed to get VM properties", "vm", vm.Name(), "error", err)
			continue
		}

		// Skip templates unless specifically requested
		if moVM.Config != nil && moVM.Config.Template && !filters.IncludeTemplates {
			continue
		}

		vmModel := models.VirtualMachine{
			ID:         moVM.Reference().Value,
			Name:       moVM.Name,
			State:      string(moVM.Runtime.PowerState),
			PowerState: string(moVM.Runtime.PowerState),
			Metadata:   make(map[string]interface{}),
		}

		// Basic configuration
		if moVM.Config != nil {
			vmModel.CPUs = int(moVM.Config.Hardware.NumCPU)
			vmModel.Memory = int64(moVM.Config.Hardware.MemoryMB)
			vmModel.Config = models.VMConfig{
				Template:      moVM.Config.Template,
				GuestID:       moVM.Config.GuestId,
				UUID:          moVM.Config.Uuid,
				InstanceUUID:  moVM.Config.InstanceUuid,
				ChangeVersion: moVM.Config.ChangeVersion,
			}
			
			// Handle Modified time safely
			if !moVM.Config.Modified.IsZero() {
				vmModel.Config.Modified = moVM.Config.Modified
			}

			vmModel.Hardware = models.HardwareInfo{
				Version:           moVM.Config.Version,
				NumCPU:           int(moVM.Config.Hardware.NumCPU),
				NumCoresPerSocket: int(moVM.Config.Hardware.NumCoresPerSocket),
				MemoryMB:         int64(moVM.Config.Hardware.MemoryMB),
				Firmware:         moVM.Config.Firmware,
			}
		}

		// Guest information
		if moVM.Guest != nil {
			vmModel.OperatingSystem = moVM.Guest.GuestFullName
		}

		// Apply filters
		if p.vmMatchesFilters(vmModel, filters) {
			vmList = append(vmList, vmModel)
		}
	}

	return vmList, nil
}

// vmMatchesFilters checks if a VM matches the given filters
func (p *vmwareProvider) vmMatchesFilters(vm models.VirtualMachine, filters VMDiscoveryFilters) bool {
	// Power state filter
	if filters.PowerState != "" && vm.PowerState != filters.PowerState {
		return false
	}

	// Name filter
	if len(filters.Names) > 0 {
		nameMatch := false
		for _, name := range filters.Names {
			if strings.Contains(strings.ToLower(vm.Name), strings.ToLower(name)) {
				nameMatch = true
				break
			}
		}
		if !nameMatch {
			return false
		}
	}

	return true
}

// DiscoverNetworks discovers network configurations
func (p *vmwareProvider) DiscoverNetworks(ctx context.Context) ([]models.Network, error) {
	// Find all networks
	networks, err := p.finder.NetworkList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	var networkList []models.Network

	for _, network := range networks {
		net := models.Network{
			ID:       network.Reference().Value,
			Name:     network.GetInventoryPath(),
			Metadata: make(map[string]interface{}),
		}

		// Determine network type
		switch network.(type) {
		case *object.Network:
			net.Type = "standard"
		case *object.DistributedVirtualPortgroup:
			net.Type = "distributed"
		case *object.OpaqueNetwork:
			net.Type = "opaque"
		default:
			net.Type = "unknown"
		}

		networkList = append(networkList, net)
	}

	return networkList, nil
}

// DiscoverStorage discovers storage configurations
func (p *vmwareProvider) DiscoverStorage(ctx context.Context) ([]models.Storage, error) {
	// Find all datastores
	datastores, err := p.finder.DatastoreList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list datastores: %w", err)
	}

	var storageList []models.Storage

	for _, ds := range datastores {
		var moDS mo.Datastore
		err := ds.Properties(ctx, ds.Reference(), []string{"name", "summary"}, &moDS)
		if err != nil {
			p.log.Error("Failed to get datastore properties", "datastore", ds.Name(), "error", err)
			continue
		}

		storage := models.Storage{
			ID:         moDS.Reference().Value,
			Name:       moDS.Name,
			Accessible: moDS.Summary.Accessible,
			Metadata:   make(map[string]interface{}),
		}

		if moDS.Summary.Capacity > 0 {
			storage.Capacity = moDS.Summary.Capacity / 1024 / 1024 / 1024 // Convert to GB
			storage.FreeSpace = moDS.Summary.FreeSpace / 1024 / 1024 / 1024
			storage.UsedSpace = storage.Capacity - storage.FreeSpace
		}

		// Get datastore type
		if moDS.Summary.Type != "" {
			storage.Type = moDS.Summary.Type
		}

		storageList = append(storageList, storage)
	}

	return storageList, nil
}

// Simplified implementations for interface compliance

func (p *vmwareProvider) DiscoverResourcePools(ctx context.Context) ([]models.ResourcePool, error) {
	p.log.Info("Resource pool discovery simplified for initial build")
	return []models.ResourcePool{}, nil
}

func (p *vmwareProvider) DiscoverTemplates(ctx context.Context) ([]models.Template, error) {
	p.log.Info("Template discovery simplified for initial build")
	return []models.Template{}, nil
}

func (p *vmwareProvider) DiscoverDatacenters(ctx context.Context) ([]models.Datacenter, error) {
	dcs, err := p.finder.DatacenterList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list datacenters: %w", err)
	}

	var datacenterList []models.Datacenter
	for _, dc := range dcs {
		datacenter := models.Datacenter{
			ID:       dc.Reference().Value,
			Name:     dc.Name(),
			Provider: "vmware",
			Metadata: make(map[string]interface{}),
		}
		datacenterList = append(datacenterList, datacenter)
	}

	return datacenterList, nil
}

func (p *vmwareProvider) DiscoverClusters(ctx context.Context, datacenter string) ([]models.Cluster, error) {
	p.log.Info("Cluster discovery simplified for initial build")
	return []models.Cluster{}, nil
}

func (p *vmwareProvider) DiscoverHosts(ctx context.Context, cluster string) ([]models.Host, error) {
	p.log.Info("Host discovery simplified for initial build")
	return []models.Host{}, nil
}

// GetName returns the provider name
func (p *vmwareProvider) GetName() string {
	return "vmware"
}

// IsConnected returns true if connected to vCenter
func (p *vmwareProvider) IsConnected() bool {
	return p.connected && p.client != nil
}

// Connect without configuration (implements Provider interface)
func (p *vmwareProvider) Connect(ctx context.Context) error {
	return fmt.Errorf("use ConnectVMware(ctx, config.VMwareConfig) instead")
}
