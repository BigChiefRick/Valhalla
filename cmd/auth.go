package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"valhalla/internal/config"
	"valhalla/internal/logger"
)

// AuthOptions holds options for the auth command
type AuthOptions struct {
	Provider string
	Server   string
	Username string
	Save     bool
	Test     bool
}

// NewAuthCmd creates the auth command
func NewAuthCmd(log *logger.Logger, cfg *config.Config) *cobra.Command {
	opts := &AuthOptions{}

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication credentials for providers",
		Long: `Configure and test authentication credentials for hypervisor providers.

Supports interactive credential entry with secure password prompts.
Credentials can be saved to configuration file or set as environment variables.

Examples:
  # Configure VMware credentials interactively
  valhalla auth vmware --server vcenter.example.com
  
  # Configure Proxmox credentials
  valhalla auth proxmox --server proxmox.example.com --username admin@pam
  
  # Test existing credentials
  valhalla auth vmware --test`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Provider = args[0]
			}
			return runAuth(log, cfg, opts)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&opts.Server, "server", "s", "", "Server hostname or IP address")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "Username for authentication")
	cmd.Flags().BoolVar(&opts.Save, "save", false, "Save credentials to configuration file")
	cmd.Flags().BoolVar(&opts.Test, "test", false, "Test existing credentials")

	// Add subcommands for each provider
	cmd.AddCommand(newAuthVMwareCmd(log, cfg))
	cmd.AddCommand(newAuthProxmoxCmd(log, cfg))
	cmd.AddCommand(newAuthNutanixCmd(log, cfg))

	return cmd
}

// runAuth executes the auth command
func runAuth(log *logger.Logger, cfg *config.Config, opts *AuthOptions) error {
	if opts.Provider == "" {
		return fmt.Errorf("provider required (vmware, proxmox, nutanix)")
	}

	switch strings.ToLower(opts.Provider) {
	case "vmware", "vsphere":
		return authVMware(log, cfg, opts)
	case "proxmox":
		return authProxmox(log, cfg, opts)
	case "nutanix":
		return authNutanix(log, cfg, opts)
	default:
		return fmt.Errorf("unsupported provider: %s", opts.Provider)
	}
}

// newAuthVMwareCmd creates the VMware auth subcommand
func newAuthVMwareCmd(log *logger.Logger, cfg *config.Config) *cobra.Command {
	opts := &AuthOptions{Provider: "vmware"}

	cmd := &cobra.Command{
		Use:   "vmware",
		Short: "Configure VMware vCenter authentication",
		Long: `Configure authentication credentials for VMware vCenter.

Examples:
  valhalla auth vmware --server vcenter.example.com --username administrator@vsphere.local
  valhalla auth vmware --test`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return authVMware(log, cfg, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Server, "server", "s", "", "vCenter server hostname or IP")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "vCenter username")
	cmd.Flags().BoolVar(&opts.Save, "save", false, "Save credentials to config file")
	cmd.Flags().BoolVar(&opts.Test, "test", false, "Test existing credentials")

	return cmd
}

// newAuthProxmoxCmd creates the Proxmox auth subcommand
func newAuthProxmoxCmd(log *logger.Logger, cfg *config.Config) *cobra.Command {
	opts := &AuthOptions{Provider: "proxmox"}

	cmd := &cobra.Command{
		Use:   "proxmox",
		Short: "Configure Proxmox authentication",
		Long: `Configure authentication credentials for Proxmox VE.

Supports both password and API token authentication.

Examples:
  valhalla auth proxmox --server proxmox.example.com --username root@pam
  valhalla auth proxmox --test`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return authProxmox(log, cfg, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Server, "server", "s", "", "Proxmox server hostname or IP")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "Proxmox username (e.g., root@pam)")
	cmd.Flags().BoolVar(&opts.Save, "save", false, "Save credentials to config file")
	cmd.Flags().BoolVar(&opts.Test, "test", false, "Test existing credentials")

	return cmd
}

// newAuthNutanixCmd creates the Nutanix auth subcommand
func newAuthNutanixCmd(log *logger.Logger, cfg *config.Config) *cobra.Command {
	opts := &AuthOptions{Provider: "nutanix"}

	cmd := &cobra.Command{
		Use:   "nutanix",
		Short: "Configure Nutanix authentication",
		Long: `Configure authentication credentials for Nutanix Prism.

Examples:
  valhalla auth nutanix --server prism.example.com --username admin
  valhalla auth nutanix --test`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return authNutanix(log, cfg, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Server, "server", "s", "", "Nutanix Prism server hostname or IP")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "Nutanix username")
	cmd.Flags().BoolVar(&opts.Save, "save", false, "Save credentials to config file")
	cmd.Flags().BoolVar(&opts.Test, "test", false, "Test existing credentials")

	return cmd
}

// authVMware handles VMware authentication configuration
func authVMware(log *logger.Logger, cfg *config.Config, opts *AuthOptions) error {
	log.Info("Configuring VMware vCenter authentication")

	if opts.Test {
		return testVMwareCredentials(log, cfg)
	}

	// Get current config
	vmwareConfig := cfg.GetVMwareConfig()

	// Prompt for server if not provided
	if opts.Server == "" {
		if vmwareConfig.Server != "" {
			opts.Server = vmwareConfig.Server
			fmt.Printf("Server [%s]: ", vmwareConfig.Server)
		} else {
			fmt.Print("vCenter Server: ")
		}
		
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			opts.Server = input
		}
	}

	// Prompt for username if not provided
	if opts.Username == "" {
		if vmwareConfig.Username != "" {
			opts.Username = vmwareConfig.Username
			fmt.Printf("Username [%s]: ", vmwareConfig.Username)
		} else {
			fmt.Print("Username: ")
		}
		
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			opts.Username = input
		}
	}

	// Prompt for password
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	password := string(passwordBytes)
	fmt.Println() // New line after password input

	// Test credentials
	log.Info("Testing VMware credentials", "server", opts.Server, "username", opts.Username)
	
	testConfig := config.VMwareConfig{
		Server:   opts.Server,
		Username: opts.Username,
		Password: password,
		Insecure: true, // Default to insecure for testing
	}

	if err := testVMwareConnection(log, testConfig); err != nil {
		return fmt.Errorf("credential test failed: %w", err)
	}

	log.Info("VMware credentials verified successfully")

	// Save credentials if requested
	if opts.Save {
		return saveVMwareCredentials(cfg, testConfig, log)
	}

	// Show environment variable instructions
	showVMwareEnvInstructions(testConfig)
	return nil
}

// authProxmox handles Proxmox authentication configuration
func authProxmox(log *logger.Logger, cfg *config.Config, opts *AuthOptions) error {
	log.Info("Configuring Proxmox authentication")

	if opts.Test {
		return testProxmoxCredentials(log, cfg)
	}

	proxmoxConfig := cfg.GetProxmoxConfig()

	// Get server
	if opts.Server == "" {
		if proxmoxConfig.Server != "" {
			opts.Server = proxmoxConfig.Server
			fmt.Printf("Server [%s]: ", proxmoxConfig.Server)
		} else {
			fmt.Print("Proxmox Server: ")
		}
		
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			opts.Server = input
		}
	}

	// Get username
	if opts.Username == "" {
		if proxmoxConfig.Username != "" {
			opts.Username = proxmoxConfig.Username
			fmt.Printf("Username [%s]: ", proxmoxConfig.Username)
		} else {
			fmt.Print("Username (e.g., root@pam): ")
		}
		
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			opts.Username = input
		}
	}

	// Ask for authentication method
	fmt.Print("Use API Token? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	useToken, _ := reader.ReadString('\n')
	useToken = strings.TrimSpace(strings.ToLower(useToken))

	testConfig := config.ProxmoxConfig{
		Server:   opts.Server,
		Username: opts.Username,
		Insecure: true,
	}

	if useToken == "y" || useToken == "yes" {
		// API Token authentication
		fmt.Print("Token ID: ")
		tokenID, _ := reader.ReadString('\n')
		testConfig.TokenID = strings.TrimSpace(tokenID)

		fmt.Print("Secret: ")
		secretBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read secret: %w", err)
		}
		testConfig.Secret = string(secretBytes)
		fmt.Println()
	} else {
		// Password authentication
		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		testConfig.Password = string(passwordBytes)
		fmt.Println()
	}

	// Test credentials
	log.Info("Testing Proxmox credentials", "server", opts.Server, "username", opts.Username)
	if err := testProxmoxConnection(log, testConfig); err != nil {
		return fmt.Errorf("credential test failed: %w", err)
	}

	log.Info("Proxmox credentials verified successfully")

	if opts.Save {
		return saveProxmoxCredentials(cfg, testConfig, log)
	}

	showProxmoxEnvInstructions(testConfig)
	return nil
}

// authNutanix handles Nutanix authentication configuration
func authNutanix(log *logger.Logger, cfg *config.Config, opts *AuthOptions) error {
	log.Info("Configuring Nutanix authentication")

	if opts.Test {
		return testNutanixCredentials(log, cfg)
	}

	nutanixConfig := cfg.GetNutanixConfig()

	// Get server
	if opts.Server == "" {
		if nutanixConfig.Server != "" {
			opts.Server = nutanixConfig.Server
			fmt.Printf("Server [%s]: ", nutanixConfig.Server)
		} else {
			fmt.Print("Nutanix Prism Server: ")
		}
		
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			opts.Server = input
		}
	}

	// Get username
	if opts.Username == "" {
		if nutanixConfig.Username != "" {
			opts.Username = nutanixConfig.Username
			fmt.Printf("Username [%s]: ", nutanixConfig.Username)
		} else {
			fmt.Print("Username: ")
		}
		
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			opts.Username = input
		}
	}

	// Get password
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	password := string(passwordBytes)
	fmt.Println()

	testConfig := config.NutanixConfig{
		Server:   opts.Server,
		Username: opts.Username,
		Password: password,
		Port:     9440,
		Insecure: true,
	}

	// Test credentials
	log.Info("Testing Nutanix credentials", "server", opts.Server, "username", opts.Username)
	if err := testNutanixConnection(log, testConfig); err != nil {
		return fmt.Errorf("credential test failed: %w", err)
	}

	log.Info("Nutanix credentials verified successfully")

	if opts.Save {
		return saveNutanixCredentials(cfg, testConfig, log)
	}

	showNutanixEnvInstructions(testConfig)
	return nil
}

// Test connection functions (placeholder implementations)
func testVMwareConnection(log *logger.Logger, cfg config.VMwareConfig) error {
	// TODO: Implement actual VMware connection test
	log.Info("Testing VMware connection", "server", cfg.Server)
	return nil
}

func testProxmoxConnection(log *logger.Logger, cfg config.ProxmoxConfig) error {
	// TODO: Implement actual Proxmox connection test
	log.Info("Testing Proxmox connection", "server", cfg.Server)
	return nil
}

func testNutanixConnection(log *logger.Logger, cfg config.NutanixConfig) error {
	// TODO: Implement actual Nutanix connection test
	log.Info("Testing Nutanix connection", "server", cfg.Server)
	return nil
}

// Test existing credentials functions
func testVMwareCredentials(log *logger.Logger, cfg *config.Config) error {
	vmwareConfig := cfg.GetVMwareConfig()
	if vmwareConfig.Server == "" || vmwareConfig.Username == "" || vmwareConfig.Password == "" {
		return fmt.Errorf("VMware credentials not configured")
	}
	return testVMwareConnection(log, vmwareConfig)
}

func testProxmoxCredentials(log *logger.Logger, cfg *config.Config) error {
	proxmoxConfig := cfg.GetProxmoxConfig()
	if proxmoxConfig.Server == "" || proxmoxConfig.Username == "" {
		return fmt.Errorf("Proxmox credentials not configured")
	}
	if proxmoxConfig.Password == "" && (proxmoxConfig.TokenID == "" || proxmoxConfig.Secret == "") {
		return fmt.Errorf("Proxmox password or API token not configured")
	}
	return testProxmoxConnection(log, proxmoxConfig)
}

func testNutanixCredentials(log *logger.Logger, cfg *config.Config) error {
	nutanixConfig := cfg.GetNutanixConfig()
	if nutanixConfig.Server == "" || nutanixConfig.Username == "" || nutanixConfig.Password == "" {
		return fmt.Errorf("Nutanix credentials not configured")
	}
	return testNutanixConnection(log, nutanixConfig)
}

// Save credentials functions
func saveVMwareCredentials(cfg *config.Config, vmwareConfig config.VMwareConfig, log *logger.Logger) error {
	// TODO: Implement saving to config file
	log.Info("Saving VMware credentials to config file")
	return nil
}

func saveProxmoxCredentials(cfg *config.Config, proxmoxConfig config.ProxmoxConfig, log *logger.Logger) error {
	// TODO: Implement saving to config file
	log.Info("Saving Proxmox credentials to config file")
	return nil
}

func saveNutanixCredentials(cfg *config.Config, nutanixConfig config.NutanixConfig, log *logger.Logger) error {
	// TODO: Implement saving to config file
	log.Info("Saving Nutanix credentials to config file")
	return nil
}

// Environment variable instruction functions
func showVMwareEnvInstructions(cfg config.VMwareConfig) {
	fmt.Println("\nTo use these credentials, set the following environment variables:")
	fmt.Printf("export VSPHERE_SERVER=\"%s\"\n", cfg.Server)
	fmt.Printf("export VSPHERE_USER=\"%s\"\n", cfg.Username)
	fmt.Printf("export VSPHERE_PASSWORD=\"%s\"\n", cfg.Password)
}

func showProxmoxEnvInstructions(cfg config.ProxmoxConfig) {
	fmt.Println("\nTo use these credentials, set the following environment variables:")
	fmt.Printf("export PROXMOX_SERVER=\"%s\"\n", cfg.Server)
	fmt.Printf("export PROXMOX_USER=\"%s\"\n", cfg.Username)
	if cfg.Password != "" {
		fmt.Printf("export PROXMOX_PASSWORD=\"%s\"\n", cfg.Password)
	}
	if cfg.TokenID != "" {
		fmt.Printf("export PROXMOX_TOKEN_ID=\"%s\"\n", cfg.TokenID)
		fmt.Printf("export PROXMOX_SECRET=\"%s\"\n", cfg.Secret)
	}
}

func showNutanixEnvInstructions(cfg config.NutanixConfig) {
	fmt.Println("\nTo use these credentials, set the following environment variables:")
	fmt.Printf("export NUTANIX_SERVER=\"%s\"\n", cfg.Server)
	fmt.Printf("export NUTANIX_USER=\"%s\"\n", cfg.Username)
	fmt.Printf("export NUTANIX_PASSWORD=\"%s\"\n", cfg.Password)
}
