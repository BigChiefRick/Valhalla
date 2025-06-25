package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"valhalla/internal/logger"
)

// Validator provides validation functionality for various file types
type Validator struct {
	log *logger.Logger
}

// NewValidator creates a new validator
func NewValidator(log *logger.Logger) *Validator {
	return &Validator{
		log: log,
	}
}

// ValidateOptions holds options for validation
type ValidateOptions struct {
	Format    string `json:"format"`
	Recursive bool   `json:"recursive"`
	Fix       bool   `json:"fix"`
	Strict    bool   `json:"strict"`
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Path     string            `json:"path"`
	Valid    bool              `json:"valid"`
	Issues   []*ValidationIssue `json:"issues"`
	Fixed    int               `json:"fixed"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ValidationIssue represents a validation issue
type ValidationIssue struct {
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // error, warning, info
	Rule     string `json:"rule"`
	Fixed    bool   `json:"fixed"`
}

// ValidateFile validates a single file
func (v *Validator) ValidateFile(path string, opts ValidateOptions) (*ValidationResult, error) {
	v.log.Info("Validating file", "path", path)

	// Determine format if auto
	format := opts.Format
	if format == "auto" {
		format = v.detectFormat(path)
	}

	result := &ValidationResult{
		Path:     path,
		Valid:    true,
		Issues:   []*ValidationIssue{},
		Metadata: make(map[string]interface{}),
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", path)
	}

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Validate based on format
	switch format {
	case "terraform", "tf":
		v.validateTerraform(string(content), result, opts)
	case "json":
		v.validateJSON(string(content), result, opts)
	case "yaml", "yml":
		v.validateYAML(string(content), result, opts)
	case "pulumi":
		v.validatePulumi(string(content), result, opts)
	case "ansible":
		v.validateAnsible(string(content), result, opts)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	// Apply fixes if requested and possible
	if opts.Fix && result.Fixed > 0 {
		// TODO: Implement fix application
		v.log.Info("Applied fixes", "file", path, "fixes", result.Fixed)
	}

	result.Valid = len(result.Issues) == 0 || v.onlyWarnings(result.Issues)
	return result, nil
}

// ValidateDirectory validates all files in a directory
func (v *Validator) ValidateDirectory(dirPath string, opts ValidateOptions) ([]*ValidationResult, error) {
	v.log.Info("Validating directory", "path", dirPath, "recursive", opts.Recursive)

	var results []*ValidationResult
	
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip if not recursive and not the root directory
			if !opts.Recursive && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file should be validated
		if v.shouldValidateFile(path, opts.Format) {
			result, err := v.ValidateFile(path, opts)
			if err != nil {
				v.log.Error("Failed to validate file", "path", path, "error", err)
				return nil // Continue with other files
			}
			results = append(results, result)
		}

		return nil
	}

	if err := filepath.Walk(dirPath, walkFunc); err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return results, nil
}

// detectFormat detects the file format based on extension and content
func (v *Validator) detectFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	switch ext {
	case ".tf", ".hcl":
		return "terraform"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".py":
		// Check if it's a Pulumi Python file
		if strings.Contains(path, "pulumi") {
			return "pulumi"
		}
		return "python"
	case ".ts", ".js":
		if strings.Contains(path, "pulumi") {
			return "pulumi"
		}
		return "typescript"
	case ".go":
		if strings.Contains(path, "pulumi") {
			return "pulumi"
		}
		return "go"
	case ".cs":
		if strings.Contains(path, "pulumi") {
			return "pulumi"
		}
		return "csharp"
	default:
		return "unknown"
	}
}

// shouldValidateFile determines if a file should be validated
func (v *Validator) shouldValidateFile(path string, format string) bool {
	if format != "auto" && format != "unknown" {
		return v.detectFormat(path) == format
	}

	// Validate common IaC and config files
	detectedFormat := v.detectFormat(path)
	supportedFormats := []string{"terraform", "json", "yaml", "pulumi"}
	
	for _, supported := range supportedFormats {
		if detectedFormat == supported {
			return true
		}
	}

	return false
}

// onlyWarnings checks if all issues are warnings (not errors)
func (v *Validator) onlyWarnings(issues []*ValidationIssue) bool {
	for _, issue := range issues {
		if issue.Severity == "error" {
			return false
		}
	}
	return true
}

// Validation functions for specific formats
func (v *Validator) validateTerraform(content string, result *ValidationResult, opts ValidateOptions) {
	// Basic Terraform syntax validation
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for common Terraform issues
		if strings.Contains(line, "source = \"") && !strings.Contains(line, "version = ") {
			result.Issues = append(result.Issues, &ValidationIssue{
				Line:     i + 1,
				Message:  "Module source should include version constraint",
				Severity: "warning",
				Rule:     "terraform-module-version",
			})
		}

		// Check for hardcoded values
		if strings.Contains(line, "ami-") && !strings.Contains(line, "var.") {
			result.Issues = append(result.Issues, &ValidationIssue{
				Line:     i + 1,
				Message:  "Hardcoded AMI ID should be parameterized",
				Severity: "warning",
				Rule:     "terraform-hardcoded-ami",
			})
		}
	}
}

func (v *Validator) validateJSON(content string, result *ValidationResult, opts ValidateOptions) {
	// Basic JSON syntax validation
	if !strings.HasPrefix(strings.TrimSpace(content), "{") && !strings.HasPrefix(strings.TrimSpace(content), "[") {
		result.Issues = append(result.Issues, &ValidationIssue{
			Line:     1,
			Message:  "Invalid JSON format",
			Severity: "error",
			Rule:     "json-syntax",
		})
	}

	// Check for common JSON issues
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, "\\") && !strings.Contains(line, "\\\\") {
			result.Issues = append(result.Issues, &ValidationIssue{
				Line:     i + 1,
				Message:  "Unescaped backslash in JSON",
				Severity: "warning",
				Rule:     "json-escape",
			})
		}
	}
}

func (v *Validator) validateYAML(content string, result *ValidationResult, opts ValidateOptions) {
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		// Check for tab characters (YAML should use spaces)
		if strings.Contains(line, "\t") {
			result.Issues = append(result.Issues, &ValidationIssue{
				Line:     i + 1,
				Message:  "YAML should use spaces instead of tabs",
				Severity: "error",
				Rule:     "yaml-tabs",
			})
		}
	}
}

func (v *Validator) validatePulumi(content string, result *ValidationResult, opts ValidateOptions) {
	// Basic Pulumi validation
	if !strings.Contains(content, "import pulumi") && !strings.Contains(content, "import * as pulumi") {
		result.Issues = append(result.Issues, &ValidationIssue{
			Line:     1,
			Message:  "Pulumi program should import pulumi",
			Severity: "warning",
			Rule:     "pulumi-import",
		})
	}
}

func (v *Validator) validateAnsible(content string, result *ValidationResult, opts ValidateOptions) {
	// Basic Ansible playbook validation
	if !strings.Contains(content, "hosts:") && !strings.Contains(content, "- name:") {
		result.Issues = append(result.Issues, &ValidationIssue{
			Line:     1,
			Message:  "Ansible playbook should contain hosts or tasks",
			Severity: "warning",
			Rule:     "ansible-structure",
		})
	}
}
