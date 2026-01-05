# AI Prompt: Multi-LLM Backend Support

## 1. Feature Description
This feature introduces a flexible, extensible architecture to support multiple Large Language Model (LLM) backends within claude-reactor, such as z.ai or others. It allows users to configure a specific LLM provider and model on a per-project, per-account basis. This is achieved by abstracting the authentication and container configuration, enabling claude-reactor to dynamically equip containers with the necessary tools and credentials for the selected LLM, while preserving the core principles of account isolation and zero-configuration experience.

## 2. Goal Statement
To enable developers to seamlessly switch between different LLM backends within claude-reactor for different projects, by introducing a configuration-driven approach that manages provider-specific authentication and container environments, thus breaking the dependency on a single LLM provider.

## Project Analysis & current state

### Technology & architecture
- **Go-based CLI**: The project is a robust CLI built with Go and the Cobra framework. The core logic for running containers is in `cmd/claude-reactor/commands/run.go`.
- **Interface-Driven Design**: Core components are defined by interfaces in `pkg/interfaces.go`, which will allow for easy extension.
- **Configuration Management**: Configuration is handled by `internal/reactor/config/manager.go`, which reads settings from `.claude-reactor` files within the session directory `~/.claude-reactor/{account}/{project-name}-{project-hash}/`. This is the ideal place to add LLM backend configuration.
- **Authentication Management**: The `internal/reactor/auth/manager.go` handles account-specific credentials. This can be extended to manage credentials for multiple LLM providers.
- **Docker Integration**: The `internal/reactor/docker/manager.go` and `Dockerfile` manage container creation. The `Dockerfile` contains the standard Claude CLI, and the `run` command will inject provider-specific environment variables to configure it.
- **Account Isolation**: The existing directory structure `~/.claude-reactor/{account}/` is a perfect foundation for storing provider-specific credentials in an isolated manner.

### current state
Currently, claude-reactor is hard-wired for the Claude CLI. Authentication is centered around the `ANTHROPIC_API_KEY` and Claude-specific configuration files (`~/.claude.json`). The container image is built with the Claude CLI pre-installed. There is no mechanism to specify or configure alternative LLM backends, forcing all projects and accounts to use the same underlying LLM provider.

## context & problem definition

### problem statement
**Who**: Developers using claude-reactor who want to work with different LLMs for different projects (e.g., using Claude for creative writing and a coding-specific model like z.ai for software development).
**What**: They are unable to switch LLM backends easily. The current system assumes a single, Claude-based environment, requiring manual and complex workarounds to even attempt using a different model, which breaks the project's core value proposition.
**Why**: This limitation restricts the tool's utility and flexibility, forcing developers to use other solutions for projects that require different LLMs and preventing claude-reactor from becoming a truly versatile development tool.

### success criteria
- [ ] Users can specify an LLM provider (e.g., `claude`, `zai`) and model in their `.claude-reactor` configuration.
- [ ] The system stores and manages authentication credentials for at least two different LLM providers in a secure, isolated manner.
- [ ] The container environment is dynamically configured with the necessary CLI tools for the selected LLM provider.
- [ ] All existing functionality (account isolation, smart container reuse, mount management) works seamlessly with the new multi-LLM architecture.
- [ ] The default behavior remains unchanged: if no provider is specified, it defaults to `claude`.

## technical requirements

### functional requirements
- [ ] Add a `--llm-provider` flag to the `run` command. Using this flag will persist the chosen provider to the project's `.claude-reactor` configuration file.
- [ ] If a new provider is specified, the tool should interactively prompt the user for credentials and save them.
- [ ] Extend the authentication manager to handle provider-specific credentials, storing them in a new directory structure like `~/.claude-reactor/{account}/llms/{provider}/`.
- [ ] Update the `run` command to inject provider-specific environment variables into the container.
- [ ] The `run` command must mount the correct, provider-specific credential files into the container.
- [ ] Implement a default provider (`claude`) if the `llm` section is not present in the configuration.
- [ ] Comprehensive documentation and operational tooling via Makefile.
- [ ] Developer onboarding experience <10 minutes from clone to running.

### non-functional requirements
- **Performance**: The overhead for selecting an LLM backend should not increase container startup time by more than 10%.
- **Security**: Provider-specific credentials must be isolated at the account and provider level with strict file permissions (0600).
- **Extensibility**: The design should make it easy to add new LLM providers in the future by creating a new provider-specific authentication module.
- **Operations**: All common tasks accessible via single Makefile commands.
- **Documentation**: Comprehensive docs/ structure with setup, deployment, troubleshooting guides.
- **Developer Experience**: <10 minutes from git clone to running locally.

### Generic Provider Support
To ensure maximum flexibility, the system will support both "known" and "generic" LLM providers without requiring code changes to add new providers.

- **Known Providers**: A small, optional, hardcoded map can be maintained within the application for common providers (e.g., a map of `provider_name` to `api_key_variable_name`, like `"claude": "ANTHROPIC_API_KEY"`). This allows for a slightly more user-friendly prompt (e.g., "Please enter your Anthropic API key:").
- **Generic Providers**: If a user specifies a provider *not* in the known list, the tool will fall back to a generic interactive setup, asking the user for both the name of the API key environment variable and its value. This allows users to configure any new or custom LLM backend on the fly.
- **Credential File**: The interactive setup will create a standard `.env` file in the provider's credential directory (`~/.claude-reactor/{account}/llms/{provider}/.env`). Users can manually add more environment variables (like model names or API base URLs) to this file later if needed.

### Technical Constraints
- The solution must not break backward compatibility with existing `.claude-reactor` configurations.
- The implementation must adhere to the existing interface-driven design.
- The solution should avoid creating a separate Docker image variant for each LLM provider to prevent image proliferation. Using a single, adaptable image is preferred.

## Data & Database changes

### Data model updates
**New Configuration Structure (`.claude-reactor` file):**
```yaml
variant: go
account: cam
danger: false
llm:
  provider: zai # Defaults to "claude" if not specified
  model: glm-4.6 # Optional: model name/version
```

**New Directory Structure for Credentials:**
```
~/.claude-reactor/
├── {account}/
│   ├── llms/
│   │   ├── claude/
│   │   │   ├── .claude.json
│   │   │   └── .claude-reactor-claude-env
│   │   └── zai/
│   │       └── .zai_env # Or whatever z.ai requires
│   ├── {project-name}-{project-hash}/
│   │   └── .claude-reactor
...
```

### Data migration plan
N/A. The new configuration fields are optional. Existing configurations will continue to work by defaulting to the `claude` provider. The new credential directory structure will be created as needed.

## API & Backend changes

### Data access pattern
The `ConfigManager` in `internal/reactor/config/manager.go` will be updated to parse the new `llm` section from the YAML configuration. The `AuthManager` will use the `llm.provider` field to determine which credential path to use.

### server actions
**`internal/reactor/config/manager.go`**
- Modify the `Config` struct in `pkg/interfaces.go` to include the `LLMConfig` struct (`provider` and `model` strings).
- Update `LoadConfig` to correctly parse the `llm` section from YAML and set the default provider to `claude` if not present.

**`internal/reactor/auth/manager.go`**
- Create a new method `EnsureProviderCredentials(account, provider string) error`. This method checks if a provider's `.env` file exists. If not, it should trigger the interactive setup (generic or known) and save the credentials.
- Create a method `GetProviderEnv(account, provider string) ([]string, error)` that reads the provider's `.env` file and returns its contents as a slice of strings in `KEY=VALUE` format, ready for container injection.
- Create a method `GetProviderMounts(account, provider string) []string` that returns a list of provider-specific files to mount (e.g., `~/.claude.json` for the `claude` provider).

**`cmd/claude-reactor/commands/run.go`**
- The main `Run` function will orchestrate the calls. It will first call `EnsureProviderCredentials`, then `GetProviderEnv` and `GetProviderMounts`, and pass the results to the Docker manager when creating the container.

## frontend changes
N/A - CLI application.

## Implementation plan

### phase 1 - Configuration and Credential Structure
**Goal**: Update the configuration and create the new credential directory structure. All modifications will be in the `internal/` directory.
- [ ] Modify the `Config` and add the `LLMConfig` struct in `pkg/interfaces.go`.
- [ ] Update the `LoadConfig` method in `internal/reactor/config/manager.go` to parse the new `llm` section and apply default values.
- [ ] Implement the new credential directory logic in `internal/reactor/auth/manager.go` by adding a `GetProviderCredentialPath` method to generate provider-specific paths (e.g., `~/.claude-reactor/account/llms/provider`).
- [ ] Write unit tests for the new configuration parsing and path generation in `internal/reactor/config/manager_test.go` and `internal/reactor/auth/manager_test.go`.

### phase 2 - Environment Variable Management
**Goal**: Configure the container with the correct environment variables for the selected LLM provider.
- [ ] Confirm that the `Dockerfile` requires no modifications. The standard Claude CLI is always used.
- [ ] In `internal/reactor/auth/manager.go`, implement the logic to locate and read the provider-specific `.env` file (e.g., `~/.claude-reactor/{account}/llms/{provider}/.env`).
- [ ] The `run` command logic in `cmd/claude-reactor/commands/run.go` will be responsible for taking the variables read by the `AuthManager` and adding them to the container's configuration before it is created.
- [ ] This includes ensuring that any special environment variables needed to point the Claude CLI to a different API endpoint, like `ANTHROPIC_API_BASE`, are passed to the container.

### phase 3 - Integrating Logic in the `run` Command
**Goal**: Tie the configuration, credential prompting, and environment variable injection together within the `run` command.
- [ ] In `cmd/claude-reactor/commands/run.go`, add the `--llm-provider` string flag.
- [ ] In the `Run` function of the `run` command, add a pre-flight check. If the `--llm-provider` flag is used, call a new method on the `AuthManager` (e.g., `EnsureProviderCredentials`) to handle the interactive prompt and credential saving if the provider is new.
- [ ] After the check, use the `ConfigManager` to persist the provider choice to the project's `.claude-reactor` file.
- [ ] Before creating the container, fetch the provider-specific environment variables using the `AuthManager` and add them to the `docker.ContainerCreate` call.

### phase 4 - Documentation and Operations
**Goal**: Document the new feature and ensure it's easy to use.
- [ ] Create `docs/multi-llm-support.md` explaining how to configure and use different LLM backends.
- [ ] Provide examples for configuring `claude` and `zai`.
- [ ] Update the main `README.md` to mention the new capability.
- [ ] Implement comprehensive Makefile with all development, testing, and deployment commands.
- [ ] Validate <10 minute developer onboarding experience.
- [ ] Document troubleshooting procedures for common issues (e.g., missing credentials, failed CLI installation).

## 5. Testing Strategy
### Unit Tests
- **Config Manager**: Test loading configurations with and without the `llm` section, and verify default values.
- **Auth Manager**: Test the generation of credential paths for different accounts and providers.
- **Run Command**: Test the logic that prepares mounts and build arguments based on the loaded configuration.

### Integration Tests
- **Environment Injection**: Verify that for a given provider, the correct environment variables from the provider-specific `.env` file are present inside the running container.
- **Container Run**: Run a container configured for `claude` and verify it works. Run a container configured for a mock `zai` provider and verify the correct environment variables are set.
- **Credential Mounting**: Verify that for a given provider, the correct credential files (like `.claude.json`) are mounted into the container at the expected location with the correct permissions.

### End-to-End (E2E) Tests
- **Full Claude Workflow**: Configure a project to use the default `claude` provider. Run the container, execute a `claude` command, and verify it works.
- **Full z.ai Workflow (First Time Setup)**: In a new project, run `claude-reactor run --llm-provider=zai`. The test should mock user input to provide a test API key at the interactive prompt. It should then verify that the `.claude-reactor` file is updated and that the `ZAI_API_KEY` environment variable is correctly injected into the container.
- **Switching Workflow**: Use the same account but switch between a `claude` project and a `zai` project, verifying that the correct container environment (via environment variables) and credentials are used each time.

## 6. Security Considerations
### Authentication & Authorization
- Credentials for each LLM provider will be stored in separate, account-locked directories (`~/.claude-reactor/{account}/llms/{provider}/`).
- File permissions for all credential files will be set to `0600` to restrict access.

### Data Validation & Sanitization
- The `llm.provider` name from the configuration will be validated against an allow-list of supported providers to ensure the correct credential paths are used.
- Any paths derived from configuration will be sanitized.

### Potential Vulnerabilities
- **Credential Leakage**: A misconfigured mount could expose credentials. The new, more organized credential structure and rigorous testing will mitigate this.

## 7. Rollout & Deployment
### Feature Flags
N/A. This feature is designed to be backward-compatible. Existing configurations will work as before, defaulting to the `claude` provider.

### Deployment Steps
Standard deployment process. The changes are self-contained and don't require special deployment steps.

### Rollback Plan
If issues arise, a standard rollback of the codebase via git is sufficient. Since the feature is opt-in through configuration and backward-compatible, a rollback is low-risk.

## 8. Open Questions & Assumptions
- **Assumption**: Most LLM backends can be configured by pointing the Claude CLI to a different API endpoint and providing an API key, all through environment variables.
- **Assumption**: The credentials for different LLMs are primarily file-based (config files, env files) and can be mounted into the container.
- **Question**: How should we handle LLMs that require more complex, non-file-based authentication? (This can be deferred until such a provider is needed).
- **Question**: For the `z.ai` example, what specific environment variables are needed to configure the Claude CLI to use it as a backend? Further research will be needed for each new provider.
