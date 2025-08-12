package internal

import (
	"claude-reactor/pkg"
	"claude-reactor/internal/architecture"
	"claude-reactor/internal/auth"
	"claude-reactor/internal/config"
	"claude-reactor/internal/devcontainer"
	"claude-reactor/internal/docker"
	"claude-reactor/internal/logging"
	"claude-reactor/internal/mount"
	"claude-reactor/internal/template"
)

// NewAppContainer creates and initializes the application dependency container
func NewAppContainer() (*pkg.AppContainer, error) {
	// Initialize logger first
	logger := logging.NewLogger()
	
	// Initialize architecture detector
	archDetector := architecture.NewDetector(logger)
	
	// Initialize configuration manager
	configMgr := config.NewManager(logger)
	
	// Initialize Docker manager
	dockerMgr, err := docker.NewManager(logger)
	if err != nil {
		return nil, err
	}
	
	// Initialize authentication manager
	authMgr := auth.NewManager(logger)
	
	// Initialize mount manager
	mountMgr := mount.NewManager(logger)
	
	// Initialize devcontainer manager
	devContainerMgr := devcontainer.NewManager(logger, configMgr)
	
	// Initialize template manager
	templateMgr := template.NewManager(logger, configMgr, devContainerMgr)
	
	return &pkg.AppContainer{
		ArchDetector:    archDetector,
		ConfigMgr:       configMgr,
		DockerMgr:       dockerMgr,
		AuthMgr:         authMgr,
		MountMgr:        mountMgr,
		DevContainerMgr: devContainerMgr,
		TemplateMgr:     templateMgr,
		Logger:          logger,
	}, nil
}