# Feature: Project Templates & Scaffolding

## 1. Description
Create an intelligent project scaffolding system that generates optimized development environments based on project type, automatically creating appropriate directory structures, configuration files, and development workflows. This eliminates the "blank canvas" problem and provides developers with production-ready project foundations that include claude-reactor integration, testing frameworks, and deployment configurations.

## Goal statement
To provide developers with intelligent, one-command project initialization that creates production-ready development environments with optimal tooling, testing, and deployment configurations tailored to their specific technology stack.

## Project Analysis & current state

### Technology & architecture
- **Current claude-reactor**: Go-based CLI with auto-detection for 5 container variants (base, go, full, cloud, k8s)
- **Project Detection Logic**: Existing auto-detection in `internal/config/manager.go` for Go, Rust, Node.js, Python, Java, K8s projects
- **Template System**: No current scaffolding system - opportunity to build from scratch
- **Configuration Management**: Existing `.claude-reactor` configuration system for project preferences
- **Container Integration**: Established Docker SDK integration for seamless environment setup
- **Build Automation**: Makefile system with 25+ targets for professional development workflows
- **Key Files**: `internal/config/manager.go`, `claude-reactor` CLI, project detection logic, Dockerfile variants

### current state
**Current Developer Workflow:**
1. Developer creates empty project directory
2. Manually initializes language-specific files (go.mod, package.json, etc.)
3. Manually runs `./claude-reactor run` for container setup
4. Manually creates development tooling (Makefile, tests, CI/CD, etc.)
5. Manual integration of best practices and project structure

**Missing Scaffolding Integration:**
- No project template system or intelligent initialization
- No automated best practice integration (testing, linting, formatting)
- No development workflow templates (Makefile, scripts, CI/CD)
- Manual setup creates inconsistent project structures across teams
- No integration between project templates and claude-reactor variants

## context & problem definition

### problem statement
**Who**: Developers and teams starting new projects who want consistent, production-ready development environments
**What**: Currently face setup friction, inconsistent project structures, missing best practices, and manual integration of development tooling
**Why**: Manual project setup leads to inconsistent structures, forgotten best practices, and significant time investment before actual development can begin

**Current Pain Points:**
- **Setup Friction**: 30+ minutes to set up proper project structure and tooling
- **Inconsistent Standards**: Different developers create different project structures
- **Missing Best Practices**: Forgotten linting, formatting, testing setup, and CI/CD integration
- **Tool Integration**: Manual claude-reactor setup and variant selection
- **Team Onboarding**: New team members must learn project structure patterns

### success criteria
- [ ] **One-Command Initialization**: Complete project ready within 60 seconds of running template command
- [ ] **Intelligent Detection**: Automatically selects optimal claude-reactor variant based on template choice
- [ ] **Best Practice Integration**: Templates include testing, linting, formatting, and CI/CD configurations
- [ ] **Team Consistency**: Same project structure for all team members regardless of template timing
- [ ] **Production Ready**: Generated projects include deployment configurations and operational tooling
- [ ] **Extensibility**: Easy to add custom templates and modify existing ones

## technical requirements

### functional requirements
- [ ] **Template Engine**: Flexible templating system supporting variable substitution and conditional logic
- [ ] **Multi-Language Support**: Templates for Go, Rust, Node.js, Python, Java, and hybrid projects
- [ ] **Claude-Reactor Integration**: Automatic `.claude-reactor` configuration with optimal variant selection
- [ ] **Development Tooling**: Automated Makefile, testing framework, linting, and formatting setup
- [ ] **CI/CD Templates**: GitHub Actions, GitLab CI, and other pipeline configurations
- [ ] **Documentation Generation**: Automatic README, CONTRIBUTING, and development documentation
- [ ] **License Integration**: Support for common open source licenses with proper attribution
- [ ] **Git Integration**: Automated git initialization, .gitignore, and commit hooks setup
- [ ] **Environment Configuration**: Docker Compose, environment variables, and secrets management templates
- [ ] **Interactive Mode**: Guided project creation with questions and recommendations
- [ ] Comprehensive documentation and operational tooling via Makefile
- [ ] Developer onboarding experience <10 minutes from clone to running

### non-functional requirements
- **Performance**: Template generation and project initialization within 60 seconds
- **Reliability**: 99% successful project generation across all supported templates and platforms
- **Usability**: Intuitive CLI interface with clear prompts and helpful defaults
- **Maintainability**: Template system that's easy to update, extend, and customize
- **Compatibility**: Works across macOS, Linux, and Windows (WSL2) environments
- **Extensibility**: Plugin architecture for custom templates and organizational standards
- **Operations**: All common tasks accessible via single Makefile commands
- **Documentation**: Comprehensive docs/ structure with setup, deployment, troubleshooting guides
- **Developer Experience**: <10 minutes from git clone to running locally

### Technical Constraints
- Must integrate seamlessly with existing claude-reactor architecture and variant system
- Template files must be embedded in Go binary or easily distributedå
- Must work with existing project detection logic without conflicts
- Cannot break existing project workflows or configurations
- Must support both interactive and non-interactive (CI/CD) usage
- Template system must be maintainable by developers without Go template expertise
- Must respect existing `.claude-reactor` configurations in existing projects

## Data & Database changes

### Data model updates
**Template Metadata Structure:**
```go
type ProjectTemplate struct {
    ID          string
    Name        string
    Description string
    Category    string
    Languages   []string
    Variant     string // claude-reactor variant (base, go, full, cloud, k8s)
    Variables   map[string]VariableConfig
    Files       []TemplateFile
}

type VariableConfig struct {
    Type        string // string, bool, select
    Description string
    Default     string
    Options     []string // for select type
    Required    bool
}
```

### Data migration plan
N/A - New feature, no existing data to migrate.

## API & Backend changes

### Data access pattern
Template files will be embedded in the Go binary using `embed` package for easy distribution.

### server actions
N/A - Client-side CLI feature, no server-side components.

### Database queries
N/A - No database interaction required.

### API Routes
N/A - CLI-based feature, no API endpoints.

## frontend changes

### New components
N/A - CLI-based tool with no web frontend components.

### Page updates
N/A - Command-line interface updates only.

## Implementation plan

### phase 1 - Template Engine Foundation
**Goal**: Build core templating system with variable substitution and file generation
- [ ] Design template metadata format and validation schema
- [ ] Implement Go template engine with custom functions and helpers
- [ ] Create template variable collection system (interactive and non-interactive)
- [ ] Add template file embedding system using Go embed package
- [ ] Implement basic template validation and error handling
- [ ] Create initial template structure for Go and Node.js projects

### phase 2 - CLI Integration
**Goal**: Integrate template system with claude-reactor CLI and project detection
- [ ] Add `claude-reactor init` command with template selection
- [ ] Implement template listing and description display
- [ ] Add interactive template variable collection with prompts
- [ ] Integrate with existing project detection logic to prevent conflicts
- [ ] Add automatic `.claude-reactor` configuration generation with optimal variant selection
- [ ] Implement non-interactive mode for CI/CD and automation usage

### phase 3 - Comprehensive Template Library
**Goal**: Create production-ready templates for all supported languages and frameworks
- [ ] Create comprehensive Go project templates (CLI, web service, library)
- [ ] Add Rust project templates (CLI, web service, library)
- [ ] Implement Node.js templates (Express, React, Vue, CLI tools)
- [ ] Add Python templates (FastAPI, Flask, Django, CLI, data science)
- [ ] Create Java templates (Spring Boot, CLI, library)
- [ ] Add specialized templates (Kubernetes operators, cloud functions, etc.)

### phase 4 - Development Workflow Integration
**Goal**: Include comprehensive development tooling and best practices in all templates
- [ ] Create Makefile templates with development, testing, and deployment targets
- [ ] Add testing framework setup (Go: testify, Node.js: Jest, Python: pytest, etc.)
- [ ] Implement linting and formatting configuration (golangci-lint, ESLint, Black, etc.)
- [ ] Add pre-commit hooks and Git configuration templates
- [ ] Create CI/CD pipeline templates (GitHub Actions, GitLab CI, CircleCI)
- [ ] Add Docker and Docker Compose configurations for development environments

### phase 5 - Advanced Features & Customization
**Goal**: Enable team customization and advanced template features
- [ ] Implement custom template directory support for organizational standards
- [ ] Add template inheritance and composition system
- [ ] Create template update and upgrade mechanisms
- [ ] Add validation for generated projects (syntax, dependencies, etc.)
- [ ] Implement template versioning and compatibility checking
- [ ] Add analytics and usage tracking for template optimization

### phase 6 - Documentation & Developer Experience
**Goal**: Comprehensive documentation and smooth onboarding experience
- [ ] Create comprehensive template development guide
- [ ] Document template variable system and customization options
- [ ] Add template creation tutorial and best practices guide
- [ ] Create troubleshooting guide for common template issues
- [ ] Document team workflow integration and template sharing
- [ ] Validate <10 minute developer onboarding experience with templates
- [ ] Create video tutorials for template usage and customization

## 5. Testing Strategy

### Unit Tests
- **Template Engine**: Test variable substitution, conditionals, and file generation
- **Template Validation**: Test template metadata parsing and validation
- **Variable Collection**: Test interactive and non-interactive variable gathering
- **CLI Integration**: Test command parsing, template selection, and project generation
- **File Operations**: Test directory creation, file writing, and permission handling

### Integration Tests
- **End-to-End Generation**: Test complete project generation for each template
- **Claude-Reactor Integration**: Test automatic variant selection and configuration generation
- **Multi-Platform**: Test template generation across macOS, Linux, and Windows (WSL2)
- **Git Integration**: Test repository initialization and configuration
- **Development Workflow**: Test generated Makefile targets and tooling integration

### End-to-End (E2E) Tests
- **Complete Developer Workflow**: Template selection → generation → development → testing → deployment
- **Multi-Template Testing**: Generate and validate projects from each available template
- **Team Collaboration**: Test shared template usage and customization across team members
- **CI/CD Integration**: Test non-interactive template generation in automated environments
- **Template Updates**: Test template upgrade and migration workflows

## 6. Security Considerations

### Authentication & Authorization
- Template system operates locally with no authentication required
- Custom template directories should respect filesystem permissions
- No network access required for core template functionality

### Data Validation & Sanitization
- **Template Variable Validation**: Sanitize all user inputs to template variables
- **File Path Validation**: Ensure generated file paths cannot escape project directory
- **Template Content Scanning**: Validate template files don't contain malicious content
- **Dependency Validation**: Check generated project dependencies against known vulnerabilities

### Potential Vulnerabilities
- **Path Traversal**: Malicious templates could attempt to write files outside project directory - mitigation: strict path validation
- **Code Injection**: Template variables could inject malicious code - mitigation: input sanitization and safe templating
- **Dependency Confusion**: Generated projects might include malicious dependencies - mitigation: use trusted dependency sources
- **Resource Exhaustion**: Large templates could consume excessive resources - mitigation: resource limits and validation

## 7. Rollout & Deployment

### Feature Flags
No feature flags needed - this is an additive feature that doesn't change existing workflows.

### Deployment Steps
1. **Template Development**: Create and test initial template library
2. **CLI Integration**: Add template commands to claude-reactor CLI
3. **Documentation**: Complete template usage and development guides
4. **Beta Testing**: Test with development teams for feedback and iteration
5. **Public Release**: Announce template system and provide migration guides

### Rollback Plan
- **No Impact Rollback**: Feature is additive - simply don't use template commands
- **Template Removal**: Remove template commands from CLI if needed
- **Full Backwards Compatibility**: All existing claude-reactor workflows remain unchanged
- **Generated Project Independence**: Generated projects work independently of template system

## 8. Open Questions & Assumptions

### Open Questions
1. Should templates be embedded in binary or downloaded from registry/repository?
2. How do we handle template versioning and backwards compatibility?
3. Should we support template composition (multiple templates in one project)?
4. What's the best approach for organizational template distribution and sharing?
5. Should templates include IDE-specific configurations (VS Code, IntelliJ)?
6. How do we handle template updates for existing projects?
7. What level of customization should be built into core templates vs custom templates?

### Assumptions
- Developers prefer opinionated templates with best practices over minimal templates
- Template generation time under 60 seconds is acceptable for the value provided
- Teams will want to customize templates for organizational standards
- Generated projects should be independent of the template system after creation
- Template system should be extensible for community contributions
- Interactive template configuration is preferred over extensive command-line flags