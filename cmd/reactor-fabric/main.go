package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"claude-reactor/internal/fabric"
	"claude-reactor/pkg"
)

var (
	// Version information - will be set during build
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Execute runs the root command
func Execute() error {
	ctx := context.Background()
	
	// Initialize fabric orchestrator
	orchestrator, err := fabric.NewOrchestrator()
	if err != nil {
		return fmt.Errorf("failed to initialize orchestrator: %w", err)
	}

	rootCmd := newRootCmd(orchestrator)
	return rootCmd.ExecuteContext(ctx)
}

func newRootCmd(orchestrator pkg.FabricOrchestrator) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "reactor-fabric",
		Short: "Reactor-Fabric - Distributed MCP orchestration system",
		Long: `Reactor-Fabric is a distributed AI operating system designed to orchestrate 
a suite of specialized, containerized AI agents. The system dynamically spawns 
and manages these agents on-demand, based on a declarative YAML configuration, 
and is designed to handle multiple, concurrent client connections.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, GitCommit, BuildDate),
		Run: func(cmd *cobra.Command, args []string) {
			// Default action - show help
			cmd.Help()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("log-level", "info", "Set log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringP("config", "c", "claude-mcp-suite.yaml", "Path to configuration file")
	rootCmd.PersistentFlags().StringP("listen", "l", "localhost:8080", "Address and port to listen on")
	
	// Add subcommands
	rootCmd.AddCommand(newStartCmd(orchestrator))
	rootCmd.AddCommand(newValidateConfigCmd())
	rootCmd.AddCommand(newVersionCmd())

	return rootCmd
}

func newStartCmd(orchestrator pkg.FabricOrchestrator) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the MCP orchestration server",
		Long: `Start the Reactor-Fabric orchestration server. This will parse the 
configuration file, validate the setup, and start listening for MCP client connections.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("config")
			listen, _ := cmd.Flags().GetString("listen")
			
			return orchestrator.Start(cmd.Context(), configFile, listen)
		},
	}
}

func newValidateConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the MCP suite configuration file",
		Long: `Validate the claude-mcp-suite.yaml configuration file for schema 
and logical errors without starting the orchestrator.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("config")
			
			// This will be implemented in Phase 1-2
			fmt.Printf("Validating configuration file: %s\n", configFile)
			fmt.Println("⚠ Configuration validation not yet implemented (Phase 1 feature)")
			return nil
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("reactor-fabric %s (commit: %s, built: %s)\n", Version, GitCommit, BuildDate)
		},
	}
}