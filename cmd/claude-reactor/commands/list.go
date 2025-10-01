package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	
	"claude-reactor/pkg"
)

// ProjectInfo represents project metadata for list command
type ProjectInfo struct {
	Account       string    `json:"account"`
	ProjectName   string    `json:"project_name"`
	ProjectHash   string    `json:"project_hash"`
	ProjectPath   string    `json:"project_path"`
	Containers    []string  `json:"containers"`
	LastUsed      time.Time `json:"last_used"`
	SessionDir    string    `json:"session_dir"`
}

// ListResponse represents the complete list command response
type ListResponse struct {
	Projects []ProjectInfo `json:"projects"`
	Summary  struct {
		TotalAccounts   int `json:"total_accounts"`
		TotalProjects   int `json:"total_projects"`
		TotalContainers int `json:"total_containers"`
	} `json:"summary"`
}

// NewListCmd creates the list command for showing accounts, projects, and containers
func NewListCmd(app *pkg.AppContainer) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List accounts, projects, and containers",
		Long: `List all Claude accounts, projects, and their associated containers.

Shows account isolation, project-specific sessions, and container status.
Use --json for machine-readable output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle help case when app is nil
			if app == nil {
				return cmd.Help()
			}
			return RunList(cmd, app)
		},
	}

	// List command flags
	listCmd.Flags().BoolP("json", "j", false, "Output in JSON format for scripting")

	return listCmd
}

// RunList handles the actual list command logic
func RunList(cmd *cobra.Command, app *pkg.AppContainer) error {
	if app == nil {
		return fmt.Errorf("application container is not initialized")
	}

	// Parse command flags
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Scan ~/.claude-reactor for all accounts and projects
	projects, err := scanClaudeReactorDirectory(app)
	if err != nil {
		return fmt.Errorf("failed to scan claude-reactor directory: %w", err)
	}

	// Build response
	response := ListResponse{
		Projects: projects,
	}

	// Calculate summary
	accountSet := make(map[string]bool)
	totalContainers := 0
	for _, project := range projects {
		accountSet[project.Account] = true
		totalContainers += len(project.Containers)
	}

	response.Summary.TotalAccounts = len(accountSet)
	response.Summary.TotalProjects = len(projects)
	response.Summary.TotalContainers = totalContainers

	// Output results
	if jsonOutput {
		return outputJSON(response)
	} else {
		return outputTable(response)
	}
}

// scanClaudeReactorDirectory scans ~/.claude-reactor for all accounts and projects
func scanClaudeReactorDirectory(app *pkg.AppContainer) ([]ProjectInfo, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	claudeReactorDir := filepath.Join(homeDir, ".claude-reactor")
	
	// Check if directory exists
	if _, err := os.Stat(claudeReactorDir); os.IsNotExist(err) {
		app.Logger.Info("No ~/.claude-reactor directory found")
		return []ProjectInfo{}, nil
	}

	var projects []ProjectInfo

	// Read account directories
	accountEntries, err := os.ReadDir(claudeReactorDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read claude-reactor directory: %w", err)
	}

	for _, accountEntry := range accountEntries {
		// Skip files (we only want directories for accounts)
		if !accountEntry.IsDir() {
			continue
		}

		accountName := accountEntry.Name()
		accountDir := filepath.Join(claudeReactorDir, accountName)

		// Read project directories within this account
		projectEntries, err := os.ReadDir(accountDir)
		if err != nil {
			app.Logger.Warnf("Failed to read account directory %s: %v", accountDir, err)
			continue
		}

		for _, projectEntry := range projectEntries {
			if !projectEntry.IsDir() {
				continue
			}

			projectDirName := projectEntry.Name()
			sessionDir := filepath.Join(accountDir, projectDirName)

			// Parse project name and hash from directory name
			// Format: {project-name}-{project-hash}
			projectName, projectHash, err := parseProjectDirName(projectDirName)
			if err != nil {
				app.Logger.Debugf("Skipping invalid project directory: %s (%v)", projectDirName, err)
				continue
			}

			// Try to determine project path (this is challenging since we only have the hash)
			// For now, we'll leave it empty - could be enhanced later
			projectPath := "" // TODO: Could try to reverse-lookup from hash if needed

			// Get last used timestamp
			lastUsed := getLastUsedTimestamp(sessionDir)

			// Get associated containers (requires Docker manager)
			containers := getAssociatedContainers(app, accountName, projectHash)

			projects = append(projects, ProjectInfo{
				Account:     accountName,
				ProjectName: projectName,
				ProjectHash: projectHash,
				ProjectPath: projectPath,
				Containers:  containers,
				LastUsed:    lastUsed,
				SessionDir:  sessionDir,
			})
		}
	}

	// Sort projects by last used (most recent first)
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].LastUsed.After(projects[j].LastUsed)
	})

	return projects, nil
}

// parseProjectDirName parses project directory name to extract name and hash
// Format: {project-name}-{project-hash} where hash is 8 characters
func parseProjectDirName(dirName string) (string, string, error) {
	// Find the last dash followed by exactly 8 characters
	parts := strings.Split(dirName, "-")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid format: expected name-hash")
	}

	// Check if last part is 8 characters (project hash)
	lastPart := parts[len(parts)-1]
	if len(lastPart) != 8 {
		return "", "", fmt.Errorf("invalid hash: expected 8 characters, got %d", len(lastPart))
	}

	// Project name is everything except the last part
	projectName := strings.Join(parts[:len(parts)-1], "-")
	projectHash := lastPart

	return projectName, projectHash, nil
}

// getLastUsedTimestamp determines last used timestamp from session directory
func getLastUsedTimestamp(sessionDir string) time.Time {
	// Check modification time of the session directory itself
	if info, err := os.Stat(sessionDir); err == nil {
		return info.ModTime()
	}

	// Fallback: check for .claude-reactor config file in session dir
	configFile := filepath.Join(sessionDir, ".claude-reactor")
	if info, err := os.Stat(configFile); err == nil {
		return info.ModTime()
	}

	// Default to zero time if we can't determine
	return time.Time{}
}

// getAssociatedContainers finds containers associated with this account/project
func getAssociatedContainers(app *pkg.AppContainer, account, projectHash string) []string {
	// This requires Docker manager access
	dockerMgr, err := app.GetDockerManager()
	if err != nil {
		// Docker not available, return empty list
		return []string{}
	}

	// Generate expected container name pattern
	// Pattern: claude-reactor-{variant}-{arch}-{projectHash}-{account}
	// We need to check for all possible variants and architectures

	variants := []string{"base", "go", "full", "cloud", "k8s"}
	architectures := []string{"arm64", "amd64"}
	
	var containers []string

	for _, variant := range variants {
		for _, arch := range architectures {
			// Generate container name using the same pattern as run.go
			containerName := fmt.Sprintf("claude-reactor-%s-%s-%s-%s", 
				variant, arch, projectHash, account)

			// Check if this container exists
			ctx := context.Background()
			
			exists, err := dockerMgr.IsContainerRunning(ctx, containerName)
			if err == nil && exists {
				containers = append(containers, containerName)
			}
		}
	}

	return containers
}

// outputJSON outputs the results in JSON format
func outputJSON(response ListResponse) error {
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

// outputTable outputs the results in flat table format
func outputTable(response ListResponse) error {
	if len(response.Projects) == 0 {
		fmt.Println("No projects found in ~/.claude-reactor/")
		fmt.Printf("Summary: %d accounts, %d projects, %d containers\n",
			response.Summary.TotalAccounts,
			response.Summary.TotalProjects,
			response.Summary.TotalContainers)
		return nil
	}

	// Print header
	fmt.Printf("%-15s %-20s %-8s %-3s %-20s %s\n",
		"ACCOUNT", "PROJECT", "HASH", "CTR", "LAST USED", "SESSION DIR")
	fmt.Printf("%-15s %-20s %-8s %-3s %-20s %s\n",
		strings.Repeat("-", 15),
		strings.Repeat("-", 20),
		strings.Repeat("-", 8),
		strings.Repeat("-", 3),
		strings.Repeat("-", 20),
		strings.Repeat("-", 20))

	// Print projects
	for _, project := range response.Projects {
		lastUsedStr := "never"
		if !project.LastUsed.IsZero() {
			lastUsedStr = formatRelativeTime(project.LastUsed)
		}

		fmt.Printf("%-15s %-20s %-8s %-3d %-20s %s\n",
			truncate(project.Account, 15),
			truncate(project.ProjectName, 20),
			project.ProjectHash,
			len(project.Containers),
			lastUsedStr,
			project.SessionDir)
	}

	// Print summary
	fmt.Printf("\nSummary: %d accounts, %d projects, %d containers\n",
		response.Summary.TotalAccounts,
		response.Summary.TotalProjects,
		response.Summary.TotalContainers)

	return nil
}

// formatRelativeTime formats a timestamp relative to now
func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	} else if duration < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	} else {
		return t.Format("2006-01-02")
	}
}

// truncate truncates a string to the specified length
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}