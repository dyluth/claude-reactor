package reactor

import (
	"claude-reactor/internal/reactor/architecture"
	"claude-reactor/internal/reactor/auth"
	"claude-reactor/internal/reactor/config"
	"claude-reactor/internal/reactor/docker"
	"claude-reactor/internal/reactor/docker/validation"
	"claude-reactor/internal/reactor/logging"
	"claude-reactor/internal/reactor/mount"
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

	// Initialize authentication manager
	authMgr := auth.NewManager(logger)

	// Initialize mount manager
	mountMgr := mount.NewManager(logger)

	// Docker components are initialized lazily when needed
	return &pkg.AppContainer{
		ArchDetector:   archDetector,
		ConfigMgr:      configMgr,
		DockerMgr:      nil, // Initialized lazily
		AuthMgr:        authMgr,
		MountMgr:       mountMgr,
		ImageValidator: nil, // Initialized lazily
		Logger:         logger,
		Debug:          debug,
	}, nil
}

// EnsureDockerComponents initializes Docker components lazily if they haven't been initialized yet
func EnsureDockerComponents(app *pkg.AppContainer) error {
	if app.DockerMgr == nil {
		// Initialize Docker manager
		dockerMgr, err := docker.NewManager(app.Logger)
		if err != nil {
			return err
		}
		app.DockerMgr = dockerMgr

		// Initialize image validator
		imageValidator := validation.NewImageValidator(dockerMgr.GetClient(), app.Logger)
		app.ImageValidator = imageValidator
	}
	return nil
}
