package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"valhalla/cmd"
	"valhalla/internal/config"
	"valhalla/internal/logger"
)

var (
	cfgFile string
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Initialize logger
	log := logger.New()

	// Initialize configuration
	cfg := config.New()

	// Create root command
	rootCmd := &cobra.Command{
		Use:   "valhalla",
		Short: "Hypervisor Infrastructure Discovery and IaC Generation Tool",
		Long: `Valhalla bridges the gap between existing hypervisor infrastructure and modern Infrastructure as Code practices.

Discover and transform your VMware vCenter, Proxmox, and Nutanix infrastructure into battle-tested IaC templates.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Initialize config from file
			if err := cfg.InitConfig(cfgFile); err != nil {
				log.Fatal("Failed to initialize config", "error", err)
			}
		},
	}

	// Add persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.valhalla.yaml)")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	rootCmd.PersistentFlags().String("log-format", "text", "log format (text, json)")

	// Bind flags to viper
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("log-format", rootCmd.PersistentFlags().Lookup("log-format"))

	// Add subcommands
	rootCmd.AddCommand(cmd.NewDiscoverCmd(log, cfg))
	rootCmd.AddCommand(cmd.NewGenerateCmd(log, cfg))
	rootCmd.AddCommand(cmd.NewAuthCmd(log, cfg))
	rootCmd.AddCommand(cmd.NewValidateCmd(log, cfg))

	// Execute
	if err := rootCmd.Execute(); err != nil {
		log.Error("Command execution failed", "error", err)
		os.Exit(1)
	}
}
