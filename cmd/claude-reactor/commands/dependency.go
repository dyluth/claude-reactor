package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"claude-reactor/pkg"
)

// NewDependencyCmd creates the dependency management command tree
func NewDependencyCmd(app *pkg.AppContainer) *cobra.Command {
	var dependencyCmd = &cobra.Command{
		Use:   "dependency",
		Short: "Dependency management and package manager operations",
		Long: `Manage dependencies across all supported package managers.

Supports unified operations for:
• Go Modules (go.mod, go.sum)  
• Cargo (Cargo.toml, Cargo.lock)
• npm (package.json, package-lock.json)
• Yarn (package.json, yarn.lock)
• pnpm (package.json, pnpm-lock.yaml)
• pip (requirements.txt, setup.py, pyproject.toml)
• Poetry (pyproject.toml, poetry.lock)
• Pipenv (Pipfile, Pipfile.lock)
• Maven (pom.xml)
• Gradle (build.gradle, build.gradle.kts)

Examples:
  claude-reactor dependency detect     # Detect package managers in project
  claude-reactor dependency list      # List all dependencies
  claude-reactor dependency install   # Install dependencies for all package managers
  claude-reactor dependency update    # Update dependencies to latest versions
  claude-reactor dependency audit     # Scan for security vulnerabilities
  claude-reactor dependency outdated  # Check for outdated dependencies
  claude-reactor dependency report    # Generate comprehensive dependency report`,
		Aliases: []string{"deps", "dep"},
	}

	// Add subcommands
	dependencyCmd.AddCommand(
		newDependencyDetectCmd(app),
		newDependencyListCmd(app),
		newDependencyInstallCmd(app),
		newDependencyUpdateCmd(app),
		newDependencyAuditCmd(app),
		newDependencyOutdatedCmd(app),
		newDependencyReportCmd(app),
		newDependencyCleanCmd(app),
	)

	return dependencyCmd
}

// newDependencyDetectCmd detects package managers in the current project
func newDependencyDetectCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "detect [path]",
		Short: "Detect package managers in project",
		Long: `Detect package managers and their configuration files in the current or specified project directory.

Shows available package managers on the system and detected package managers in the project.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			abs, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			app.Logger.Infof("🔍 Detecting package managers in: %s", abs)
			fmt.Printf("📦 Detecting package managers in: %s\n\n", abs)

			// Detect project package managers
			packageManagers, dependencies, err := app.DependencyMgr.DetectProjectDependencies(projectPath)
			if err != nil {
				return fmt.Errorf("failed to detect dependencies: %w", err)
			}

			if len(packageManagers) == 0 {
				fmt.Printf("❌ No package managers detected in project\n\n")
			} else {
				fmt.Printf("✅ Found %d package manager(s):\n", len(packageManagers))
				for _, pm := range packageManagers {
					status := "❌ not available"
					if pm.Available {
						status = "✅ available"
					}
					
					version := ""
					if pm.Version != "" {
						version = fmt.Sprintf(" (v%s)", pm.Version)
					}
					
					fmt.Printf("  • %s%s - %s\n", pm.Name, version, status)
					if len(pm.ConfigFiles) > 0 {
						fmt.Printf("    Config files: %s\n", strings.Join(pm.ConfigFiles, ", "))
					}
					if len(pm.LockFiles) > 0 {
						fmt.Printf("    Lock files: %s\n", strings.Join(pm.LockFiles, ", "))
					}
				}
				fmt.Printf("\n📊 Total dependencies detected: %d\n", len(dependencies))
			}

			return nil
		},
	}
}

// newDependencyListCmd lists all dependencies in the project
func newDependencyListCmd(app *pkg.AppContainer) *cobra.Command {
	var showDetails bool
	var packageManager string
	
	cmd := &cobra.Command{
		Use:   "list [path]",
		Short: "List all project dependencies",
		Long: `List dependencies from all detected package managers in the project.

Shows dependency name, version, type (direct/indirect), and package manager.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			app.Logger.Infof("📋 Listing dependencies in: %s", projectPath)

			packageManagers, dependencies, err := app.DependencyMgr.DetectProjectDependencies(projectPath)
			if err != nil {
				return fmt.Errorf("failed to detect dependencies: %w", err)
			}

			if len(packageManagers) == 0 {
				fmt.Printf("❌ No package managers detected\n")
				return nil
			}

			// Filter by package manager if specified
			if packageManager != "" {
				var filtered []*pkg.DependencyInfo
				for _, dep := range dependencies {
					if dep.PackageManager == packageManager {
						filtered = append(filtered, dep)
					}
				}
				dependencies = filtered
			}

			if len(dependencies) == 0 {
				fmt.Printf("❌ No dependencies found\n")
				return nil
			}

			fmt.Printf("📋 Found %d dependencies across %d package managers:\n\n", len(dependencies), len(packageManagers))

			// Group by package manager
			depsByPM := make(map[string][]*pkg.DependencyInfo)
			for _, dep := range dependencies {
				depsByPM[dep.PackageManager] = append(depsByPM[dep.PackageManager], dep)
			}

			for pmType, deps := range depsByPM {
				if len(deps) == 0 {
					continue
				}
				
				fmt.Printf("🔧 %s (%d dependencies):\n", strings.ToUpper(pmType), len(deps))
				for _, dep := range deps {
					typeIcon := "📦"
					if dep.Type == "dev" {
						typeIcon = "🛠️"
					} else if dep.Type == "indirect" {
						typeIcon = "🔗"
					}
					
					fmt.Printf("  %s %s@%s", typeIcon, dep.Name, dep.CurrentVersion)
					
					if showDetails {
						if dep.Description != "" {
							fmt.Printf(" - %s", dep.Description)
						}
						if dep.License != "" {
							fmt.Printf(" [%s]", dep.License)
						}
						if dep.IsOutdated {
							fmt.Printf(" (outdated)")
						}
						if dep.HasVulnerability {
							fmt.Printf(" ⚠️ vulnerable")
						}
					}
					fmt.Printf("\n")
				}
				fmt.Printf("\n")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showDetails, "details", "d", false, "Show detailed dependency information")
	cmd.Flags().StringVarP(&packageManager, "manager", "m", "", "Filter by package manager (go, npm, cargo, etc.)")

	return cmd
}

// newDependencyInstallCmd installs dependencies for all package managers
func newDependencyInstallCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "install [path]",
		Short: "Install dependencies for all package managers",
		Long: `Install dependencies for all detected package managers in the project.

Runs the appropriate install command for each package manager found.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			app.Logger.Infof("📦 Installing dependencies in: %s", projectPath)
			fmt.Printf("📦 Installing dependencies in: %s\n\n", projectPath)

			results, err := app.DependencyMgr.InstallAllDependencies(projectPath)
			if err != nil {
				return fmt.Errorf("failed to install dependencies: %w", err)
			}

			if len(results) == 0 {
				fmt.Printf("❌ No package managers found\n")
				return nil
			}

			successCount := 0
			for _, result := range results {
				if result.Success {
					fmt.Printf("✅ %s: installed successfully (%s)\n", result.PackageManager, result.Duration)
					successCount++
				} else {
					fmt.Printf("❌ %s: failed - %s\n", result.PackageManager, result.Error)
				}
			}

			fmt.Printf("\n📊 Summary: %d/%d package managers installed successfully\n", successCount, len(results))

			return nil
		},
	}
}

// newDependencyUpdateCmd updates dependencies for detected package managers
func newDependencyUpdateCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "update [path]",
		Short: "Update dependencies for detected package managers",
		Long: `Update dependencies for all detected package managers in the current or specified project directory.
		
This will run the appropriate update command for each detected package manager (npm update, cargo update, etc.).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			abs, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			app.Logger.Infof("⬆️ Updating dependencies in: %s", abs)
			fmt.Printf("⬆️ Updating dependencies in: %s\n\n", abs)

			// Update dependencies for all detected package managers
			results, err := app.DependencyMgr.UpdateAllDependencies(projectPath)
			if err != nil {
				return fmt.Errorf("failed to update dependencies: %w", err)
			}

			if len(results) == 0 {
				fmt.Printf("❌ No package managers found to update\n")
				return nil
			}

			// Display results
			successCount := 0
			for _, result := range results {
				if result.Success {
					successCount++
					fmt.Printf("✅ %s: updated successfully (took %s)\n", result.PackageManager, result.Duration)
				} else {
					fmt.Printf("❌ %s: failed - %s\n", result.PackageManager, result.Error)
				}
			}

			fmt.Printf("\n📊 Summary: %d/%d package managers updated successfully\n", successCount, len(results))

			return nil
		},
	}
}

// newDependencyAuditCmd audits dependencies for vulnerabilities
func newDependencyAuditCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "audit [path]",
		Short: "Audit dependencies for security vulnerabilities",
		Long: `Audit dependencies for security vulnerabilities using appropriate tools for each package manager.
		
This will run security audits using tools like npm audit, cargo-audit, pip-audit, etc.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			abs, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			app.Logger.Infof("🔍 Auditing dependencies in: %s", abs)
			fmt.Printf("🔍 Auditing dependencies for vulnerabilities in: %s\n\n", abs)

			// Audit dependencies for all detected package managers
			auditResults, err := app.DependencyMgr.AuditAllDependencies(projectPath)
			if err != nil {
				return fmt.Errorf("failed to audit dependencies: %w", err)
			}

			if len(auditResults) == 0 {
				fmt.Printf("✅ No vulnerabilities found or no compatible audit tools available\n")
				return nil
			}

			// Display vulnerability results
			fmt.Printf("🚨 Found %d vulnerabilities:\n\n", len(auditResults))

			criticalCount := 0
			highCount := 0
			moderateCount := 0
			lowCount := 0

			for _, vuln := range auditResults {
				severityIcon := "⚠️"
				switch strings.ToLower(vuln.Severity) {
				case "critical":
					severityIcon = "🔴"
					criticalCount++
				case "high":
					severityIcon = "🟠"
					highCount++
				case "moderate":
					severityIcon = "🟡"
					moderateCount++
				case "low":
					severityIcon = "🟢"
					lowCount++
				}

				fmt.Printf("%s %s - %s\n", severityIcon, vuln.ID, vuln.Title)
				fmt.Printf("   Description: %s\n", vuln.Description)
				if vuln.FixedIn != "" {
					fmt.Printf("   Fixed in: %s\n", vuln.FixedIn)
				}
				if vuln.Reference != "" {
					fmt.Printf("   Reference: %s\n", vuln.Reference)
				}
				fmt.Println()
			}

			fmt.Printf("📊 Vulnerability Summary:\n")
			fmt.Printf("   🔴 Critical: %d\n", criticalCount)
			fmt.Printf("   🟠 High: %d\n", highCount)
			fmt.Printf("   🟡 Moderate: %d\n", moderateCount)
			fmt.Printf("   🟢 Low: %d\n", lowCount)

			return nil
		},
	}
}

// newDependencyOutdatedCmd shows outdated dependencies
func newDependencyOutdatedCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "outdated [path]",
		Short: "Show outdated dependencies",
		Long: `Show outdated dependencies for all detected package managers in the current or specified project directory.
		
This will check for available updates using tools like npm outdated, cargo-outdated, pip list --outdated, etc.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			abs, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			app.Logger.Infof("📊 Checking outdated dependencies in: %s", abs)
			fmt.Printf("📊 Checking outdated dependencies in: %s\n\n", abs)

			// Get outdated dependencies for all detected package managers
			outdatedDeps, err := app.DependencyMgr.GetAllOutdatedDependencies(projectPath)
			if err != nil {
				return fmt.Errorf("failed to check outdated dependencies: %w", err)
			}

			if len(outdatedDeps) == 0 {
				fmt.Printf("✅ All dependencies are up to date!\n")
				return nil
			}

			// Group by package manager
			pmGroups := make(map[string][]*pkg.DependencyInfo)
			for _, dep := range outdatedDeps {
				pmGroups[dep.PackageManager] = append(pmGroups[dep.PackageManager], dep)
			}

			fmt.Printf("📋 Found %d outdated dependencies:\n\n", len(outdatedDeps))

			for pmType, deps := range pmGroups {
				fmt.Printf("📦 %s:\n", strings.ToUpper(pmType))
				for _, dep := range deps {
					fmt.Printf("   %s: %s → %s", dep.Name, dep.CurrentVersion, dep.LatestVersion)
					if dep.RequestedVersion != "" && dep.RequestedVersion != dep.CurrentVersion {
						fmt.Printf(" (requested: %s)", dep.RequestedVersion)
					}
					fmt.Println()
				}
				fmt.Println()
			}

			return nil
		},
	}
}

// newDependencyReportCmd generates comprehensive dependency reports
func newDependencyReportCmd(app *pkg.AppContainer) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "report [path]",
		Short: "Generate comprehensive dependency report",
		Long: `Generate a comprehensive dependency report including dependency trees, vulnerability analysis,
outdated packages, and package manager health for the current or specified project directory.

Output formats:
  - text: Human-readable text format (default)
  - json: Machine-readable JSON format`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			abs, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			app.Logger.Infof("📊 Generating dependency report for: %s", abs)

			// Generate comprehensive report
			report, err := app.DependencyMgr.GenerateDependencyReport(projectPath)
			if err != nil {
				return fmt.Errorf("failed to generate dependency report: %w", err)
			}

			// Output in requested format
			if outputFormat == "json" {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(report)
			}

			// Text format output
			fmt.Printf("📊 Dependency Report for: %s\n", abs)
			fmt.Printf("Generated: %s\n\n", report.GeneratedAt)

			// Summary
			fmt.Printf("📋 Summary:\n")
			fmt.Printf("   Total Dependencies: %d\n", report.TotalDependencies)
			fmt.Printf("   Direct Dependencies: %d\n", report.DirectDependencies)
			fmt.Printf("   Indirect Dependencies: %d\n", report.IndirectDependencies)
			fmt.Printf("   Outdated Dependencies: %d\n", report.OutdatedDependencies)
			fmt.Printf("   Vulnerabilities: %d\n", report.Vulnerabilities)
			fmt.Printf("   Security Score: %.1f/100\n\n", report.SecurityScore)

			// Package Managers
			fmt.Printf("📦 Detected Package Managers:\n")
			for _, pm := range report.PackageManagers {
				statusIcon := "✅"
				if !pm.Available {
					statusIcon = "❌"
				}
				fmt.Printf("   %s %s (%s)\n", statusIcon, pm.Name, pm.Version)
				for _, configFile := range pm.ConfigFiles {
					fmt.Printf("     📄 %s\n", configFile)
				}
			}

			if report.Vulnerabilities > 0 {
				fmt.Printf("\n🚨 Security Vulnerabilities: %d found\n", report.Vulnerabilities)
				fmt.Printf("   See 'dependency audit' for detailed vulnerability information\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text|json)")

	return cmd
}

// newDependencyCleanCmd cleans package manager caches
func newDependencyCleanCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "clean [path]",
		Short: "Clean package manager caches",
		Long: `Clean caches for all detected package managers in the current or specified project directory.
		
This will run cache cleaning commands like npm cache clean, cargo clean, pip cache purge, etc.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			abs, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			app.Logger.Infof("🧹 Cleaning package manager caches in: %s", abs)
			fmt.Printf("🧹 Cleaning package manager caches in: %s\n\n", abs)

			// Clean caches for all detected package managers
			results, err := app.DependencyMgr.CleanAllCaches(projectPath)
			if err != nil {
				return fmt.Errorf("failed to clean caches: %w", err)
			}

			if len(results) == 0 {
				fmt.Printf("❌ No package managers found to clean\n")
				return nil
			}

			// Display results
			successCount := 0
			for _, result := range results {
				if result.Success {
					successCount++
					fmt.Printf("✅ %s: cache cleaned successfully (took %s)\n", result.PackageManager, result.Duration)
				} else {
					fmt.Printf("❌ %s: failed - %s\n", result.PackageManager, result.Error)
				}
			}

			fmt.Printf("\n📊 Summary: %d/%d package managers cleaned successfully\n", successCount, len(results))

			return nil
		},
	}
}