package mocks

import (
	"context"
	"io"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/mock"

	"claude-reactor/pkg"
)

// MockConfigManager is a mock implementation of ConfigManager
type MockConfigManager struct {
	mock.Mock
}

func (m *MockConfigManager) LoadConfig() (*pkg.Config, error) {
	args := m.Called()
	return args.Get(0).(*pkg.Config), args.Error(1)
}

func (m *MockConfigManager) SaveConfig(config *pkg.Config) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockConfigManager) ValidateConfig(config *pkg.Config) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockConfigManager) AutoDetectVariant(projectPath string) (string, error) {
	args := m.Called(projectPath)
	return args.String(0), args.Error(1)
}

func (m *MockConfigManager) GetConfigPath() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockConfigManager) GetDefaultConfig() *pkg.Config {
	args := m.Called()
	return args.Get(0).(*pkg.Config)
}

func (m *MockConfigManager) ListAccounts() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

// MockDockerManager is a mock implementation of DockerManager
type MockDockerManager struct {
	mock.Mock
}

func (m *MockDockerManager) BuildImage(ctx context.Context, variant string, platform string) error {
	args := m.Called(ctx, variant, platform)
	return args.Error(0)
}

func (m *MockDockerManager) StartContainer(ctx context.Context, config *pkg.ContainerConfig) (string, error) {
	args := m.Called(ctx, config)
	return args.String(0), args.Error(1)
}

func (m *MockDockerManager) StopContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockDockerManager) RemoveContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockDockerManager) IsContainerRunning(ctx context.Context, containerName string) (bool, error) {
	args := m.Called(ctx, containerName)
	return args.Bool(0), args.Error(1)
}

func (m *MockDockerManager) GetContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockDockerManager) RebuildImage(ctx context.Context, variant string, platform string, force bool) error {
	args := m.Called(ctx, variant, platform, force)
	return args.Error(0)
}

func (m *MockDockerManager) GetContainerStatus(ctx context.Context, containerName string) (*pkg.ContainerStatus, error) {
	args := m.Called(ctx, containerName)
	return args.Get(0).(*pkg.ContainerStatus), args.Error(1)
}

func (m *MockDockerManager) CleanContainer(ctx context.Context, containerName string) error {
	args := m.Called(ctx, containerName)
	return args.Error(0)
}

func (m *MockDockerManager) CleanAllContainers(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDockerManager) AttachToContainer(ctx context.Context, containerName string, command []string, interactive bool) error {
	args := m.Called(ctx, containerName, command, interactive)
	return args.Error(0)
}

func (m *MockDockerManager) HealthCheck(ctx context.Context, containerName string, maxRetries int) error {
	args := m.Called(ctx, containerName, maxRetries)
	return args.Error(0)
}

func (m *MockDockerManager) ListVariants() ([]pkg.VariantDefinition, error) {
	args := m.Called()
	return args.Get(0).([]pkg.VariantDefinition), args.Error(1)
}

func (m *MockDockerManager) GenerateContainerName(projectPath, variant, architecture, account string) string {
	args := m.Called(projectPath, variant, architecture, account)
	return args.String(0)
}

func (m *MockDockerManager) GenerateProjectHash(projectPath string) string {
	args := m.Called(projectPath)
	return args.String(0)
}

func (m *MockDockerManager) GetImageName(variant, architecture string) string {
	args := m.Called(variant, architecture)
	return args.String(0)
}

func (m *MockDockerManager) CleanImages(ctx context.Context, all bool) error {
	args := m.Called(ctx, all)
	return args.Error(0)
}

func (m *MockDockerManager) BuildImageWithRegistry(ctx context.Context, variant, platform string, devMode, registryOff, pullLatest bool) error {
	args := m.Called(ctx, variant, platform, devMode, registryOff, pullLatest)
	return args.Error(0)
}

func (m *MockDockerManager) GetClient() *client.Client {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*client.Client)
}

// MockAuthManager is a mock implementation of AuthManager
type MockAuthManager struct {
	mock.Mock
}

func (m *MockAuthManager) GetAuthConfig(account string) (*pkg.AuthConfig, error) {
	args := m.Called(account)
	return args.Get(0).(*pkg.AuthConfig), args.Error(1)
}

func (m *MockAuthManager) SetupAuth(account string, apiKey string) error {
	args := m.Called(account, apiKey)
	return args.Error(0)
}

func (m *MockAuthManager) ValidateAuth(account string) error {
	args := m.Called(account)
	return args.Error(0)
}

func (m *MockAuthManager) IsAuthenticated(account string) bool {
	args := m.Called(account)
	return args.Bool(0)
}

func (m *MockAuthManager) GetAccountConfigPath(account string) string {
	args := m.Called(account)
	return args.String(0)
}

func (m *MockAuthManager) SaveAPIKey(account, apiKey string) error {
	args := m.Called(account, apiKey)
	return args.Error(0)
}

func (m *MockAuthManager) GetAPIKeyFile(account string) string {
	args := m.Called(account)
	return args.String(0)
}

func (m *MockAuthManager) CopyMainConfigToAccount(account string) error {
	args := m.Called(account)
	return args.Error(0)
}

// MockMountManager is a mock implementation of MountManager
type MockMountManager struct {
	mock.Mock
}

func (m *MockMountManager) ValidateMountPath(path string) (string, error) {
	args := m.Called(path)
	return args.String(0), args.Error(1)
}

func (m *MockMountManager) AddMountToConfig(config *pkg.ContainerConfig, sourcePath, targetPath string) error {
	args := m.Called(config, sourcePath, targetPath)
	return args.Error(0)
}

func (m *MockMountManager) GetMountSummary(mounts []pkg.Mount) string {
	args := m.Called(mounts)
	return args.String(0)
}

func (m *MockMountManager) UpdateMountSettings(mountPaths []string) error {
	args := m.Called(mountPaths)
	return args.Error(0)
}

// MockArchDetector is a mock implementation of ArchDetector
type MockArchDetector struct {
	mock.Mock
}

func (m *MockArchDetector) GetHostArchitecture() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockArchDetector) GetDockerPlatform() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockArchDetector) IsMultiArchSupported() bool {
	args := m.Called()
	return args.Bool(0)
}

// MockLogger is a mock implementation of Logger
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(args ...interface{}) {
	m.Called(args...)
}

func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Info(args ...interface{}) {
	m.Called(args...)
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Warn(args ...interface{}) {
	m.Called(args...)
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Error(args ...interface{}) {
	m.Called(args...)
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Fatal(args ...interface{}) {
	m.Called(args...)
}

func (m *MockLogger) Fatalf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) WithField(key string, value interface{}) pkg.Logger {
	args := m.Called(key, value)
	return args.Get(0).(pkg.Logger)
}

func (m *MockLogger) WithFields(fields map[string]interface{}) pkg.Logger {
	args := m.Called(fields)
	return args.Get(0).(pkg.Logger)
}

// MockDevContainerManager is a mock of DevContainerManager interface
type MockDevContainerManager struct {
	mock.Mock
}

// GenerateDevContainer mocks the GenerateDevContainer method
func (m *MockDevContainerManager) GenerateDevContainer(projectPath string, config *pkg.Config) error {
	args := m.Called(projectPath, config)
	return args.Error(0)
}

// ValidateDevContainer mocks the ValidateDevContainer method
func (m *MockDevContainerManager) ValidateDevContainer(projectPath string) error {
	args := m.Called(projectPath)
	return args.Error(0)
}

// GetExtensionsForProject mocks the GetExtensionsForProject method
func (m *MockDevContainerManager) GetExtensionsForProject(projectType string, variant string) ([]string, error) {
	args := m.Called(projectType, variant)
	return args.Get(0).([]string), args.Error(1)
}

// CreateDevContainerConfig mocks the CreateDevContainerConfig method
func (m *MockDevContainerManager) CreateDevContainerConfig(config *pkg.DevContainerConfig) ([]byte, error) {
	args := m.Called(config)
	return args.Get(0).([]byte), args.Error(1)
}

// DetectProjectType mocks the DetectProjectType method
func (m *MockDevContainerManager) DetectProjectType(projectPath string) (*pkg.ProjectDetectionResult, error) {
	args := m.Called(projectPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkg.ProjectDetectionResult), args.Error(1)
}

// UpdateDevContainer mocks the UpdateDevContainer method
func (m *MockDevContainerManager) UpdateDevContainer(projectPath string, config *pkg.Config) error {
	args := m.Called(projectPath, config)
	return args.Error(0)
}

// RemoveDevContainer mocks the RemoveDevContainer method
func (m *MockDevContainerManager) RemoveDevContainer(projectPath string) error {
	args := m.Called(projectPath)
	return args.Error(0)
}