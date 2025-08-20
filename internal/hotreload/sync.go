package hotreload

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/google/uuid"

	"claude-reactor/pkg"
)

// containerSync implements pkg.ContainerSync for syncing files between host and container
type containerSync struct {
	logger      pkg.Logger
	dockerClient *client.Client
	sessions    map[string]*syncSessionState
	sessionsMux sync.RWMutex
}

// syncSessionState tracks the state of a sync session
type syncSessionState struct {
	id          string
	containerID string
	mappings    []*pkg.SyncMapping
	startTime   time.Time
	status      string
	stats       *pkg.SyncStats
	ctx         context.Context
	cancel      context.CancelFunc
	done        chan struct{}
}

// NewContainerSync creates a new container sync implementation
func NewContainerSync(logger pkg.Logger, dockerClient *client.Client) pkg.ContainerSync {
	return &containerSync{
		logger:       logger,
		dockerClient: dockerClient,
		sessions:     make(map[string]*syncSessionState),
	}
}

func (cs *containerSync) SyncFile(containerID string, hostPath string, containerPath string) error {
	cs.logger.Debugf("Syncing file %s to container %s at %s", hostPath, containerID, containerPath)
	
	// Check if container exists and is running
	if err := cs.validateContainer(containerID); err != nil {
		return fmt.Errorf("container validation failed: %w", err)
	}
	
	// Read file from host
	fileContent, err := os.ReadFile(hostPath)
	if err != nil {
		return fmt.Errorf("failed to read host file %s: %w", hostPath, err)
	}
	
	// Create tar archive with the file
	tarBuffer := &bytes.Buffer{}
	tarWriter := tar.NewWriter(tarBuffer)
	
	// Get file info
	fileInfo, err := os.Stat(hostPath)
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", hostPath, err)
	}
	
	// Create tar header
	header := &tar.Header{
		Name: filepath.Base(containerPath),
		Mode: int64(fileInfo.Mode()),
		Size: int64(len(fileContent)),
		ModTime: fileInfo.ModTime(),
	}
	
	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}
	
	if _, err := tarWriter.Write(fileContent); err != nil {
		return fmt.Errorf("failed to write file content to tar: %w", err)
	}
	
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}
	
	// Copy tar archive to container
	containerDir := filepath.Dir(containerPath)
	err = cs.dockerClient.CopyToContainer(
		context.Background(),
		containerID,
		containerDir,
		tarBuffer,
		container.CopyToContainerOptions{},
	)
	
	if err != nil {
		return fmt.Errorf("failed to copy file to container: %w", err)
	}
	
	cs.logger.Debugf("Successfully synced file %s to container", hostPath)
	return nil
}

func (cs *containerSync) SyncDirectory(containerID string, hostPath string, containerPath string, options *pkg.SyncOptions) error {
	cs.logger.Infof("Syncing directory %s to container %s at %s", hostPath, containerID, containerPath)
	
	// Set default options
	if options == nil {
		options = &pkg.SyncOptions{
			PreservePermissions: true,
			PreserveTimestamps:  true,
			Recursive:          true,
		}
	}
	
	// Check if container exists and is running
	if err := cs.validateContainer(containerID); err != nil {
		return fmt.Errorf("container validation failed: %w", err)
	}
	
	// Create tar archive with directory contents
	tarBuffer := &bytes.Buffer{}
	tarWriter := tar.NewWriter(tarBuffer)
	
	err := filepath.Walk(hostPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip if not recursive and not in root directory
		if !options.Recursive {
			rel, err := filepath.Rel(hostPath, path)
			if err != nil {
				return err
			}
			if strings.Contains(rel, string(filepath.Separator)) {
				return nil
			}
		}
		
		// Check exclude patterns
		if cs.shouldExcludeFromSync(path, options.ExcludePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Calculate relative path within archive
		relPath, err := filepath.Rel(hostPath, path)
		if err != nil {
			return err
		}
		
		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		
		header.Name = relPath
		if !options.PreservePermissions {
			if info.IsDir() {
				header.Mode = 0755
			} else {
				header.Mode = 0644
			}
		}
		
		if !options.PreserveTimestamps {
			header.ModTime = time.Now()
		}
		
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		
		// Write file content if it's a regular file
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			
			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}
		
		return nil
	})
	
	if err != nil {
		tarWriter.Close()
		return fmt.Errorf("failed to create tar archive: %w", err)
	}
	
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}
	
	// Copy tar archive to container
	if options.DryRun {
		cs.logger.Infof("Dry run: would sync directory %s to container", hostPath)
		return nil
	}
	
	err = cs.dockerClient.CopyToContainer(
		context.Background(),
		containerID,
		containerPath,
		tarBuffer,
		container.CopyToContainerOptions{},
	)
	
	if err != nil {
		return fmt.Errorf("failed to copy directory to container: %w", err)
	}
	
	cs.logger.Infof("Successfully synced directory %s to container", hostPath)
	return nil
}

func (cs *containerSync) GetSyncStatus(containerID string) (*pkg.SyncStatus, error) {
	// Check if container exists
	if err := cs.validateContainer(containerID); err != nil {
		return nil, err
	}
	
	// Find active sync sessions for this container
	cs.sessionsMux.RLock()
	defer cs.sessionsMux.RUnlock()
	
	var activeSession *syncSessionState
	for _, session := range cs.sessions {
		if session.containerID == containerID {
			activeSession = session
			break
		}
	}
	
	status := &pkg.SyncStatus{
		ContainerID: containerID,
		Active:      activeSession != nil,
	}
	
	if activeSession != nil {
		status.LastSync = activeSession.startTime.Format(time.RFC3339)
		status.FilesSynced = activeSession.stats.TotalFiles
		// TODO: Add error tracking
	}
	
	return status, nil
}

func (cs *containerSync) StartContinuousSync(containerID string, mappings []*pkg.SyncMapping) (*pkg.SyncSession, error) {
	cs.logger.Infof("Starting continuous sync for container: %s", containerID)
	
	// Validate container
	if err := cs.validateContainer(containerID); err != nil {
		return nil, fmt.Errorf("container validation failed: %w", err)
	}
	
	// Create session
	sessionID := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())
	
	session := &syncSessionState{
		id:          sessionID,
		containerID: containerID,
		mappings:    mappings,
		startTime:   time.Now(),
		status:      "active",
		stats: &pkg.SyncStats{
			TotalFiles:      0,
			SyncOperations:  0,
			FailedSyncs:     0,
		},
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	
	// Store session
	cs.sessionsMux.Lock()
	cs.sessions[sessionID] = session
	cs.sessionsMux.Unlock()
	
	// Start continuous sync in background
	go cs.continuousSyncLoop(session)
	
	return &pkg.SyncSession{
		ID:          sessionID,
		ContainerID: containerID,
		Mappings:    mappings,
		StartTime:   session.startTime.Format(time.RFC3339),
		Status:      session.status,
		Stats:       session.stats,
	}, nil
}

func (cs *containerSync) StopContinuousSync(sessionID string) error {
	cs.sessionsMux.Lock()
	session, exists := cs.sessions[sessionID]
	if !exists {
		cs.sessionsMux.Unlock()
		return fmt.Errorf("sync session %s not found", sessionID)
	}
	delete(cs.sessions, sessionID)
	cs.sessionsMux.Unlock()
	
	// Cancel context
	session.cancel()
	
	// Wait for sync loop to finish
	select {
	case <-session.done:
	case <-time.After(5 * time.Second):
		cs.logger.Warnf("Timeout waiting for sync session %s to stop", sessionID)
	}
	
	cs.logger.Infof("Stopped continuous sync session: %s", sessionID)
	return nil
}

// continuousSyncLoop runs the continuous sync process
func (cs *containerSync) continuousSyncLoop(session *syncSessionState) {
	defer close(session.done)
	defer cs.logger.Infof("Continuous sync loop ended for session: %s", session.id)
	
	ticker := time.NewTicker(1 * time.Second) // Check every second
	defer ticker.Stop()
	
	for {
		select {
		case <-session.ctx.Done():
			return
			
		case <-ticker.C:
			// TODO: Implement actual file change detection and sync
			// For now, this is a placeholder that would integrate with file watcher
			cs.logger.Debugf("Continuous sync check for session: %s", session.id)
		}
	}
}

// validateContainer checks if container exists and is running
func (cs *containerSync) validateContainer(containerID string) error {
	ctx := context.Background()
	
	// Get container info
	containerInfo, err := cs.dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("container %s not found: %w", containerID, err)
	}
	
	// Check if container is running
	if containerInfo.State.Status != "running" {
		return fmt.Errorf("container %s is not running (status: %s)", 
			containerID, containerInfo.State.Status)
	}
	
	return nil
}

// shouldExcludeFromSync checks if a path should be excluded from sync
func (cs *containerSync) shouldExcludeFromSync(path string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err == nil && matched {
			return true
		}
		// Also check full path
		if matched, err := filepath.Match(pattern, path); err == nil && matched {
			return true
		}
		// Handle directory patterns
		if strings.HasSuffix(pattern, "/") {
			dirName := strings.TrimSuffix(pattern, "/")
			if strings.Contains(path, "/"+dirName+"/") || strings.HasSuffix(path, "/"+dirName) {
				return true
			}
		}
	}
	return false
}