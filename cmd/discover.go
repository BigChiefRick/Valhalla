package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"valhalla/internal/config"
	"valhalla/internal/discovery"
	"valhalla/internal/logger"
	"valhalla/internal/models"
	"valhalla/internal/output"
)

// DiscoverOptions holds options for the discover command
type DiscoverOptions struct {
	Providers    []string
	OutputFormat string
	OutputFile   string
	Datacenter   string
	Cluster      string
	Node         string
	Concurrent   int
	Timeout      time.Duration
	DryRun       bool
}

// NewDiscoverCmd creates the discover command
func NewDiscoverCmd(log *logger.Logger, cfg *config.Config) *cobra.Command {
	opts := &DiscoverOptions{}

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover infrastructure from hypervisor environments",
		Long: `Discover and catalog infrastructure resources from VMware vCenter, Proxmox, and Nutanix environments.

Examples:
  # Discover VMware infrastructure
  valhalla discover --provider vmware --datacenter "Production DC"
  
  # Discover Proxmox infrastructure
  valhalla discover --provider proxmox --node "pve-01"
  
  # Discover all supported providers
  valhalla discover --provider vmware,proxmox,nutanix
  
  # Save results to file
  valhalla discover --provider vmware --output-file infrastructure.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiscover(log, cfg, opts)
		},
	}

	// Add flags
	cmd.Flags().StringSliceVarP(&opts.Providers, "provider", "p", []string{}, "Providers to discover (vmware, proxmox, nutanix)")
	cmd.Flags().StringVarP(&opts.OutputFormat, "format", "f", "table", "Output format (table, json, yaml)")
	cmd.Flags().StringVarP(&opts.OutputFile, "output-file", "o", "", "Output file path")
	cmd.Flags().StringVar(&opts.Datacenter, "datacenter", "", "VMware datacenter to discover")
	cmd.Flags().StringVar(&opts.Cluster, "cluster", "", "Cluster to discover")
	cmd.Flags().StringVar(&opts.Node, "node", "", "Proxmox node to discover")
	cmd.Flags().IntVar(&opts.Concurrent, "concurrent", 10, "Number of concurrent discovery operations")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 5*time.Minute, "Discovery timeout")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Perform a dry run without making API calls")

	// Mark required flags
	cmd.MarkFlagRequired("provider")

	return cmd
}

// runDiscover executes the discovery process
func runDiscover(log *logger.Logger, cfg *config.Config, opts *DiscoverOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	log.StartOperation("Infrastructure discovery", "providers", opts.Providers)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Initialize discovery engine
	engine := discovery.NewEngine(log, cfg)

	// Aggregate results from all providers
	var allResults []*models.Infrastructure

	// Discover from each provider
	for _, provider := range opts.Providers {
		providerLog := log.WithProvider(provider)
		
		if opts.DryRun {
			providerLog.Info("Dry run mode - skipping actual discovery")
			continue
		}

		providerLog.StartOperation("Provider discovery")

		switch strings.ToLower(provider) {
		case "vmware", "vsphere":
			results, err := opts.discoverVMware(ctx, engine, providerLog, cfg)
			if err != nil {
				providerLog.FailOperation("VMware discovery", err)
				return err
			}
			allResults = append(allResults, results...)

		case "proxmox":
			results, err := opts.discoverProxmox(ctx, engine, providerLog, cfg)
			if err != nil {
				providerLog.FailOperation("Proxmox discovery", err)
				return err
			}
			allResults = append(allResults, results...)

		case "nutanix":
			results, err := opts.discoverNutanix(ctx, engine, providerLog, cfg)
			if err != nil {
				providerLog.FailOperation("Nutanix discovery", err)
				return err
			}
			allResults = append(allResults, results...)

		default:
			return fmt.Errorf("unsupported provider: %s", provider)
		}

		providerLog.CompleteOperation("Provider discovery")
	}

	// Output results
	if err := outputResults(log, opts, allResults); err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	log.CompleteOperation("Infrastructure discovery", 
		"total_resources", getTotalResourceCount(allResults),
		"providers", len(opts.Providers))

	return nil
}

// getTotalResourceCount calculates total number of resources discovered
func getTotalResourceCount(results []*models.Infrastructure) int {
	total := 0
	for _, infra := range results {
		total += len(infra.VirtualMachines)
		total += len(infra.Networks)
		total += len(infra.Storage)
		total += len(infra.ResourcePools)
	}
	return total
}

// discoverVMware discovers VMware infrastructure
func (opts *DiscoverOptions) discoverVMware(ctx context.Context, engine *discovery.Engine, log *logger.Logger, cfg *config.Config) ([]*models.Infrastructure, error) {
	vmwareConfig := cfg.GetVMwareConfig()
	
	// Validate VMware configuration
	if vmwareConfig.Server == "" {
		return nil, fmt.Errorf("VMware server not configured")
	}

	// Override datacenter if specified
	if opts.Datacenter != "" {
		vmwareConfig.Datacenter = opts.Datacenter
	}
	if opts.Cluster != "" {
		vmwareConfig.Cluster = opts.Cluster
	}

	log.Info("Connecting to VMware vCenter", "server", vmwareConfig.Server, "datacenter", vmwareConfig.Datacenter)

	return engine.DiscoverVMware(ctx, vmwareConfig)
}

// discoverProxmox discovers Proxmox infrastructure
func (opts *DiscoverOptions) discoverProxmox(ctx context.Context, engine *discovery.Engine, log *logger.Logger, cfg *config.Config) ([]*models.Infrastructure, error) {
	proxmoxConfig := cfg.GetProxmoxConfig()
	
	// Validate Proxmox configuration
	if proxmoxConfig.Server == "" {
		return nil, fmt.Errorf("Proxmox server not configured")
	}

	// Override node if specified
	if opts.Node != "" {
		proxmoxConfig.Node = opts.Node
	}

	log.Info("Connecting to Proxmox", "server", proxmoxConfig.Server, "node", proxmoxConfig.Node)

	return engine.DiscoverProxmox(ctx, proxmoxConfig)
}

// discoverNutanix discovers Nutanix infrastructure
func (opts *DiscoverOptions) discoverNutanix(ctx context.Context, engine *discovery.Engine, log *logger.Logger, cfg *config.Config) ([]*models.Infrastructure, error) {
	nutanixConfig := cfg.GetNutanixConfig()
	
	// Validate Nutanix configuration
	if nutanixConfig.Server == "" {
		return nil, fmt.Errorf("Nutanix server not configured")
	}

	// Override cluster if specified
	if opts.Cluster != "" {
		nutanixConfig.Cluster = opts.Cluster
	}

	log.Info("Connecting to Nutanix", "server", nutanixConfig.Server, "cluster", nutanixConfig.Cluster)

	return engine.DiscoverNutanix(ctx, nutanixConfig)
}

// outputResults outputs discovery results in the specified format
func outputResults(log *logger.Logger, opts *DiscoverOptions, results []*models.Infrastructure) error {
	// Create output formatter
	formatter := output.NewFormatter(opts.OutputFormat)

	// Format results
	formattedOutput, err := formatter.Format(results)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Output to file or stdout
	if opts.OutputFile != "" {
		// Ensure output directory exists
		dir := strings.TrimSuffix(opts.OutputFile, "/"+strings.Split(opts.OutputFile, "/")[len(strings.Split(opts.OutputFile, "/"))-1])
		if dir != opts.OutputFile {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		// Write to file
		if err := os.WriteFile(opts.OutputFile, formattedOutput, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		log.Info("Results written to file", "file", opts.OutputFile)
	} else {
		// Write to stdout
		fmt.Print(string(formattedOutput))
	}

	return nil
}
