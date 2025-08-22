package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"claude-reactor/pkg"
)

// NewHotReloadCmd creates the hotreload command with installation instructions
func NewHotReloadCmd(app *pkg.AppContainer) *cobra.Command {
	var hotReloadCmd = &cobra.Command{
		Use:   "hotreload",
		Short: "Hot reload functionality for faster development cycles",
		Long: `Hot reload provides automatic file watching, building, and container synchronization
for faster development cycles. Changes to your project files are automatically detected,
built (if needed), and synchronized with the running container.

This feature supports multiple project types including Go, Node.js, Python, Rust, and Java.`,
	}

	// Command flags
	var (
		hotReloadContainerFlag string
		hotReloadWatchPatternsFlag []string
		hotReloadIgnorePatternsFlag []string
		hotReloadDebounceFlag int
		hotReloadDisableBuildFlag bool
		hotReloadDisableSyncFlag bool
		hotReloadVerboseFlag bool
	)

	// Start command
	var hotReloadStartCmd = &cobra.Command{
		Use:   "start [project-path]",
		Short: "Start hot reload for a project",
		Long: `Start hot reload monitoring for a project. This will:

1. Auto-detect the project type (Go, Node.js, Python, etc.)
2. Set up file watching with appropriate patterns
3. Configure build triggers for the detected language
4. Start container synchronization for fast file updates

If no project path is specified, the current directory is used.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Determine project path
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}
			
			// Get absolute project path
			if !strings.HasPrefix(projectPath, "/") {
				if projectPath == "." {
					cwd, err := os.Getwd()
					if err != nil {
						return fmt.Errorf("failed to get current directory: %w", err)
					}
					projectPath = cwd
				} else {
					cwd, err := os.Getwd()
					if err != nil {
						return fmt.Errorf("failed to get current directory: %w", err)
					}
					projectPath = cwd + "/" + strings.TrimPrefix(projectPath, "./")
				}
			}

			fmt.Printf("🔥 Starting hot reload for project: %s\n", projectPath)

			// Determine container ID
			containerID := hotReloadContainerFlag
			if containerID == "" {
				// Auto-detect running container for this project
				projectHash := app.DockerMgr.GenerateProjectHash(projectPath)
				
				// Get current configuration
				config, err := app.ConfigMgr.LoadConfig()
				if err != nil {
					// Use default account if no config
					config = &pkg.Config{Account: "default"}
				}
				
				// Generate expected container name
				variant := config.Variant
				if variant == "" {
					variant = "base"
				}
				
				architecture, _ := app.ArchDetector.GetHostArchitecture()
				containerName := fmt.Sprintf("claude-reactor-v2-%s-%s-%s-%s", variant, architecture, projectHash, config.Account)
				
				// Check if container is running
				running, err := app.DockerMgr.IsContainerRunning(ctx, containerName)
				if err != nil {
					return fmt.Errorf("failed to check container status: %w", err)
				}
				
				if !running {
					return fmt.Errorf("no running container found for project. Please start a container first or specify --container")
				}
				
				// Get container status to get ID
				status, err := app.DockerMgr.GetContainerStatus(ctx, containerName)
				if err != nil {
					return fmt.Errorf("failed to get container status: %w", err)
				}
				
				containerID = status.ID
				fmt.Printf("🔍 Auto-detected container: %s\n", containerID[:12])
			}

			// Create hot reload options
			options := &pkg.HotReloadOptions{
				AutoDetect:          true,
				EnableNotifications: true,
			}

			// Configure watch patterns if specified
			if len(hotReloadWatchPatternsFlag) > 0 || len(hotReloadIgnorePatternsFlag) > 0 || hotReloadDebounceFlag != 500 {
				options.WatchConfig = &pkg.WatchConfig{
					IncludePatterns: hotReloadWatchPatternsFlag,
					ExcludePatterns: hotReloadIgnorePatternsFlag,
					DebounceDelay:   hotReloadDebounceFlag,
					Recursive:       true,
					EnableBuild:     !hotReloadDisableBuildFlag,
					EnableHotReload: !hotReloadDisableSyncFlag,
					ContainerName:   containerID,
				}
			}

			// Start hot reload
			session, err := app.HotReloadMgr.StartHotReload(projectPath, containerID, options)
			if err != nil {
				return fmt.Errorf("failed to start hot reload: %w", err)
			}

			fmt.Printf("✅ Hot reload started successfully\n")
			fmt.Printf("📋 Session ID: %s\n", session.ID)
			if session.ProjectInfo != nil {
				fmt.Printf("📁 Project Type: %s (%s) - %.1f%% confidence\n", 
					session.ProjectInfo.Type, session.ProjectInfo.Framework, session.ProjectInfo.Confidence)
			}
			fmt.Printf("🔍 Watching: %s\n", projectPath)
			fmt.Printf("📦 Container: %s\n", containerID[:12])

			if hotReloadVerboseFlag {
				fmt.Print("\n📊 Session Details:\n")
				fmt.Printf("   Start Time: %s\n", session.StartTime)
				fmt.Printf("   Status: %s\n", session.Status)
				if session.WatchSession != nil {
					fmt.Printf("   Watch Session: %s\n", session.WatchSession.ID)
				}
				if session.SyncSession != nil {
					fmt.Printf("   Sync Session: %s\n", session.SyncSession.ID)
				}
			}

			fmt.Print("\n💡 Use 'claude-reactor hotreload status' to monitor progress\n")
			fmt.Printf("💡 Use 'claude-reactor hotreload stop %s' to stop\n", session.ID)
			
			return nil
		},
	}

	// Stop command
	var hotReloadStopCmd = &cobra.Command{
		Use:   "stop <session-id>",
		Short: "Stop an active hot reload session",
		Long: `Stop an active hot reload session by its ID. This will:

1. Stop file watching
2. Stop build triggers
3. Stop container synchronization
4. Clean up session resources

Use 'hotreload list' to see active sessions and their IDs.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]
			
			fmt.Printf("🛑 Stopping hot reload session: %s\n", sessionID)

			// Stop hot reload
			err := app.HotReloadMgr.StopHotReload(sessionID)
			if err != nil {
				return fmt.Errorf("failed to stop hot reload: %w", err)
			}

			fmt.Printf("✅ Hot reload session stopped: %s\n", sessionID)
			return nil
		},
	}

	// Status command
	var hotReloadStatusCmd = &cobra.Command{
		Use:   "status [session-id]",
		Short: "Show hot reload status",
		Long: `Show detailed status information for hot reload sessions.

If a session ID is provided, shows detailed status for that specific session.
If no session ID is provided, shows a summary of all active sessions.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				// Show detailed status for specific session
				sessionID := args[0]
				status, err := app.HotReloadMgr.GetHotReloadStatus(sessionID)
				if err != nil {
					return fmt.Errorf("failed to get hot reload status: %w", err)
				}

				fmt.Print("🔥 Hot Reload Session Status\n")
				fmt.Printf("📋 Session ID: %s\n", status.SessionID)
				fmt.Printf("📊 Status: %s\n", status.Status)
				fmt.Printf("👀 Watching: %s\n", status.WatchingStatus)
				fmt.Printf("🔨 Build: %s\n", status.BuildStatus)
				fmt.Printf("🔄 Sync: %s\n", status.SyncStatus)
				fmt.Printf("⚡ Hot Reload: %s\n", status.HotReloadStatus)

				if status.Metrics != nil {
					fmt.Print("\n📈 Metrics:\n")
					fmt.Printf("   Uptime: %s\n", status.Metrics.Uptime)
					fmt.Printf("   Total Changes: %d\n", status.Metrics.TotalChanges)
					fmt.Printf("   Build Success Rate: %.1f%%\n", status.Metrics.BuildSuccessRate)
					fmt.Printf("   Average Build Time: %s\n", status.Metrics.AverageBuildTime)
					fmt.Printf("   Average Sync Time: %s\n", status.Metrics.AverageSyncTime)
				}

				if len(status.RecentActivity) > 0 {
					fmt.Print("\n📋 Recent Activity:\n")
					for _, activity := range status.RecentActivity {
						timestamp, _ := time.Parse(time.RFC3339, activity.Timestamp)
						fmt.Printf("   %s [%s] %s\n", 
							timestamp.Format("15:04:05"), 
							strings.ToUpper(activity.Level), 
							activity.Message)
					}
				}
			} else {
				// Show summary of all sessions
				sessions, err := app.HotReloadMgr.GetHotReloadSessions()
				if err != nil {
					return fmt.Errorf("failed to get hot reload sessions: %w", err)
				}

				if len(sessions) == 0 {
					fmt.Print("📭 No active hot reload sessions\n")
					fmt.Print("💡 Use 'claude-reactor hotreload start' to begin hot reloading\n")
					return nil
				}

				fmt.Printf("🔥 Active Hot Reload Sessions (%d)\n\n", len(sessions))

				for i, session := range sessions {
					fmt.Printf("%d. Session: %s\n", i+1, session.ID)
					fmt.Printf("   📁 Project: %s\n", session.ProjectPath)
					if session.ProjectInfo != nil {
						fmt.Printf("   🏷️  Type: %s (%s)\n", session.ProjectInfo.Type, session.ProjectInfo.Framework)
					}
					fmt.Printf("   📦 Container: %s\n", session.ContainerID[:12])
					fmt.Printf("   📊 Status: %s\n", session.Status)
					
					startTime, _ := time.Parse(time.RFC3339, session.StartTime)
					uptime := time.Since(startTime)
					fmt.Printf("   ⏱️  Uptime: %s\n", formatDuration(uptime))
					
					if session.LastActivity != "" {
						lastActivity, _ := time.Parse(time.RFC3339, session.LastActivity)
						fmt.Printf("   🕐 Last Activity: %s ago\n", time.Since(lastActivity).Truncate(time.Second))
					}
					
					if i < len(sessions)-1 {
						fmt.Print("\n")
					}
				}

				fmt.Print("\n💡 Use 'hotreload status <session-id>' for detailed information\n")
			}

			return nil
		},
	}

	// List command (alias for status with no args)
	var hotReloadListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all hot reload sessions",
		Long: `List all active hot reload sessions with their IDs, project paths,
container information, and current status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return hotReloadStatusCmd.RunE(cmd, []string{}) // Reuse the status command logic
		},
	}

	// Config command
	var hotReloadConfigCmd = &cobra.Command{
		Use:   "config <session-id>",
		Short: "Update hot reload configuration",
		Long: `Update the configuration for an active hot reload session.

This allows you to modify watching patterns, build settings, and sync options
without stopping and restarting the session.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]
			
			// Check if any config flags were provided
			hasConfigChanges := len(hotReloadWatchPatternsFlag) > 0 || 
				len(hotReloadIgnorePatternsFlag) > 0 || 
				hotReloadDebounceFlag != -1 ||
				hotReloadDisableBuildFlag ||
				hotReloadDisableSyncFlag

			if !hasConfigChanges {
				return fmt.Errorf("no configuration changes specified. Use --watch, --ignore, --debounce, --no-build, or --no-sync flags")
			}

			// Create new options with the specified changes
			options := &pkg.HotReloadOptions{
				AutoDetect:          true,
				EnableNotifications: true,
			}

			// Configure watch patterns
			if len(hotReloadWatchPatternsFlag) > 0 || len(hotReloadIgnorePatternsFlag) > 0 || hotReloadDebounceFlag != -1 {
				options.WatchConfig = &pkg.WatchConfig{
					Recursive:       true,
					EnableBuild:     !hotReloadDisableBuildFlag,
					EnableHotReload: !hotReloadDisableSyncFlag,
				}
				
				if len(hotReloadWatchPatternsFlag) > 0 {
					options.WatchConfig.IncludePatterns = hotReloadWatchPatternsFlag
				}
				
				if len(hotReloadIgnorePatternsFlag) > 0 {
					options.WatchConfig.ExcludePatterns = hotReloadIgnorePatternsFlag
				}
				
				if hotReloadDebounceFlag != -1 {
					options.WatchConfig.DebounceDelay = hotReloadDebounceFlag
				}
			}

			fmt.Printf("🔧 Updating hot reload configuration for session: %s\n", sessionID)

			// Update configuration
			err := app.HotReloadMgr.UpdateHotReloadConfig(sessionID, options)
			if err != nil {
				return fmt.Errorf("failed to update hot reload configuration: %w", err)
			}

			fmt.Printf("✅ Hot reload configuration updated for session: %s\n", sessionID)

			// Show what was updated
			if len(hotReloadWatchPatternsFlag) > 0 {
				fmt.Printf("👀 Updated watch patterns: %v\n", hotReloadWatchPatternsFlag)
			}
			if len(hotReloadIgnorePatternsFlag) > 0 {
				fmt.Printf("🚫 Updated ignore patterns: %v\n", hotReloadIgnorePatternsFlag)
			}
			if hotReloadDebounceFlag != -1 {
				fmt.Printf("⏱️  Updated debounce delay: %dms\n", hotReloadDebounceFlag)
			}
			if hotReloadDisableBuildFlag {
				fmt.Print("🔨 Disabled automatic building\n")
			}
			if hotReloadDisableSyncFlag {
				fmt.Print("🔄 Disabled file synchronization\n")
			}

			return nil
		},
	}

	// Add flags to start command
	hotReloadStartCmd.Flags().StringVarP(&hotReloadContainerFlag, "container", "c", "", "Target container name or ID (auto-detected if not specified)")
	hotReloadStartCmd.Flags().StringSliceVar(&hotReloadWatchPatternsFlag, "watch", nil, "File patterns to watch (e.g., '**/*.go', '*.js')")
	hotReloadStartCmd.Flags().StringSliceVar(&hotReloadIgnorePatternsFlag, "ignore", nil, "File patterns to ignore (e.g., 'node_modules/', '*.tmp')")
	hotReloadStartCmd.Flags().IntVar(&hotReloadDebounceFlag, "debounce", 500, "Debounce delay in milliseconds")
	hotReloadStartCmd.Flags().BoolVar(&hotReloadDisableBuildFlag, "no-build", false, "Disable automatic building")
	hotReloadStartCmd.Flags().BoolVar(&hotReloadDisableSyncFlag, "no-sync", false, "Disable file synchronization")
	hotReloadStartCmd.Flags().BoolVarP(&hotReloadVerboseFlag, "verbose", "v", false, "Verbose output")

	// Add flags to config command
	hotReloadConfigCmd.Flags().StringSliceVar(&hotReloadWatchPatternsFlag, "watch", nil, "Update file patterns to watch")
	hotReloadConfigCmd.Flags().StringSliceVar(&hotReloadIgnorePatternsFlag, "ignore", nil, "Update file patterns to ignore")
	hotReloadConfigCmd.Flags().IntVar(&hotReloadDebounceFlag, "debounce", -1, "Update debounce delay in milliseconds")
	hotReloadConfigCmd.Flags().BoolVar(&hotReloadDisableBuildFlag, "no-build", false, "Disable automatic building")
	hotReloadConfigCmd.Flags().BoolVar(&hotReloadDisableSyncFlag, "no-sync", false, "Disable file synchronization")

	// Add subcommands
	hotReloadCmd.AddCommand(
		hotReloadStartCmd,
		hotReloadStopCmd,
		hotReloadStatusCmd,
		hotReloadListCmd,
		hotReloadConfigCmd,
	)

	return hotReloadCmd
}

// Helper function for formatting duration
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
}