package hotreload

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"

	"claude-reactor/pkg"
)

// fileWatcher implements pkg.FileWatcher using fsnotify
type fileWatcher struct {
	logger      pkg.Logger
	sessions    map[string]*watcherSession
	sessionsMux sync.RWMutex
}

// watcherSession represents an active file watching session
type watcherSession struct {
	id           string
	projectPath  string
	config       *pkg.WatchConfig
	watcher      *fsnotify.Watcher
	stats        *pkg.WatchStats
	startTime    time.Time
	status       string
	lastError    string
	containerID  string
	
	// Debouncing
	debounceTimer *time.Timer
	pendingEvents map[string]*pkg.FileEvent
	eventsMux     sync.Mutex
	
	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewFileWatcher creates a new file watcher implementation
func NewFileWatcher(logger pkg.Logger) pkg.FileWatcher {
	return &fileWatcher{
		logger:   logger,
		sessions: make(map[string]*watcherSession),
	}
}

func (fw *fileWatcher) StartWatching(projectPath string, config *pkg.WatchConfig) (*pkg.WatchSession, error) {
	if config == nil {
		return nil, fmt.Errorf("watch config is required")
	}
	
	// Set default values
	if config.DebounceDelay <= 0 {
		config.DebounceDelay = 500 // 500ms default
	}
	
	if config.Recursive == false {
		config.Recursive = true // Default to recursive watching
	}
	
	// Create session
	sessionID := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())
	
	session := &watcherSession{
		id:          sessionID,
		projectPath: projectPath,
		config:      config,
		startTime:   time.Now(),
		status:      "starting",
		containerID: config.ContainerName,
		stats: &pkg.WatchStats{
			EventsByType: make(map[string]int),
		},
		pendingEvents: make(map[string]*pkg.FileEvent),
		ctx:           ctx,
		cancel:        cancel,
		done:          make(chan struct{}),
	}
	
	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}
	session.watcher = watcher
	
	// Add paths to watch
	if err := fw.addWatchPaths(session); err != nil {
		watcher.Close()
		cancel()
		return nil, fmt.Errorf("failed to add watch paths: %w", err)
	}
	
	// Store session
	fw.sessionsMux.Lock()
	fw.sessions[sessionID] = session
	fw.sessionsMux.Unlock()
	
	// Start watching in background
	go fw.watchLoop(session)
	
	session.status = "active"
	fw.logger.Infof("Started file watching session %s for path: %s", sessionID, projectPath)
	
	return &pkg.WatchSession{
		ID:          sessionID,
		ProjectPath: projectPath,
		Config:      config,
		StartTime:   session.startTime.Format(time.RFC3339),
		Status:      session.status,
		Stats:       session.stats,
		ContainerID: session.containerID,
	}, nil
}

func (fw *fileWatcher) StopWatching(sessionID string) error {
	fw.sessionsMux.Lock()
	session, exists := fw.sessions[sessionID]
	if !exists {
		fw.sessionsMux.Unlock()
		return fmt.Errorf("watch session %s not found", sessionID)
	}
	delete(fw.sessions, sessionID)
	fw.sessionsMux.Unlock()
	
	// Cancel context and close watcher
	session.cancel()
	if session.watcher != nil {
		session.watcher.Close()
	}
	
	// Wait for watch loop to finish
	select {
	case <-session.done:
	case <-time.After(5 * time.Second):
		fw.logger.Warnf("Timeout waiting for watch session %s to stop", sessionID)
	}
	
	fw.logger.Infof("Stopped file watching session: %s", sessionID)
	return nil
}

func (fw *fileWatcher) GetActiveSessions() ([]*pkg.WatchSession, error) {
	fw.sessionsMux.RLock()
	defer fw.sessionsMux.RUnlock()
	
	sessions := make([]*pkg.WatchSession, 0, len(fw.sessions))
	
	for _, session := range fw.sessions {
		sessions = append(sessions, &pkg.WatchSession{
			ID:          session.id,
			ProjectPath: session.projectPath,
			Config:      session.config,
			StartTime:   session.startTime.Format(time.RFC3339),
			Status:      session.status,
			Stats:       session.stats,
			LastError:   session.lastError,
			ContainerID: session.containerID,
		})
	}
	
	return sessions, nil
}

func (fw *fileWatcher) UpdateWatchConfig(sessionID string, config *pkg.WatchConfig) error {
	fw.sessionsMux.RLock()
	session, exists := fw.sessions[sessionID]
	fw.sessionsMux.RUnlock()
	
	if !exists {
		return fmt.Errorf("watch session %s not found", sessionID)
	}
	
	// Update configuration
	session.config = config
	
	// Remove existing watches and add new ones
	if err := fw.removeAllWatches(session); err != nil {
		return fmt.Errorf("failed to remove existing watches: %w", err)
	}
	
	if err := fw.addWatchPaths(session); err != nil {
		return fmt.Errorf("failed to add new watch paths: %w", err)
	}
	
	fw.logger.Infof("Updated watch configuration for session: %s", sessionID)
	return nil
}

// addWatchPaths adds directories to watch based on configuration
func (fw *fileWatcher) addWatchPaths(session *watcherSession) error {
	if session.config.Recursive {
		// Add project root and all subdirectories
		return filepath.Walk(session.projectPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			// Skip if this is a file
			if !info.IsDir() {
				return nil
			}
			
			// Check if path should be excluded
			if fw.shouldExcludePath(path, session.config.ExcludePatterns) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			
			return session.watcher.Add(path)
		})
	} else {
		// Only watch the project root directory
		return session.watcher.Add(session.projectPath)
	}
}

// removeAllWatches removes all watched paths
func (fw *fileWatcher) removeAllWatches(session *watcherSession) error {
	for _, path := range session.watcher.WatchList() {
		if err := session.watcher.Remove(path); err != nil {
			fw.logger.Warnf("Failed to remove watch on path %s: %v", path, err)
		}
	}
	return nil
}

// watchLoop is the main event processing loop
func (fw *fileWatcher) watchLoop(session *watcherSession) {
	defer close(session.done)
	defer fw.logger.Infof("Watch loop ended for session: %s", session.id)
	
	for {
		select {
		case <-session.ctx.Done():
			return
			
		case event, ok := <-session.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(session, &event)
			
		case err, ok := <-session.watcher.Errors:
			if !ok {
				return
			}
			fw.handleError(session, err)
		}
	}
}

// handleEvent processes a filesystem event
func (fw *fileWatcher) handleEvent(session *watcherSession, fsEvent *fsnotify.Event) {
	// Check if file should be ignored
	if fw.shouldIgnoreEvent(session, fsEvent) {
		return
	}
	
	// Convert fsnotify event to our FileEvent
	fileEvent := &pkg.FileEvent{
		Type:        fw.convertEventType(fsEvent.Op),
		Path:        fsEvent.Name,
		Timestamp:   time.Now().Format(time.RFC3339),
		IsDirectory: fw.isDirectory(fsEvent.Name),
	}
	
	// Get file size for created/modified events
	if fileEvent.Type == "created" || fileEvent.Type == "modified" {
		if stat := fw.getFileStats(fsEvent.Name); stat != nil {
			fileEvent.Size = stat.Size()
		}
	}
	
	// Update statistics
	session.stats.TotalEvents++
	session.stats.EventsByType[fileEvent.Type]++
	session.stats.LastActivity = fileEvent.Timestamp
	
	fw.logger.Debugf("File event: %s %s (%s)", fileEvent.Type, fileEvent.Path, session.id)
	
	// Handle debouncing
	session.eventsMux.Lock()
	session.pendingEvents[fileEvent.Path] = fileEvent
	
	// Reset debounce timer
	if session.debounceTimer != nil {
		session.debounceTimer.Stop()
	}
	
	session.debounceTimer = time.AfterFunc(
		time.Duration(session.config.DebounceDelay)*time.Millisecond,
		func() {
			fw.processDebouncedEvents(session)
		},
	)
	session.eventsMux.Unlock()
}

// processDebouncedEvents processes accumulated events after debounce delay
func (fw *fileWatcher) processDebouncedEvents(session *watcherSession) {
	session.eventsMux.Lock()
	events := make([]*pkg.FileEvent, 0, len(session.pendingEvents))
	for _, event := range session.pendingEvents {
		events = append(events, event)
	}
	session.pendingEvents = make(map[string]*pkg.FileEvent)
	session.eventsMux.Unlock()
	
	if len(events) == 0 {
		return
	}
	
	fw.logger.Debugf("Processing %d debounced events for session: %s", len(events), session.id)
	
	// Process actions for each event
	for _, event := range events {
		fw.processEventActions(session, event)
	}
}

// processEventActions executes configured actions for an event
func (fw *fileWatcher) processEventActions(session *watcherSession, event *pkg.FileEvent) {
	for _, action := range session.config.Actions {
		if fw.shouldTriggerAction(action, event) {
			if err := fw.executeAction(session, action, event); err != nil {
				fw.logger.Errorf("Failed to execute action %s for event %s: %v", 
					action.Type, event.Path, err)
				session.lastError = err.Error()
				session.status = "error"
			} else {
				session.stats.ActionsTriggered++
				fw.logger.Debugf("Executed action %s for event: %s", action.Type, event.Path)
			}
		}
	}
}

// executeAction executes a watch action
func (fw *fileWatcher) executeAction(session *watcherSession, action *pkg.WatchAction, event *pkg.FileEvent) error {
	// This is a placeholder for action execution
	// In Phase 3, we'll implement the actual action execution logic
	fw.logger.Infof("Would execute action %s for file %s (session: %s)", 
		action.Type, event.Path, session.id)
	return nil
}

// handleError processes watcher errors
func (fw *fileWatcher) handleError(session *watcherSession, err error) {
	fw.logger.Errorf("File watcher error (session %s): %v", session.id, err)
	session.lastError = err.Error()
	session.status = "error"
}

// shouldIgnoreEvent determines if an event should be ignored
func (fw *fileWatcher) shouldIgnoreEvent(session *watcherSession, event *fsnotify.Event) bool {
	// Check exclude patterns
	if fw.shouldExcludePath(event.Name, session.config.ExcludePatterns) {
		return true
	}
	
	// Check include patterns (if specified)
	if len(session.config.IncludePatterns) > 0 {
		return !fw.shouldIncludePath(event.Name, session.config.IncludePatterns)
	}
	
	return false
}

// shouldExcludePath checks if a path matches exclude patterns
func (fw *fileWatcher) shouldExcludePath(path string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err == nil && matched {
			return true
		}
		// Also check if the full path matches
		if matched, err := filepath.Match(pattern, path); err == nil && matched {
			return true
		}
		// Check common ignore patterns
		if fw.matchesCommonIgnorePattern(path, pattern) {
			return true
		}
	}
	return false
}

// shouldIncludePath checks if a path matches include patterns
func (fw *fileWatcher) shouldIncludePath(path string, includePatterns []string) bool {
	for _, pattern := range includePatterns {
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err == nil && matched {
			return true
		}
		// Also check if the full path matches
		if matched, err := filepath.Match(pattern, path); err == nil && matched {
			return true
		}
	}
	return false
}

// matchesCommonIgnorePattern checks common ignore patterns
func (fw *fileWatcher) matchesCommonIgnorePattern(path, pattern string) bool {
	// Handle directory patterns like node_modules/, .git/, etc.
	if strings.HasSuffix(pattern, "/") {
		dirName := strings.TrimSuffix(pattern, "/")
		return strings.Contains(path, "/"+dirName+"/") || strings.HasSuffix(path, "/"+dirName)
	}
	
	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}
	
	return false
}

// shouldTriggerAction determines if an action should be triggered for an event
func (fw *fileWatcher) shouldTriggerAction(action *pkg.WatchAction, event *pkg.FileEvent) bool {
	if len(action.Triggers) == 0 {
		return true // No specific triggers means match all
	}
	
	for _, trigger := range action.Triggers {
		if matched, err := filepath.Match(trigger, filepath.Base(event.Path)); err == nil && matched {
			return true
		}
		// Also check full path
		if matched, err := filepath.Match(trigger, event.Path); err == nil && matched {
			return true
		}
	}
	
	return false
}

// convertEventType converts fsnotify event type to our event type
func (fw *fileWatcher) convertEventType(op fsnotify.Op) string {
	switch {
	case op&fsnotify.Create == fsnotify.Create:
		return "created"
	case op&fsnotify.Write == fsnotify.Write:
		return "modified"
	case op&fsnotify.Remove == fsnotify.Remove:
		return "deleted"
	case op&fsnotify.Rename == fsnotify.Rename:
		return "renamed"
	case op&fsnotify.Chmod == fsnotify.Chmod:
		return "modified" // Treat permission changes as modifications
	default:
		return "unknown"
	}
}

// Helper functions

func (fw *fileWatcher) isDirectory(path string) bool {
	if stat := fw.getFileStats(path); stat != nil {
		return stat.IsDir()
	}
	return false
}

func (fw *fileWatcher) getFileStats(path string) os.FileInfo {
	if stat, err := os.Stat(path); err == nil {
		return stat
	}
	return nil
}