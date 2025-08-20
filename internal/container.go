package internal

import (
	"claude-reactor/pkg"
	"claude-reactor/internal/architecture"
	"claude-reactor/internal/auth"
	"claude-reactor/internal/config"
	"claude-reactor/internal/dependency"
	"claude-reactor/internal/devcontainer"
	"claude-reactor/internal/docker"
	"claude-reactor/internal/hotreload"
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
	
	// Initialize dependency manager
	dependencyMgr := dependency.NewManager(logger)
	
	// Initialize hot reload components
	fileWatcher := hotreload.NewFileWatcher(logger)
	buildTrigger := hotreload.NewBuildTrigger(logger)
	containerSync := hotreload.NewContainerSync(logger, dockerMgr.GetClient())
	hotReloadMgr := hotreload.NewHotReloadManager(logger, dockerMgr.GetClient())
	
	return &pkg.AppContainer{
		ArchDetector:    archDetector,
		ConfigMgr:       configMgr,
		DockerMgr:       dockerMgr,
		AuthMgr:         authMgr,
		MountMgr:        mountMgr,
		DevContainerMgr: devContainerMgr,
		TemplateMgr:     templateMgr,
		DependencyMgr:   dependencyMgr,
		FileWatcher:     fileWatcher,
		BuildTrigger:    buildTrigger,
		ContainerSync:   containerSync,
		HotReloadMgr:    hotReloadMgr,
		Logger:          logger,
	}, nil
}