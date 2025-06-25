package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"valhalla/internal/config"
	"valhalla/internal/generators"
	"valhalla/internal/logger"
	"valhalla/internal/models"
)

// GenerateOptions holds options for the generate command
type GenerateOptions struct {
	InputFile    string
	OutputFormat string
	OutputDir    string
	Provider     string
	DryRun       bool
	Validate     bool
}

// NewGenerateCmd creates the generate command
func NewGenerateCmd(log *logger.Logger, cfg *config.Config) *cobra.Command {
	opts := &GenerateOptions{}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate Infrastructure as Code from discovery results",
		Long: `Generate Infrastructure as Code templates from discovered infrastructure.

Supports multiple output formats:
- Terraform HCL (.tf files)
- Pulumi (Python, TypeScript, Go, C#)
- Ansible playbooks
- Custom templates

Examples:
  # Generate Terraform from discovery results
  valhalla generate --input discovery.json --format terraform --output-dir ./terraform
  
  # Generate Pulumi TypeScript
  valhalla generate --input discovery.json --format pulumi-typescript --output-dir ./pulumi
  
  # Generate for specific provider only
  valhalla generate --input discovery.json --provider vmware --format terraform`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(log, cfg, opts)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&opts.InputFile, "input", "i", "", "Input file with discovery results (JSON)")
	cmd.Flags().StringVarP(&opts.OutputFormat, "format", "f", "terraform", "Output format (terraform, pulumi-python, pulumi-typescript, pulumi-go, pulumi-csharp, ansible)")
	cmd.Flags().StringVarP(&opts.OutputDir, "output-dir", "o", "./output", "Output directory for generated files")
	cmd.Flags().StringVarP(&opts.Provider, "provider", "p", "", "Filter by provider (vmware, proxmox, nutanix)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be generated without creating files")
	cmd.Flags().BoolVar(&opts.Validate, "validate", true, "Validate generated templates")

	// Mark required flags
	cmd.MarkFlagRequired("input")

	return cmd
}

// runGenerate executes the IaC generation process
func runGenerate(log *logger.Logger, cfg *config.Config, opts *GenerateOptions) error {
	log.StartOperation("IaC generation", "format", opts.OutputFormat, "input", opts.InputFile)

	// Read discovery results
	log.Info("Reading discovery results", "file", opts.InputFile)
	infrastructures, err := readDiscoveryResults(opts.InputFile)
	if err != nil {
		return fmt.Errorf("failed to read discovery results: %w", err)
	}

	// Filter by provider if specified
	if opts.Provider != "" {
		infrastructures = filterByProvider(infrastructures, opts.Provider)
		if len(infrastructures) == 0 {
			return fmt.Errorf("no infrastructure found for provider: %s", opts.Provider)
		}
	}

	log.Info("Loaded infrastructure data", 
		"providers", getProviderCounts(infrastructures),
		"total_resources", getTotalResourceCount(infrastructures))

	// Create generator
	generator, err := generators.NewGenerator(opts.OutputFormat, log)
	if err != nil {
		return fmt.Errorf("failed to create generator: %w", err)
	}

	// Generate IaC templates
	log.Info("Generating IaC templates")
	results, err := generator.Generate(infrastructures, generators.GenerateOptions{
		OutputDir: opts.OutputDir,
		DryRun:    opts.DryRun,
		Validate:  opts.Validate,
	})
	if err != nil {
		log.FailOperation("IaC generation", err)
		return fmt.Errorf("generation failed: %w", err)
	}

	// Output results
	if opts.DryRun {
		log.Info("Dry run - showing what would be generated:")
		for _, result := range results {
			fmt.Printf("Would create: %s (%d bytes)\n", result.Path, result.Size)
		}
	} else {
		log.Info("Generated IaC templates", "files", len(results), "output_dir", opts.OutputDir)
		for _, result := range results {
			log.Info("Created file", "path", result.Path, "size_bytes", result.Size)
		}
	}

	log.CompleteOperation("IaC generation", "files_generated", len(results))
	return nil
}

// readDiscoveryResults reads and parses discovery results from a JSON file
func readDiscoveryResults(filename string) ([]*models.Infrastructure, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var infrastructures []*models.Infrastructure
	if err := json.Unmarshal(data, &infrastructures); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return infrastructures, nil
}

// filterByProvider filters infrastructures by provider type
func filterByProvider(infrastructures []*models.Infrastructure, provider string) []*models.Infrastructure {
	var filtered []*models.Infrastructure
	for _, infra := range infrastructures {
		if strings.EqualFold(infra.Provider, provider) {
			filtered = append(filtered, infra)
		}
	}
	return filtered
}

// getProviderCounts returns a map of provider names to resource counts
func getProviderCounts(infrastructures []*models.Infrastructure) map[string]int {
	counts := make(map[string]int)
	for _, infra := range infrastructures {
		count := len(infra.VirtualMachines) + len(infra.Networks) + len(infra.Storage)
		counts[infra.Provider] += count
	}
	return counts
}
