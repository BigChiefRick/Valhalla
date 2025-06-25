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
	log      *logger.Logger
	client   *govmomi.Client
	finder   *find.Finder
	config   config.VMwareConfig
	connected bool
}

// NewVMwareProvider creates a new VMware provider
func NewVMwareProvider(log *logger.Logger) VMwareProvider {
	return &vmwareProvider{
		log: log,
	}
}

// Connect establishes connection to vCenter with VMware-specific configuration
func (p *vmwareProvider) Connect(ctx context.Context, cfg config.VMwareConfig) error {
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
		return nil, fmt.Errorf("failed to discover VMs: %w", err)
	}
	infrastructure.VirtualMachines = vms
	p.log.Info("Discovered virtual machines", "count", len(vms))

	// Discover Networks
	p.log.Info("Discovering networks")
	networks, err := p.DiscoverNetworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover networks: %w", err)
	}
	infrastructure.Networks = networks
	p.log.Info("Discovered networks", "count", len(networks))

	// Discover Storage
	p.log.Info("Discovering storage")
	storage, err := p.DiscoverStorage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover storage: %w", err)
	}
	infrastructure.Storage = storage
	p.log.Info("Discovered storage", "count", len(storage))

	// Discover Resource Pools
	p.log.Info("Discovering resource pools")
	resourcePools, err := p.DiscoverResourcePools(ctx)
	if err != nil {
		p.log.Error("Failed to discover resource pools", "error", err)
		// Don't fail the entire discovery for resource pools
	} else {
		infrastructure.ResourcePools = resourcePools
		p.log.Info("Discovered resource pools", "count", len(resourcePools))
	}

	// Discover Templates
	p.log.Info("Discovering templates")
	templates, err := p.DiscoverTemplates(ctx)
	if err != nil {
		p.log.Error("Failed to discover templates", "error", err)
		// Don't fail the entire discovery for templates
	} else {
		infrastructure.Templates = templates
		p.log.Info("Discovered templates", "count", len(templates))
	}

	// Add metadata
	infrastructure.Metadata["total_resources"] = len(vms) + len(networks) + len(storage) + len(resourcePools) + len(templates)
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
	
	// Create property collector for efficient data retrieval
	pc := property.DefaultCollector(p.client.Client)
	
	// Define properties to retrieve
	properties := []string{
		"name",
		"summary",
		"config",
		"guest",
		"runtime",
		"datastore",
		"network",
		"resourcePool",
		"parent",
	}

	// Collect properties for all VMs
	var refs []types.ManagedObjectReference
	for _, vm := range vms {
		refs = append(refs, vm.Reference())
	}

	var moVMs []mo.VirtualMachine
	err = pc.Retrieve(ctx, refs, properties, &moVMs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VM properties: %w", err)
	}

	// Convert to our VM model
	for _, moVM := range moVMs {
		// Skip templates unless specifically requested
		if moVM.Config != nil && moVM.Config.Template && !filters.IncludeTemplates {
			continue
		}

		vm := p.convertVMToModel(moVM)
		
		// Apply filters
		if p.vmMatchesFilters(vm, filters) {
			vmList = append(vmList, vm)
		}
	}

	return vmList, nil
}

// convertVMToModel converts a govmomi VM object to our VM model
func (p *vmwareProvider) convertVMToModel(moVM mo.VirtualMachine) models.VirtualMachine {
	vm := models.VirtualMachine{
		ID:       moVM.Reference().Value,
		Name:     moVM.Name,
		State:    string(moVM.Runtime.PowerState),
		PowerState: string(moVM.Runtime.PowerState),
		Metadata: make(map[string]interface{}),
	}

	// Basic configuration
	if moVM.Config != nil {
		vm.CPUs = int(moVM.Config.Hardware.NumCPU)
		vm.Memory = int64(moVM.Config.Hardware.MemoryMB)
		vm.Config = models.VMConfig{
			Template:     moVM.Config.Template,
			GuestID:      moVM.Config.GuestId,
			UUID:         moVM.Config.Uuid,
			InstanceUUID: moVM.Config.InstanceUuid,
			ChangeVersion: moVM.Config.ChangeVersion,
		}
		
		// Handle Modified time safely
		if !moVM.Config.Modified.IsZero() {
			vm.Config.Modified = moVM.Config.Modified
		}

		vm.Hardware = models.HardwareInfo{
			Version:           moVM.Config.Version,
			NumCPU:           int(moVM.Config.Hardware.NumCPU),
			NumCoresPerSocket: int(moVM.Config.Hardware.NumCoresPerSocket),
			MemoryMB:         int64(moVM.Config.Hardware.MemoryMB),
			Firmware:         moVM.Config.Firmware,
		}

		// Parse annotations
		if moVM.Config.Annotation != "" {
			vm.Annotations = map[string]string{
				"notes": moVM.Config.Annotation,
			}
		}
	}

	// Guest information
	if moVM.Guest != nil {
		vm.OperatingSystem = moVM.Guest.GuestFullName
		if moVM.Guest.ToolsStatus != "" {
			vm.Tools = models.VMTools{
				Status:        string(moVM.Guest.ToolsStatus),
				Version:       moVM.Guest.ToolsVersion,
				RunningStatus: string(moVM.Guest.ToolsRunningStatus),
			}
		}
	}

	// Summary information
	if moVM.Summary.Config.GuestFullName != "" {
		if vm.OperatingSystem == "" {
			vm.OperatingSystem = moVM.Summary.Config.GuestFullName
		}
	}

	// Resource pool
	if moVM.ResourcePool != nil {
		vm.ResourcePool = moVM.ResourcePool.Value
	}

	// Parent folder
	if moVM.Parent != nil {
		vm.Folder = moVM.Parent.Value
	}

	// Host
	if moVM.Runtime.Host != nil {
		vm.Host = moVM.Runtime.Host.Value
	}

	// Process disks
	if moVM.Config != nil && moVM.Config.Hardware.Device != nil {
		vm.Disks = p.extractDisks(moVM.Config.Hardware.Device)
		vm.NetworkCards = p.extractNetworkCards(moVM.Config.Hardware.Device)
	}

	return vm
}

// extractDisks extracts disk information from VM hardware devices
func (p *vmwareProvider) extractDisks(devices []types.BaseVirtualDevice) []models.Disk {
	var disks []models.Disk

	for _, device := range devices {
		if disk, ok := device.(*types.VirtualDisk); ok {
			diskModel := models.Disk{
				ID:   fmt.Sprintf("%d", disk.Key),
				Size: disk.CapacityInKB / 1024 / 1024, // Convert to GB
			}

			// Get backing information
			if backing := disk.Backing; backing != nil {
				switch b := backing.(type) {
				case *types.VirtualDiskFlatVer2BackingInfo:
					diskModel.Path = b.FileName
					if b.ThinProvisioned != nil && *b.ThinProvisioned {
						diskModel.Type = "thin"
					} else {
						diskModel.Type = "thick"
					}
					if b.Datastore != nil {
						diskModel.Datastore = b.Datastore.Value
					}
				case *types.VirtualDiskSparseVer2BackingInfo:
					diskModel.Path = b.FileName
					diskModel.Type = "sparse"
				}
			}

			// Get controller information
			if controllerKey := disk.ControllerKey; controllerKey != 0 {
				diskModel.Controller = fmt.Sprintf("%d", controllerKey)
				if disk.UnitNumber != nil {
					diskModel.Unit = int(*disk.UnitNumber)
				}
			}

			disks = append(disks, diskModel)
		}
	}

	return disks
}

// extractNetworkCards extracts network card information from VM hardware devices
func (p *vmwareProvider) extractNetworkCards(devices []types.BaseVirtualDevice) []models.NetworkCard {
	var networkCards []models.NetworkCard

	for _, device := range devices {
		if nic, ok := device.(types.BaseVirtualEthernetCard); ok {
			card := models.NetworkCard{
				ID:        fmt.Sprintf("%d", nic.GetVirtualEthernetCard().Key),
				Connected: nic.GetVirtualEthernetCard().Connectable.Connected,
				StartConnect: nic.GetVirtualEthernetCard().Connectable.StartConnected,
			}

			// Get MAC address
			if mac := nic.GetVirtualEthernetCard().MacAddress; mac != "" {
				card.MACAddress = mac
			}

			// Get network adapter type
			switch nic.(type) {
			case *types.VirtualVmxnet3:
				card.Type = "vmxnet3"
			case *types.VirtualE1000:
				card.Type = "e1000"
			case *types.VirtualE1000e:
				card.Type = "e1000e"
			case *types.VirtualPCNet32:
				card.Type = "pcnet32"
			default:
				card.Type = "unknown"
			}

			// Get network backing
			if backing := nic.GetVirtualEthernetCard().Backing; backing != nil {
				switch b := backing.(type) {
				case *types.VirtualEthernetCardNetworkBackingInfo:
					card.Network = b.DeviceName
				case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
					if b.Port.PortgroupKey != "" {
						card.Network = b.Port.PortgroupKey
					}
				}
			}

			networkCards = append(networkCards, card)
		}
	}

	return networkCards
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

	// Host filter
	if filters.Host != "" && vm.Host != filters.Host {
		return false
	}

	// Resource pool filter
	if filters.ResourcePool != "" && vm.ResourcePool != filters.ResourcePool {
		return false
	}

	// Folder filter
	if filters.Folder != "" && vm.Folder != filters.Folder {
		return false
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

	// Get properties for all datastores
	var refs []types.ManagedObjectReference
	for _, ds := range datastores {
		refs = append(refs, ds.Reference())
	}

	var moDatastores []mo.Datastore
	pc := property.DefaultCollector(p.client.Client)
	err = pc.Retrieve(ctx, refs, []string{"name", "summary", "info"}, &moDatastores)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve datastore properties: %w", err)
	}

	for _, moDS := range moDatastores {
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

		// Get URL if available
		if moDS.Summary.Url != "" {
			storage.URL = moDS.Summary.Url
		}

		// Check if it's local storage
		if moDS.Info != nil {
			if vmfsInfo, ok := moDS.Info.(*types.VmfsDatastoreInfo); ok {
				storage.Local = vmfsInfo.Vmfs.Local != nil && *vmfsInfo.Vmfs.Local
				storage.SSD = vmfsInfo.Vmfs.Ssd != nil && *vmfsInfo.Vmfs.Ssd
			}
		}

		storageList = append(storageList, storage)
	}

	return storageList, nil
}

// DiscoverResourcePools discovers resource pools
func (p *vmwareProvider) DiscoverResourcePools(ctx context.Context) ([]models.ResourcePool, error) {
	// Find all resource pools
	pools, err := p.finder.ResourcePoolList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list resource pools: %w", err)
	}

	var poolList []models.ResourcePool

	for _, pool := range pools {
		rp := models.ResourcePool{
			ID:       pool.Reference().Value,
			Name:     pool.InventoryPath,
			Metadata: make(map[string]interface{}),
		}

		// Get resource pool configuration
		var moRP mo.ResourcePool
		err := pool.Properties(ctx, pool.Reference(), []string{"config", "parent", "vm"}, &moRP)
		if err != nil {
			p.log.Error("Failed to get resource pool properties", "pool", pool.Name(), "error", err)
			continue
		}

		// Handle config safely
		if moRP.Config.CpuAllocation != nil {
			rp.CPU = models.ResourceAllocation{
				Reservation: moRP.Config.CpuAllocation.Reservation,
				Limit:       moRP.Config.CpuAllocation.Limit,
				Shares:      string(moRP.Config.CpuAllocation.Shares.Level),
				SharesValue: moRP.Config.CpuAllocation.Shares.Shares,
			}
		}

		// Memory allocation
		if moRP.Config.MemoryAllocation != nil {
			rp.Memory = models.ResourceAllocation{
				Reservation: moRP.Config.MemoryAllocation.Reservation,
				Limit:       moRP.Config.MemoryAllocation.Limit,
				Shares:      string(moRP.Config.MemoryAllocation.Shares.Level),
				SharesValue: moRP.Config.MemoryAllocation.Shares.Shares,
			}
		}

		// Parent resource pool
		if moRP.Parent != nil {
			rp.Parent = moRP.Parent.Value
		}

		// VMs in this resource pool
		if moRP.Vm != nil {
			for _, vmRef := range moRP.Vm {
				rp.VMs = append(rp.VMs, vmRef.Value)
			}
		}

		poolList = append(poolList, rp)
	}

	return poolList, nil
}

// DiscoverTemplates discovers VM templates
func (p *vmwareProvider) DiscoverTemplates(ctx context.Context) ([]models.Template, error) {
	// Find all VMs (including templates)
	vms, err := p.finder.VirtualMachineList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs for template discovery: %w", err)
	}

	var templateList []models.Template

	for _, vm := range vms {
		var moVM mo.VirtualMachine
		err := vm.Properties(ctx, vm.Reference(), []string{"config", "summary"}, &moVM)
		if err != nil {
			continue
		}

		// Skip if not a template
		if moVM.Config == nil || !moVM.Config.Template {
			continue
		}

		template := models.Template{
			ID:       vm.Reference().Value,
			Name:     moVM.Name,
			CPUs:     int(moVM.Config.Hardware.NumCPU),
			Memory:   int64(moVM.Config.Hardware.MemoryMB),
			Metadata: make(map[string]interface{}),
		}

		if moVM.Summary.Config.GuestFullName != "" {
			template.OperatingSystem = moVM.Summary.Config.GuestFullName
		}

		// Parse annotations
		if moVM.Config.Annotation != "" {
			template.Annotations = map[string]string{
				"notes": moVM.Config.Annotation,
			}
		}

		// Extract disks and network cards (similar to VM discovery)
		if moVM.Config.Hardware.Device != nil {
			template.Disks = p.extractDisks(moVM.Config.Hardware.Device)
			template.NetworkCards = p.extractNetworkCards(moVM.Config.Hardware.Device)
		}

		templateList = append(templateList, template)
	}

	return templateList, nil
}

// DiscoverDatacenters discovers all datacenters
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

// DiscoverClusters discovers clusters in a datacenter
func (p *vmwareProvider) DiscoverClusters(ctx context.Context, datacenter string) ([]models.Cluster, error) {
	clusters, err := p.finder.ClusterComputeResourceList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	var clusterList []models.Cluster
	for _, cluster := range clusters {
		var moCluster mo.ClusterComputeResource
		err := cluster.Properties(ctx, cluster.Reference(), []string{"name", "summary", "host", "resourcePool"}, &moCluster)
		if err != nil {
			continue
		}

		clusterModel := models.Cluster{
			ID:         cluster.Reference().Value,
			Name:       moCluster.Name,
			Datacenter: datacenter,
			Metadata:   make(map[string]interface{}),
		}

		if moCluster.Summary != nil {
			clusterModel.TotalCPU = int64(moCluster.Summary.TotalCpu)
			clusterModel.TotalMemory = int64(moCluster.Summary.TotalMemory)
			clusterModel.UsedCPU = int64(moCluster.Summary.TotalCpu - moCluster.Summary.EffectiveCpu)
			clusterModel.UsedMemory = int64(moCluster.Summary.TotalMemory - moCluster.Summary.EffectiveMemory)
		}

		// Add hosts
		for _, hostRef := range moCluster.Host {
			clusterModel.Hosts = append(clusterModel.Hosts, hostRef.Value)
		}

		clusterList = append(clusterList, clusterModel)
	}

	return clusterList, nil
}

// DiscoverHosts discovers hosts in a cluster
func (p *vmwareProvider) DiscoverHosts(ctx context.Context, cluster string) ([]models.Host, error) {
	hosts, err := p.finder.HostSystemList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list hosts: %w", err)
	}

	var hostList []models.Host
	for _, host := range hosts {
		var moHost mo.HostSystem
		err := host.Properties(ctx, host.Reference(), []string{"name", "summary", "vm", "datastore", "network"}, &moHost)
		if err != nil {
			continue
		}

		hostModel := models.Host{
			ID:              host.Reference().Value,
			Name:            moHost.Name,
			Type:            "ESXi",
			State:           string(moHost.Summary.Runtime.PowerState),
			ConnectionState: string(moHost.Summary.Runtime.ConnectionState),
			Cluster:         cluster,
			Metadata:        make(map[string]interface{}),
		}

		if moHost.Summary.Config != nil {
			hostModel.Version = moHost.Summary.Config.Product.Version
		}

		if moHost.Summary.Hardware != nil {
			hostModel.CPU = models.HostResource{
				Total: int64(moHost.Summary.Hardware.CpuMhz) * int64(moHost.Summary.Hardware.NumCpuCores),
			}
			hostModel.Memory = models.HostResource{
				Total: int64(moHost.Summary.Hardware.MemorySize / 1024 / 1024), // Convert to MB
			}
		}

		if moHost.Summary.QuickStats != nil {
			hostModel.CPU.Used = int64(moHost.Summary.QuickStats.OverallCpuUsage)
			hostModel.Memory.Used = int64(moHost.Summary.QuickStats.OverallMemoryUsage)
		}

		// Add VMs
		for _, vmRef := range moHost.Vm {
			hostModel.VMs = append(hostModel.VMs, vmRef.Value)
		}

		hostList = append(hostList, hostModel)
	}

	return hostList, nil
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
	return fmt.Errorf("use Connect(ctx, config.VMwareConfig) instead")
}
