package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"valhalla/internal/config"
	"valhalla/internal/discovery/providers"
	"valhalla/internal/logger"
	"valhalla/internal/models"
)

// Engine orchestrates infrastructure discovery across multiple providers
type Engine struct {
	log      *logger.Logger
	config   *config.Config
	providers map[string]providers.Provider
	mu       sync.RWMutex
}

// NewEngine creates a new discovery engine
func NewEngine(log *logger.Logger, cfg *config.Config) *Engine {
	return &Engine{
		log:       log,
		config:    cfg,
		providers: make(map[string]providers.Provider),
	}
}

// DiscoverVMware discovers VMware vSphere infrastructure
func (e *Engine) DiscoverVMware(ctx context.Context, cfg config.VMwareConfig) ([]*models.Infrastructure, error) {
	e.log.Info("Starting VMware discovery", "server", cfg.Server)

	// Create VMware provider
	provider := providers.NewVMwareProvider(e.log)
	
	// Connect to vCenter
	if err := provider.ConnectVMware(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to connect to VMware: %w", err)
	}
	defer provider.Disconnect()

	// Perform discovery
	infrastructure, err := provider.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("VMware discovery failed: %w", err)
	}

	return []*models.Infrastructure{infrastructure}, nil
}

// DiscoverProxmox discovers Proxmox infrastructure
func (e *Engine) DiscoverProxmox(ctx context.Context, cfg config.ProxmoxConfig) ([]*models.Infrastructure, error) {
	e.log.Info("Starting Proxmox discovery", "server", cfg.Server)

	// TODO: Implement Proxmox provider
	e.log.Warn("Proxmox provider not yet implemented")
	
	// Create placeholder infrastructure
	infrastructure := &models.Infrastructure{
		Provider:      "proxmox",
		Server:        cfg.Server,
		Node:          cfg.Node,
		DiscoveryTime: time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	infrastructure.Metadata["status"] = "not_implemented"
	infrastructure.Metadata["message"] = "Proxmox discovery will be implemented in next phase"

	return []*models.Infrastructure{infrastructure}, nil
}

// DiscoverNutanix discovers Nutanix infrastructure
func (e *Engine) DiscoverNutanix(ctx context.Context, cfg config.NutanixConfig) ([]*models.Infrastructure, error) {
	e.log.Info("Starting Nutanix discovery", "server", cfg.Server)

	// TODO: Implement Nutanix provider
	e.log.Warn("Nutanix provider not yet implemented")
	
	// Create placeholder infrastructure
	infrastructure := &models.Infrastructure{
		Provider:      "nutanix",
		Server:        cfg.Server,
		Cluster:       cfg.Cluster,
		DiscoveryTime: time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	infrastructure.Metadata["status"] = "not_implemented"
	infrastructure.Metadata["message"] = "Nutanix discovery will be implemented in next phase"

	return []*models.Infrastructure{infrastructure}, nil
}

// DiscoverAll discovers infrastructure from all configured providers
func (e *Engine) DiscoverAll(ctx context.Context) ([]*models.Infrastructure, error) {
	e.log.Info("Starting multi-provider discovery")

	var allResults []*models.Infrastructure
	var errors []error

	// Discover VMware if configured
	vmwareConfig := e.config.GetVMwareConfig()
	if vmwareConfig.Server != "" {
		results, err := e.DiscoverVMware(ctx, vmwareConfig)
		if err != nil {
			errors = append(errors, fmt.Errorf("VMware discovery failed: %w", err))
		} else {
			allResults = append(allResults, results...)
		}
	}

	// Discover Proxmox if configured
	proxmoxConfig := e.config.GetProxmoxConfig()
	if proxmoxConfig.Server != "" {
		results, err := e.DiscoverProxmox(ctx, proxmoxConfig)
		if err != nil {
			errors = append(errors, fmt.Errorf("Proxmox discovery failed: %w", err))
		} else {
			allResults = append(allResults, results...)
		}
	}

	// Discover Nutanix if configured
	nutanixConfig := e.config.GetNutanixConfig()
	if nutanixConfig.Server != "" {
		results, err := e.DiscoverNutanix(ctx, nutanixConfig)
		if err != nil {
			errors = append(errors, fmt.Errorf("Nutanix discovery failed: %w", err))
		} else {
			allResults = append(allResults, results...)
		}
	}

	// Handle errors
	if len(errors) > 0 && len(allResults) == 0 {
		return nil, fmt.Errorf("all provider discoveries failed: %v", errors)
	}

	if len(errors) > 0 {
		e.log.Warn("Some provider discoveries failed", "errors", len(errors))
		for _, err := range errors {
			e.log.Error("Provider discovery error", "error", err)
		}
	}

	e.log.Info("Multi-provider discovery completed", 
		"total_infrastructures", len(allResults),
		"failed_providers", len(errors))

	return allResults, nil
}

// RegisterProvider registers a custom provider
func (e *Engine) RegisterProvider(name string, provider providers.Provider) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.providers[name] = provider
	e.log.Info("Registered custom provider", "name", name)
}

// GetProvider returns a registered provider
func (e *Engine) GetProvider(name string) (providers.Provider, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	provider, exists := e.providers[name]
	return provider, exists
}

// GetRegisteredProviders returns all registered provider names
func (e *Engine) GetRegisteredProviders() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	var names []string
	for name := range e.providers {
		names = append(names, name)
	}
	return names
}

// ValidateProviderConfig validates provider configurations
func (e *Engine) ValidateProviderConfig(provider string) error {
	switch provider {
	case "vmware", "vsphere":
		cfg := e.config.GetVMwareConfig()
		if cfg.Server == "" {
			return fmt.Errorf("VMware server not configured")
		}
		if cfg.Username == "" {
			return fmt.Errorf("VMware username not configured")
		}
		if cfg.Password == "" {
			return fmt.Errorf("VMware password not configured")
		}
	case "proxmox":
		cfg := e.config.GetProxmoxConfig()
		if cfg.Server == "" {
			return fmt.Errorf("Proxmox server not configured")
		}
		if cfg.Username == "" {
			return fmt.Errorf("Proxmox username not configured")
		}
		if cfg.Password == "" && (cfg.TokenID == "" || cfg.Secret == "") {
			return fmt.Errorf("Proxmox password or API token not configured")
		}
	case "nutanix":
		cfg := e.config.GetNutanixConfig()
		if cfg.Server == "" {
			return fmt.Errorf("Nutanix server not configured")
		}
		if cfg.Username == "" {
			return fmt.Errorf("Nutanix username not configured")
		}
		if cfg.Password == "" {
			return fmt.Errorf("Nutanix password not configured")
		}
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
	
	return nil
}

// GetSupportedProviders returns list of supported providers
func (e *Engine) GetSupportedProviders() []string {
	return []string{"vmware", "proxmox", "nutanix"}
}
