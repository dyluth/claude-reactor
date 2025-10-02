# Feature: Radical Simplification - Return to Core Mission

## 1. Description
Complete removal of feature bloat and non-core functionality to return claude-reactor to its original mission: "a simple safe way to run Claude in a Docker container with account isolation". This involves removing thousands of lines of scope creep including template management, dependency management, hot reload, build commands, and devcontainer functionality that have obscured the tool's core purpose.

## Goal statement
To restore claude-reactor to a focused, maintainable tool that does one thing well: securely running Claude CLI in Docker containers with proper account isolation, eliminating all functionality that duplicates existing tools or falls outside this core mission.

## Project Analysis & current state

### Technology & architecture
- **Current CLI**: 11 commands with ~6000+ lines of bloated functionality
- **Core Architecture**: Go-based CLI with Docker SDK integration
- **Container System**: Multi-stage Dockerfile with 5 variants (base, go, full, cloud, k8s)
- **Account Isolation**: Configuration management with per-account container naming
- **Build System**: Makefile with comprehensive targets for all operations
- **Key Bloat Files**: `template.go` (696 lines), `dependency.go` (585 lines), `hotreload.go` (~400 lines), `build.go` (~200 lines), `devcontainer.go` (~200 lines)

### current state
**Current Command Structure (BLOATED):**
- `run` ✅ Core functionality - THE ACTUAL PURPOSE
- `config` ✅ Core functionality - Account isolation support
- `clean` ✅ Core functionality - Container lifecycle management
- `build` ❌ BLOAT - Duplicates Makefile, 200+ lines of unnecessary code
- `debug` ⚠️ CONFUSING - Name conflicts with --debug flag, should be `info`
- `template` ❌ EXTREME BLOAT - 696 lines competing with cookiecutter/yeoman
- `dependency` ❌ MASSIVE BLOAT - 585 lines competing with npm/cargo/pip
- `hotreload` ❌ BROKEN BLOAT - 400+ lines of non-functional complexity
- `devcontainer` ❌ EDITOR BLOAT - VS Code specific functionality
- `completion` ✅ Standard CLI practice

**Internal Package Bloat:**
- `internal/reactor/template/` - Complete project scaffolding system
- `internal/reactor/dependency/` - Multi-language package manager wrapper
- `internal/reactor/hotreload/` - Non-functional file watching system
- `internal/reactor/devcontainer/` - VS Code integration
- Associated tests, configs, and documentation

## context & problem definition

### problem statement
**Who**: Maintainers and users of claude-reactor who need a simple, focused tool
**What**: Tool has suffered severe feature creep, becoming a complex development environment that duplicates existing tools and confuses its core purpose
**Why**: Feature bloat makes the tool harder to maintain, understand, and use, while providing no additional value over existing specialized tools

**Current Pain Points:**
- **Confused Identity**: Tool description claims to be "comprehensive development environment" instead of simple Claude runner
- **Maintenance Burden**: 4000+ lines of non-core code requiring ongoing maintenance
- **User Confusion**: 11 commands when 5 would suffice, unclear which features actually work
- **Duplicated Functionality**: Build command duplicates Makefile, dependency commands duplicate package managers
- **Broken Features**: HotReload advertised but non-functional, creating false expectations
- **Name Conflicts**: `debug` command vs `--debug` flag confusion

### success criteria
- [ ] **Focused Command Set**: Reduce from 11 to 5 essential commands (run, config, clean, info, completion)
- [ ] **Massive Code Reduction**: Remove 4000+ lines of non-core functionality
- [ ] **Clear Documentation**: README clearly states tool purpose and directs developers to Makefile for build operations
- [ ] **No Broken Features**: All remaining commands must be fully functional
- [ ] **Clean Architecture**: AppContainer and internal packages contain only core functionality
- [ ] **Zero Duplication**: No CLI commands that duplicate Makefile or standard tools
- [ ] **Simplified Onboarding**: Tool purpose and usage immediately clear to new users

## technical requirements

### functional requirements
- [ ] Remove `template` command and all template management functionality
- [ ] Remove `dependency` command and all package manager wrapper functionality
- [ ] Remove `hotreload` command and all file watching functionality
- [ ] Remove `build` command and redirect users to Makefile in documentation
- [ ] Remove `devcontainer` command and all VS Code integration
- [ ] Rename `debug` command to `info` to eliminate flag confusion
- [ ] Remove all internal packages for deleted functionality
- [ ] Remove all tests for deleted functionality
- [ ] Update AppContainer to remove unused manager fields
- [ ] Update README to direct developers to Makefile for build operations
- [ ] Clean up imports and dependencies for removed packages
- [ ] Comprehensive documentation and operational tooling via Makefile
- [ ] Developer onboarding experience <10 minutes from clone to running

### non-functional requirements
- **Simplicity**: Tool must have clear, focused purpose with minimal command surface
- **Maintainability**: Codebase must be easy to understand and modify
- **Performance**: Faster startup due to reduced initialization overhead
- **Documentation**: All operations clearly documented with Makefile as build authority
- **Operations**: All common tasks accessible via single Makefile commands
- **Documentation**: Comprehensive docs/ structure with setup, deployment, troubleshooting guides
- **Developer Experience**: <10 minutes from git clone to running locally

### Technical Constraints
- Must maintain backward compatibility for core commands (run, config, clean)
- Must preserve all Docker container functionality and account isolation
- Must maintain existing Makefile targets that replaced CLI functionality
- Cannot break existing .claude-reactor configuration files

## Data & Database changes

### Data model updates
N/A - Configuration model remains unchanged for core functionality

### Data migration plan
N/A - No data migration needed, only feature removal

## API & Backend changes

### Data access pattern
N/A - No API changes, CLI-only tool

### server actions
N/A - No server-side functionality

### Database queries
N/A - No database interaction

### API Routes
N/A - CLI tool, no API endpoints

## frontend changes

### New components
N/A - CLI tool, no frontend components

### Page updates
N/A - CLI tool, no web interface

## Implementation plan

### phase 1 - Remove Template Management System
**Goal**: Complete removal of template functionality and internal packages
- [ ] Remove `cmd/claude-reactor/commands/template.go` (696 lines)
- [ ] Remove `internal/reactor/template/` directory completely
- [ ] Remove template manager from `internal/reactor/container.go` AppContainer
- [ ] Remove template tests from `internal/reactor/template/` directories
- [ ] Remove template imports from main.go and command registration
- [ ] Update any documentation referencing template functionality

### phase 2 - Remove Dependency Management System
**Goal**: Complete removal of dependency wrapper functionality
- [ ] Remove `cmd/claude-reactor/commands/dependency.go` (585 lines)
- [ ] Remove `internal/reactor/dependency/` directory completely
- [ ] Remove dependency manager from `internal/reactor/container.go` AppContainer
- [ ] Remove dependency tests from `internal/reactor/dependency/` directories
- [ ] Remove dependency imports from main.go and command registration
- [ ] Update any documentation referencing dependency management

### phase 3 - Remove HotReload System
**Goal**: Complete removal of broken hot reload functionality
- [ ] Remove `cmd/claude-reactor/commands/hotreload.go` (~400 lines)
- [ ] Remove `internal/reactor/hotreload/` directory completely
- [ ] Remove hotreload managers from `internal/reactor/container.go` AppContainer
- [ ] Remove hotreload tests from `internal/reactor/hotreload/` directories
- [ ] Remove hotreload imports from main.go and command registration
- [ ] Remove `ai-prompts/4-hot-reload-file-watching.md` specification

### phase 4 - Remove Build Command and DevContainer
**Goal**: Remove CLI build duplication and editor-specific functionality
- [ ] Remove `cmd/claude-reactor/commands/build.go` (~200 lines)
- [ ] Remove `cmd/claude-reactor/commands/devcontainer.go` (~200 lines)
- [ ] Remove `internal/reactor/devcontainer/` directory completely
- [ ] Remove devcontainer manager from `internal/reactor/container.go` AppContainer
- [ ] Remove devcontainer tests from `internal/reactor/devcontainer/` directories
- [ ] Remove build and devcontainer imports from main.go and command registration

### phase 5 - Rename Debug Command and Clean AppContainer
**Goal**: Eliminate command/flag confusion and clean up container structure
- [ ] Rename `debug` command to `info` in `cmd/claude-reactor/commands/debug.go`
- [ ] Update command registration to use `info` instead of `debug`
- [ ] Remove unused manager fields from `pkg.AppContainer` struct
- [ ] Remove unused imports from `internal/reactor/container.go`
- [ ] Update any tests that reference the old debug command name
- [ ] Clean up any remaining dead code or unused imports

### phase 6 - Documentation and Final Cleanup
**Goal**: Update all documentation to reflect simplified tool and direct users to Makefile
- [ ] Update README.md to remove references to deleted functionality
- [ ] Add clear section directing developers to use Makefile for build operations
- [ ] Update tool description to reflect core mission only
- [ ] Remove documentation for deleted commands from any help text
- [ ] Update CLAUDE.md project guidance to reflect simplified architecture
- [ ] Remove any example usage of deleted commands
- [ ] Validate all remaining commands work correctly
- [ ] Ensure help text accurately reflects available functionality

## 5. Testing Strategy

### Unit Tests
- Remove all unit tests for deleted functionality (template, dependency, hotreload, devcontainer)
- Verify remaining core functionality tests still pass (run, config, clean, info)
- Add tests to verify deleted commands return appropriate "command not found" errors

### Integration Tests
- Remove integration tests for deleted functionality
- Verify Docker integration tests still work for core commands
- Test that AppContainer initialization works with reduced manager set

### End-to-End (E2E) Tests
- Remove E2E tests for deleted functionality
- Verify core user journey still works: configure account → run container → clean up
- Test that simplified command structure provides clear user experience

## 6. Security Considerations

### Authentication & Authorization
No changes - account isolation functionality preserved in core commands

### Data Validation & Sanitization
Reduced attack surface due to removed functionality - fewer input validation points needed

### Potential Vulnerabilities
Significant reduction in potential attack vectors due to removal of complex functionality like template generation and dependency management

## 7. Rollout & Deployment

### Feature Flags
N/A - Clean removal, no feature flags needed

### Deployment Steps
Standard deployment process - no special deployment considerations needed

### Rollback Plan
N/A - This is a simplification that cannot be easily rolled back. The removed functionality would need to be re-implemented if needed (which it shouldn't be).

## 8. Open Questions & Assumptions

### Assumptions
- No external users currently depend on deleted functionality (template, dependency, hotreload, devcontainer, build)
- Makefile contains all necessary build targets to replace CLI build command
- Core functionality (run, config, clean) provides sufficient value for tool's mission
- Users who need template/dependency management will use appropriate specialized tools
- Simplified tool will be easier to maintain and understand

### Implementation Notes
- This is a major breaking change that fundamentally changes the tool's scope
- The goal is to make it appear as if the removed functionality never existed
- All code, tests, documentation, and examples for removed features must be completely eliminated
- The result should be a clean, focused tool that clearly communicates its purpose
- README.md must clearly direct developers to Makefile for build operations