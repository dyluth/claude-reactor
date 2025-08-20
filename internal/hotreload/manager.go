package hotreload

import (
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/uuid"

	"claude-reactor/pkg"
)

// hotReloadManager implements pkg.HotReloadManager
type hotReloadManager struct {
	logger        pkg.Logger
	fileWatcher   pkg.FileWatcher
	buildTrigger  pkg.BuildTrigger
	containerSync pkg.ContainerSync
	sessions      map[string]*hotReloadSessionState
	sessionsMux   sync.RWMutex
}

// hotReloadSessionState tracks the internal state of a hot reload session
type hotReloadSessionState struct {
	session     *pkg.HotReloadSession
	activities  []*pkg.ActivityEvent
	metrics     *pkg.HotReloadMetrics
	lastActivity time.Time
	buildCount   int
	successCount int
	failCount    int
}

// NewHotReloadManager creates a new hot reload manager
func NewHotReloadManager(logger pkg.Logger, dockerClient *client.Client) pkg.HotReloadManager {
	return &hotReloadManager{
		logger:        logger,
		fileWatcher:   NewFileWatcher(logger),
		buildTrigger:  NewBuildTrigger(logger),
		containerSync: NewContainerSync(logger, dockerClient),
		sessions:      make(map[string]*hotReloadSessionState),
	}
}

func (hrm *hotReloadManager) StartHotReload(projectPath string, containerID string, options *pkg.HotReloadOptions) (*pkg.HotReloadSession, error) {
	hrm.logger.Infof("Starting hot reload for project: %s, container: %s", projectPath, containerID)
	
	// Set default options
	if options == nil {
		options = &pkg.HotReloadOptions{
			AutoDetect:          true,
			EnableNotifications: true,
		}
	}
	
	// Auto-detect project type if enabled
	var projectInfo *pkg.ProjectBuildInfo
	var err error
	
	if options.AutoDetect {
		projectInfo, err = hrm.buildTrigger.DetectProjectType(projectPath)
		if err != nil {
			return nil, fmt.Errorf("failed to detect project type: %w", err)
		}
		hrm.logger.Infof("Detected project type: %s (%s) with confidence %.1f%%", 
			projectInfo.Type, projectInfo.Framework, projectInfo.Confidence)
	}
	
	// Create session
	sessionID := uuid.New().String()
	session := &pkg.HotReloadSession{
		ID:           sessionID,
		ProjectPath:  projectPath,
		ContainerID:  containerID,
		ProjectInfo:  projectInfo,
		Options:      options,
		StartTime:    time.Now().Format(time.RFC3339),
		Status:       "starting",
	}
	
	// Create internal session state
	sessionState := &hotReloadSessionState{
		session:      session,
		activities:   make([]*pkg.ActivityEvent, 0),
		lastActivity: time.Now(),
		metrics: &pkg.HotReloadMetrics{
			TotalChanges:     0,
			BuildSuccessRate: 100.0,
		},
	}
	
	// Store session
	hrm.sessionsMux.Lock()
	hrm.sessions[sessionID] = sessionState
	hrm.sessionsMux.Unlock()
	
	// Set up file watching
	if err := hrm.setupFileWatching(sessionState); err != nil {
		hrm.removeSession(sessionID)
		return nil, fmt.Errorf("failed to setup file watching: %w", err)
	}
	
	// Set up container sync
	if err := hrm.setupContainerSync(sessionState); err != nil {
		hrm.removeSession(sessionID)
		return nil, fmt.Errorf("failed to setup container sync: %w", err)
	}
	
	// Add startup activity
	hrm.addActivity(sessionState, &pkg.ActivityEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      "session_started",
		Message:   fmt.Sprintf("Hot reload session started for %s project", projectInfo.Type),
		Level:     "info",
	})
	
	session.Status = "active"
	hrm.logger.Infof("Hot reload session %s started successfully", sessionID)
	
	return session, nil
}

func (hrm *hotReloadManager) StopHotReload(sessionID string) error {
	hrm.sessionsMux.RLock()
	sessionState, exists := hrm.sessions[sessionID]
	hrm.sessionsMux.RUnlock()
	
	if !exists {
		return fmt.Errorf("hot reload session %s not found", sessionID)
	}
	
	hrm.logger.Infof("Stopping hot reload session: %s", sessionID)
	
	// Stop file watching
	if sessionState.session.WatchSession != nil {
		if err := hrm.fileWatcher.StopWatching(sessionState.session.WatchSession.ID); err != nil {
			hrm.logger.Warnf("Failed to stop file watching: %v", err)
		}
	}
	
	// Stop container sync
	if sessionState.session.SyncSession != nil {
		if err := hrm.containerSync.StopContinuousSync(sessionState.session.SyncSession.ID); err != nil {
			hrm.logger.Warnf("Failed to stop container sync: %v", err)
		}
	}
	
	// Add shutdown activity
	hrm.addActivity(sessionState, &pkg.ActivityEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      "session_stopped",
		Message:   "Hot reload session stopped",
		Level:     "info",
	})
	
	// Update session status
	sessionState.session.Status = "stopped"
	
	// Remove session
	hrm.removeSession(sessionID)
	
	hrm.logger.Infof("Hot reload session %s stopped", sessionID)
	return nil
}

func (hrm *hotReloadManager) GetHotReloadSessions() ([]*pkg.HotReloadSession, error) {
	hrm.sessionsMux.RLock()
	defer hrm.sessionsMux.RUnlock()
	
	sessions := make([]*pkg.HotReloadSession, 0, len(hrm.sessions))
	for _, sessionState := range hrm.sessions {
		// Update last activity
		sessionState.session.LastActivity = sessionState.lastActivity.Format(time.RFC3339)
		sessions = append(sessions, sessionState.session)
	}
	
	return sessions, nil
}

func (hrm *hotReloadManager) UpdateHotReloadConfig(sessionID string, options *pkg.HotReloadOptions) error {
	hrm.sessionsMux.RLock()
	sessionState, exists := hrm.sessions[sessionID]
	hrm.sessionsMux.RUnlock()
	
	if !exists {
		return fmt.Errorf("hot reload session %s not found", sessionID)
	}
	
	// Update options
	sessionState.session.Options = options
	
	// Update file watching configuration if needed
	if options.WatchConfig != nil && sessionState.session.WatchSession != nil {
		if err := hrm.fileWatcher.UpdateWatchConfig(sessionState.session.WatchSession.ID, options.WatchConfig); err != nil {
			return fmt.Errorf("failed to update watch config: %w", err)
		}
	}
	
	// Add configuration update activity
	hrm.addActivity(sessionState, &pkg.ActivityEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      "config_updated",
		Message:   "Hot reload configuration updated",
		Level:     "info",
	})
	
	hrm.logger.Infof("Updated hot reload config for session: %s", sessionID)
	return nil
}

func (hrm *hotReloadManager) GetHotReloadStatus(sessionID string) (*pkg.HotReloadStatus, error) {
	hrm.sessionsMux.RLock()
	sessionState, exists := hrm.sessions[sessionID]
	hrm.sessionsMux.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("hot reload session %s not found", sessionID)
	}
	
	// Calculate uptime
	startTime, _ := time.Parse(time.RFC3339, sessionState.session.StartTime)
	uptime := time.Since(startTime)
	sessionState.metrics.Uptime = uptime.String()
	
	// Calculate success rate
	if sessionState.buildCount > 0 {
		sessionState.metrics.BuildSuccessRate = float64(sessionState.successCount) / float64(sessionState.buildCount) * 100.0
	}
	
	status := &pkg.HotReloadStatus{
		SessionID:       sessionID,
		Status:          sessionState.session.Status,
		WatchingStatus:  "active", // TODO: Get from file watcher
		BuildStatus:     "idle",   // TODO: Track build status
		SyncStatus:      "active", // TODO: Get from container sync
		HotReloadStatus: "active", // TODO: Track hot reload server status
		RecentActivity:  hrm.getRecentActivities(sessionState, 10),
		Metrics:         sessionState.metrics,
	}
	
	return status, nil
}

// setupFileWatching configures and starts file watching for the session
func (hrm *hotReloadManager) setupFileWatching(sessionState *hotReloadSessionState) error {
	session := sessionState.session
	
	// Create watch configuration
	watchConfig := &pkg.WatchConfig{
		IncludePatterns: []string{"**/*"},
		ExcludePatterns: []string{
			"node_modules/", ".git/", "target/", "build/", "dist/", "__pycache__/",
			"*.log", "*.tmp", ".DS_Store", "Thumbs.db",
		},
		DebounceDelay:   500,
		Recursive:       true,
		EnableBuild:     true,
		EnableHotReload: true,
		ContainerName:   session.ContainerID,
	}
	
	// Use custom watch config if provided
	if session.Options.WatchConfig != nil {
		watchConfig = session.Options.WatchConfig
		watchConfig.ContainerName = session.ContainerID
	}
	
	// Add project-specific patterns if detected
	if session.ProjectInfo != nil {
		watchConfig.IncludePatterns = session.ProjectInfo.WatchPatterns
		watchConfig.ExcludePatterns = append(watchConfig.ExcludePatterns, session.ProjectInfo.IgnorePatterns...)
		watchConfig.ProjectType = session.ProjectInfo.Type
	}
	
	// Create watch actions
	watchConfig.Actions = []*pkg.WatchAction{
		{
			Type:        "build",
			Triggers:    watchConfig.IncludePatterns,
			InContainer: false,
			Timeout:     300000, // 5 minutes
		},
		{
			Type:        "sync",
			Triggers:    watchConfig.IncludePatterns,
			InContainer: true,
		},
	}
	
	// Start file watching
	watchSession, err := hrm.fileWatcher.StartWatching(session.ProjectPath, watchConfig)
	if err != nil {
		return err
	}
	
	session.WatchSession = watchSession
	hrm.logger.Infof("File watching started for session: %s", session.ID)
	return nil
}

// setupContainerSync configures and starts container synchronization
func (hrm *hotReloadManager) setupContainerSync(sessionState *hotReloadSessionState) error {
	session := sessionState.session
	
	// Create sync mappings
	mappings := []*pkg.SyncMapping{
		{
			HostPath:        session.ProjectPath,
			ContainerPath:   "/app", // Default container path
			Direction:       "host-to-container",
			WatchPatterns:   []string{"**/*"},
			ExcludePatterns: []string{"node_modules/", ".git/", "target/", "build/", "dist/"},
		},
	}
	
	// Start continuous sync
	syncSession, err := hrm.containerSync.StartContinuousSync(session.ContainerID, mappings)
	if err != nil {
		return err
	}
	
	session.SyncSession = syncSession
	hrm.logger.Infof("Container sync started for session: %s", session.ID)
	return nil
}

// addActivity adds an activity event to the session
func (hrm *hotReloadManager) addActivity(sessionState *hotReloadSessionState, activity *pkg.ActivityEvent) {
	sessionState.activities = append(sessionState.activities, activity)
	sessionState.lastActivity = time.Now()
	
	// Keep only last 50 activities
	if len(sessionState.activities) > 50 {
		sessionState.activities = sessionState.activities[1:]
	}
	
	// Update metrics
	sessionState.metrics.TotalChanges++
	
	hrm.logger.Debugf("Added activity to session %s: %s", sessionState.session.ID, activity.Message)
}

// getRecentActivities returns the most recent activities
func (hrm *hotReloadManager) getRecentActivities(sessionState *hotReloadSessionState, limit int) []*pkg.ActivityEvent {
	if len(sessionState.activities) <= limit {
		return sessionState.activities
	}
	
	return sessionState.activities[len(sessionState.activities)-limit:]
}

// removeSession removes a session from the manager
func (hrm *hotReloadManager) removeSession(sessionID string) {
	hrm.sessionsMux.Lock()
	defer hrm.sessionsMux.Unlock()
	
	delete(hrm.sessions, sessionID)
}