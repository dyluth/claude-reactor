package reactor

import (
	"claude-reactor/internal/reactor/architecture"
	"claude-reactor/internal/reactor/auth"
	"claude-reactor/internal/reactor/config"
	"claude-reactor/internal/reactor/dependency"
	"claude-reactor/internal/reactor/devcontainer"
	"claude-reactor/internal/reactor/docker"
	"claude-reactor/internal/reactor/docker/validation"
	"claude-reactor/internal/reactor/hotreload"
	"claude-reactor/internal/reactor/logging"
	"claude-reactor/internal/reactor/mount"
	"claude-reactor/internal/reactor/template"
	"claude-reactor/pkg"
)

// NewAppContainer creates and initializes the application dependency container
func NewAppContainer(debug bool, verbose bool, logLevel string) (*pkg.AppContainer, error) {
	// Initialize logger first with provided settings
	logger := logging.NewLoggerWithFlags(debug, verbose, logLevel)

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

	// Initialize image validator
	imageValidator := validation.NewImageValidator(dockerMgr.GetClient(), logger)

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
		ImageValidator:  imageValidator,
		Logger:          logger,
		Debug:           debug,
	}, nil
}
