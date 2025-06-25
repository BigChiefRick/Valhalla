package generators

import (
	"fmt"
	"strings"

	"valhalla/internal/logger"
	"valhalla/internal/models"
)

// Generator defines the interface for IaC generators
type Generator interface {
	// Generate creates IaC templates from infrastructure models
	Generate(infrastructures []*models.Infrastructure, opts GenerateOptions) ([]*GenerateResult, error)
	
	// GetName returns the generator name
	GetName() string
	
	// GetSupportedFormats returns supported output formats
	GetSupportedFormats() []string
	
	// Validate validates the generated templates
	Validate(results []*GenerateResult) error
}

// GenerateOptions holds options for IaC generation
type GenerateOptions struct {
	OutputDir    string            `json:"output_dir"`
	DryRun       bool              `json:"dry_run"`
	Validate     bool              `json:"validate"`
	Variables    map[string]string `json:"variables,omitempty"`
	Templates    map[string]string `json:"templates,omitempty"`
	Overwrite    bool              `json:"overwrite"`
	FormatCode   bool              `json:"format_code"`
	AddComments  bool              `json:"add_comments"`
	Modular      bool              `json:"modular"`
}

// GenerateResult represents the result of IaC generation
type GenerateResult struct {
	Path      string                 `json:"path"`
	Content   []byte                 `json:"content"`
	Size      int                    `json:"size"`
	Type      string                 `json:"type"` // main, variables, outputs, modules
	Provider  string                 `json:"provider"`
	Resources []string               `json:"resources"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewGenerator creates a new generator based on the format
func NewGenerator(format string, log *logger.Logger) (Generator, error) {
	switch strings.ToLower(format) {
	case "terraform", "tf":
		return NewTerraformGenerator(log), nil
	case "pulumi-python":
		return NewPulumiGenerator("python", log), nil
	case "pulumi-typescript", "pulumi-ts":
		return NewPulumiGenerator("typescript", log), nil
	case "pulumi-go":
		return NewPulumiGenerator("go", log), nil
	case "pulumi-csharp", "pulumi-cs":
		return NewPulumiGenerator("csharp", log), nil
	case "ansible":
		return NewAnsibleGenerator(log), nil
	default:
		return nil, fmt.Errorf("unsupported generator format: %s", format)
	}
}

// GetAvailableFormats returns all available generator formats
func GetAvailableFormats() []string {
	return []string{
		"terraform",
		"pulumi-python",
		"pulumi-typescript",
		"pulumi-go",
		"pulumi-csharp",
		"ansible",
	}
}

// BaseGenerator provides common functionality for all generators
type BaseGenerator struct {
	log    *logger.Logger
	name   string
	format string
}

// NewBaseGenerator creates a new base generator
func NewBaseGenerator(name, format string, log *logger.Logger) *BaseGenerator {
	return &BaseGenerator{
		log:    log,
		name:   name,
		format: format,
	}
}

// GetName returns the generator name
func (g *BaseGenerator) GetName() string {
	return g.name
}

// GetFormat returns the generator format
func (g *BaseGenerator) GetFormat() string {
	return g.format
}

// Log returns the logger
func (g *BaseGenerator) Log() *logger.Logger {
	return g.log
}

// FilterInfrastructureByProvider filters infrastructures by provider
func (g *BaseGenerator) FilterInfrastructureByProvider(infrastructures []*models.Infrastructure, provider string) []*models.Infrastructure {
	if provider == "" {
		return infrastructures
	}

	var filtered []*models.Infrastructure
	for _, infra := range infrastructures {
		if strings.EqualFold(infra.Provider, provider) {
			filtered = append(filtered, infra)
		}
	}
	return filtered
}

// GenerateResourceName creates a valid resource name from a given name
func (g *BaseGenerator) GenerateResourceName(name string) string {
	// Replace invalid characters with underscores
	resourceName := strings.ReplaceAll(name, " ", "_")
	resourceName = strings.ReplaceAll(resourceName, "-", "_")
	resourceName = strings.ReplaceAll(resourceName, ".", "_")
	resourceName = strings.ToLower(resourceName)
	
	// Ensure it starts with a letter
	if len(resourceName) > 0 && (resourceName[0] < 'a' || resourceName[0] > 'z') {
		resourceName = "res_" + resourceName
	}
	
	return resourceName
}

// SanitizeValue sanitizes a value for use in generated code
func (g *BaseGenerator) SanitizeValue(value string) string {
	// Escape quotes and other special characters
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, `\`, `\\`)
	return value
}

// ResourceCounter tracks resource counts for naming
type ResourceCounter struct {
	counts map[string]int
}

// NewResourceCounter creates a new resource counter
func NewResourceCounter() *ResourceCounter {
	return &ResourceCounter{
		counts: make(map[string]int),
	}
}

// GetNext returns the next number for a resource type
func (rc *ResourceCounter) GetNext(resourceType string) int {
	rc.counts[resourceType]++
	return rc.counts[resourceType]
}

// GetUniqueName generates a unique resource name
func (rc *ResourceCounter) GetUniqueName(resourceType, baseName string) string {
	count := rc.GetNext(resourceType)
	if count == 1 {
		return baseName
	}
	return fmt.Sprintf("%s_%d", baseName, count)
}
