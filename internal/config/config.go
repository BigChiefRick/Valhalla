package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for Valhalla
type Config struct {
	Debug     bool          `mapstructure:"debug"`
	LogFormat string        `mapstructure:"log_format"`
	Providers ProvidersConfig `mapstructure:"providers"`
	Output    OutputConfig  `mapstructure:"output"`
}

// ProvidersConfig holds provider-specific configurations
type ProvidersConfig struct {
	VMware  VMwareConfig  `mapstructure:"vmware"`
	Proxmox ProxmoxConfig `mapstructure:"proxmox"`
	Nutanix NutanixConfig `mapstructure:"nutanix"`
}

// VMwareConfig holds VMware vCenter configuration
type VMwareConfig struct {
	Server     string `mapstructure:"server"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	Insecure   bool   `mapstructure:"insecure"`
	Datacenter string `mapstructure:"datacenter"`
	Cluster    string `mapstructure:"cluster"`
}

// ProxmoxConfig holds Proxmox configuration
type ProxmoxConfig struct {
	Server   string `mapstructure:"server"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	TokenID  string `mapstructure:"token_id"`
	Secret   string `mapstructure:"secret"`
	Node     string `mapstructure:"node"`
	Insecure bool   `mapstructure:"insecure"`
}

// NutanixConfig holds Nutanix configuration
type NutanixConfig struct {
	Server   string `mapstructure:"server"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Port     int    `mapstructure:"port"`
	Insecure bool   `mapstructure:"insecure"`
	Cluster  string `mapstructure:"cluster"`
}

// OutputConfig holds output configuration
type OutputConfig struct {
	Format    string `mapstructure:"format"`
	Directory string `mapstructure:"directory"`
	Filename  string `mapstructure:"filename"`
}

// New creates a new Config instance
func New() *Config {
	return &Config{}
}

// InitConfig initializes the configuration from file and environment
func (c *Config) InitConfig(cfgFile string) error {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}

		// Search config in home directory with name ".valhalla" (without extension)
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".valhalla")
	}

	// Environment variable handling
	viper.SetEnvPrefix("VALHALLA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	c.setDefaults()

	// Read in config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; ignore error
	}

	// Unmarshal config
	if err := viper.Unmarshal(c); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// setDefaults sets default configuration values
func (c *Config) setDefaults() {
	viper.SetDefault("debug", false)
	viper.SetDefault("log_format", "text")
	viper.SetDefault("output.format", "table")
	viper.SetDefault("output.directory", "./output")
	viper.SetDefault("output.filename", "infrastructure")
	
	// VMware defaults
	viper.SetDefault("providers.vmware.insecure", true)
	viper.SetDefault("providers.vmware.datacenter", "")
	viper.SetDefault("providers.vmware.cluster", "")
	
	// Proxmox defaults
	viper.SetDefault("providers.proxmox.insecure", true)
	viper.SetDefault("providers.proxmox.node", "")
	
	// Nutanix defaults
	viper.SetDefault("providers.nutanix.port", 9440)
	viper.SetDefault("providers.nutanix.insecure", true)
	viper.SetDefault("providers.nutanix.cluster", "")
}

// GetVMwareConfig returns VMware configuration with environment variable overrides
func (c *Config) GetVMwareConfig() VMwareConfig {
	cfg := c.Providers.VMware
	
	// Override with environment variables
	if server := os.Getenv("VSPHERE_SERVER"); server != "" {
		cfg.Server = server
	}
	if username := os.Getenv("VSPHERE_USER"); username != "" {
		cfg.Username = username
	}
	if password := os.Getenv("VSPHERE_PASSWORD"); password != "" {
		cfg.Password = password
	}
	
	return cfg
}

// GetProxmoxConfig returns Proxmox configuration with environment variable overrides
func (c *Config) GetProxmoxConfig() ProxmoxConfig {
	cfg := c.Providers.Proxmox
	
	// Override with environment variables
	if server := os.Getenv("PROXMOX_SERVER"); server != "" {
		cfg.Server = server
	}
	if username := os.Getenv("PROXMOX_USER"); username != "" {
		cfg.Username = username
	}
	if password := os.Getenv("PROXMOX_PASSWORD"); password != "" {
		cfg.Password = password
	}
	if tokenID := os.Getenv("PROXMOX_TOKEN_ID"); tokenID != "" {
		cfg.TokenID = tokenID
	}
	if secret := os.Getenv("PROXMOX_SECRET"); secret != "" {
		cfg.Secret = secret
	}
	
	return cfg
}

// GetNutanixConfig returns Nutanix configuration with environment variable overrides
func (c *Config) GetNutanixConfig() NutanixConfig {
	cfg := c.Providers.Nutanix
	
	// Override with environment variables
	if server := os.Getenv("NUTANIX_SERVER"); server != "" {
		cfg.Server = server
	}
	if username := os.Getenv("NUTANIX_USER"); username != "" {
		cfg.Username = username
	}
	if password := os.Getenv("NUTANIX_PASSWORD"); password != "" {
		cfg.Password = password
	}
	
	return cfg
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Output.Directory != "" {
		if err := os.MkdirAll(c.Output.Directory, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}
	
	return nil
}

// WriteConfigFile writes the current configuration to a file
func (c *Config) WriteConfigFile(filename string) error {
	viper.SetConfigFile(filename)
	return viper.WriteConfig()
}

// GetConfigFile returns the path to the config file
func (c *Config) GetConfigFile() string {
	return viper.ConfigFileUsed()
}
