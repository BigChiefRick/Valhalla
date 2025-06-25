package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"valhalla/internal/config"
	"valhalla/internal/logger"
	"valhalla/internal/validation"
)

// ValidateOptions holds options for the validate command
type ValidateOptions struct {
	Path      string
	Format    string
	Recursive bool
	Fix       bool
	Strict    bool
}

// NewValidateCmd creates the validate command
func NewValidateCmd(log *logger.Logger, cfg *config.Config) *cobra.Command {
	opts := &ValidateOptions{}

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate generated Infrastructure as Code templates",
		Long: `Validate Infrastructure as Code templates for syntax, best practices, and compatibility.

Supports validation of:
- Terraform HCL files (.tf)
- Pulumi programs
- Ansible playbooks
- Discovery result files

Examples:
  # Validate Terraform files in a directory
  valhalla validate --path ./terraform --format terraform
  
  # Validate discovery results
  valhalla validate --path discovery.json --format json
  
  # Validate recursively with fixes
  valhalla validate --path ./output --recursive --fix`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Path = args[0]
			}
			return runValidate(log, cfg, opts)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&opts.Path, "path", "p", ".", "Path to validate (file or directory)")
	cmd.Flags().StringVarP(&opts.Format, "format", "f", "auto", "Format to validate (auto, terraform, pulumi, ansible, json)")
	cmd.Flags().BoolVarP(&opts.Recursive, "recursive", "r", false, "Validate recursively")
	cmd.Flags().BoolVar(&opts.Fix, "fix", false, "Attempt to fix validation issues")
	cmd.Flags().BoolVar(&opts.Strict, "strict", false, "Use strict validation rules")

	return cmd
}

// runValidate executes the validation process
func runValidate(log *logger.Logger, cfg *config.Config, opts *ValidateOptions) error {
	log.StartOperation("Validation", "path", opts.Path, "format", opts.Format)

	// Check if path exists
	if _, err := os.Stat(opts.Path); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", opts.Path)
	}

	// Determine if path is a file or directory
	fileInfo, err := os.Stat(opts.Path)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	validator := validation.NewValidator(log)

	var results []*validation.ValidationResult
	var validationErr error

	if fileInfo.IsDir() {
		// Validate directory
		results, validationErr = validator.ValidateDirectory(opts.Path, validation.ValidateOptions{
			Format:    opts.Format,
			Recursive: opts.Recursive,
			Fix:       opts.Fix,
			Strict:    opts.Strict,
		})
	} else {
		// Validate single file
		result, validationErr := validator.ValidateFile(opts.Path, validation.ValidateOptions{
			Format: opts.Format,
			Fix:    opts.Fix,
			Strict: opts.Strict,
		})
		if validationErr == nil {
			results = []*validation.ValidationResult{result}
		}
	}

	if validationErr != nil {
		log.FailOperation("Validation", validationErr)
		return fmt.Errorf("validation failed: %w", validationErr)
	}

	// Process results
	totalIssues := 0
	totalWarnings := 0
	totalErrors := 0
	totalFixed := 0

	for _, result := range results {
		if len(result.Issues) > 0 {
			log.Info("Validation issues found", "file", result.Path, "issues", len(result.Issues))
			
			for _, issue := range result.Issues {
				switch issue.Severity {
				case "error":
					totalErrors++
					log.Error("Validation error", "file", result.Path, "line", issue.Line, "message", issue.Message)
				case "warning":
					totalWarnings++
					log.Warn("Validation warning", "file", result.Path, "line", issue.Line, "message", issue.Message)
				}
				
				if issue.Fixed {
					totalFixed++
				}
			}
		} else {
			log.Info("Validation passed", "file", result.Path)
		}
		
		totalIssues += len(result.Issues)
	}

	// Summary
	if totalIssues == 0 {
		log.Info("All validations passed", "files_validated", len(results))
	} else {
		log.Info("Validation summary", 
			"files_validated", len(results),
			"total_issues", totalIssues,
			"errors", totalErrors,
			"warnings", totalWarnings,
			"fixed", totalFixed)
	}

	log.CompleteOperation("Validation", "files_validated", len(results), "issues_found", totalIssues)

	// Return error if there were validation errors (not warnings)
	if totalErrors > 0 {
		return fmt.Errorf("validation failed with %d errors", totalErrors)
	}

	return nil
}
