# **Feature 6: distributed MCP orchestration system**

## **1\. Description**

This document outlines the scope for **Reactor-Fabric**, a distributed AI operating system designed to orchestrate a suite of specialised, containerised AI agents. The system will dynamically spawn and manage these agents on-demand, based on a declarative YAML configuration, and is designed to handle multiple, concurrent client connections. This moves beyond single-agent interactions to enable complex, collaborative workflows where multiple AI specialists work in concert. A key principle of this architecture is its "serverless" nature: a fresh, ephemeral container is created for every MCP request, ensuring perfect task isolation and context preservation. The core value is to create a powerful, scalable, and context-aware environment for advanced AI-driven development and automation.

## **Goal statement**

To create a standalone system that allows a user to define a suite of specialised AI agents in a single YAML file and orchestrate them to perform complex tasks via on-demand, containerised MCP servers.

## **Project Analysis & current state**

### **Technology & architecture**

The project will be built as an extension of the existing **claude-reactor** ecosystem. The core technology stack is:

* **Language**: Go  
* **CLI Framework**: Cobra (as used in claude-reactor)  
* **Containerisation**: Docker (specifically targeting Docker Desktop for the initial implementation)  
* **Container Orchestration**: Docker SDK for Go will be used to programmatically manage the lifecycle of agent containers.  
* **Communication Protocol**: Model Context Protocol (MCP) will be the standard for inter-agent and client-orchestrator communication.

The new central component will be the **Reactor-Fabric Orchestrator**, a new Go binary responsible for parsing the suite configuration and managing the agent containers.

### **current state**

Currently, claude-reactor operates on a single-container model. A user interacts with one Claude instance within one Docker container at a time. While powerful, this architecture limits tasks to the context window and capabilities of a single agent. There is no native mechanism for coordinating multiple, specialised claude-reactor instances or other MCP-based tools in a collaborative fashion.

## **context & problem definition**

### **problem statement**

Sophisticated software development tasks (e.g., building a full-stack feature, running a security audit, performing a complex refactor) often require multiple domains of expertise (frontend, backend, database, security, testing). A single LLM agent, constrained by a finite context window, struggles to manage the complexity and specialised knowledge required for all these domains simultaneously. This leads to context loss, inefficient workflows, and a ceiling on the complexity of tasks that can be automated.

### **success criteria**

* A user can define a suite of at least three distinct MCP services in a claude-mcp-suite.yaml file and start the orchestrator with a single command.  
* A standard claude-reactor client can successfully register with the orchestrator and request an MCP service, which is then dynamically spawned.  
* The dynamically spawned container can successfully access the same project directory as the client that requested it.  
* The orchestrator successfully tears down the MCP server container after the task is complete.  
* A second, independent claude-reactor client can connect to the same orchestrator and successfully use its services concurrently without interfering with the first client.

## **technical requirements**

### **functional requirements**

* \[ \] **Backward Compatibility**: The existing single-container functionality of claude-reactor must remain completely unchanged and fully operational. The reactor-fabric orchestration is a purely additive feature. The claude-reactor binary must not have a hard dependency on the reactor-fabric orchestrator and must function perfectly when it is not present.  
* \[ \] The **Reactor-Fabric Orchestrator** must be implemented as a standalone Go CLI application.  
* \[ \] The orchestrator must be able to parse a claude-mcp-suite.yaml file that defines a collection of available MCP services.  
* \[ \] The orchestrator must handle multiple, concurrent client connections, maintaining a separate session and file system context for each one.  
* \[ \] Each service definition in the YAML must specify the Docker image to use. It should also support passing through generic configuration, which for claude-reactor based services would include settings like account and danger mode, and for third-party servers would be specific to their needs.  
* \[ \] The orchestrator will expose an MCP endpoint for clients to connect to.  
* \[ \] The orchestrator must support configurable network settings (e.g., via CLI flags or environment variables) to allow it to run in custom Docker networks and listen on non-default ports, enabling isolated test instances.  
* \[ \] When a client requests a tool provided by a configured MCP service, the orchestrator must dynamically start a **new** Docker container for that service.  
* \[ \] The client (claude-reactor) must pass its file system context to the orchestrator by calling the fabric/registerClient tool immediately after connecting.  
* \[ \] **Crucially, the orchestrator must use the received context to ensure any spawned container has access to the same project files as the client container that initiated the request.**  
* \[ \] The orchestrator must proxy the MCP communication between the client and the newly spawned container.  
* \[ \] The orchestrator must shut down and remove service containers. The default behaviour will be a teardown after 1 minute of idle activity, and this timeout should be configurable in the claude-mcp-suite.yaml.  
* \[ \] If the Docker daemon is unresponsive or returns an error, the orchestrator must handle this gracefully, creating detailed error logs and returning a defined JSON-RPC 2.0 Error Object to the client.  
* \[ \] Comprehensive documentation and operational tooling via Makefile.  
* \[ \] Developer onboarding experience \<10 minutes from clone to running.

### **non-functional requirements**

* **Performance**:  
  * **P95 Container Start Time**: The 95th percentile for the duration between the orchestrator receiving a tool request and the target container's health check passing must be **under 3.5 seconds**.  
  * **P99 Proxy Latency**: The overhead introduced by the orchestrator's proxy layer must not exceed **50 milliseconds** at the 99th percentile.  
  * **Memory Overhead**: Orchestrator memory usage \<200MB; individual agent containers \<500MB.  
* **Scalability**:  
  * The system must support 10+ concurrent specialised agent containers on a typical developer machine (16GB RAM, 4-8 CPU cores).  
  * The architecture must be designed to be compatible with a future Kubernetes deployment model.  
* **Reliability**:  
  * **Service Discovery**: 99.9% success rate for client-to-orchestrator connections.  
  * **Graceful Degradation**: Failure of a single agent container must not crash the orchestrator. The error should be isolated and reported to the relevant client.  
* **Operations**: The Makefile must include a make validate-config target that checks a given claude-mcp-suite.yaml for schema and logical errors.  
* **Developer Experience**: The \<10 minutes onboarding time must be validated by an automated script (make test-onboarding).

### **Technical Constraints**

* The initial implementation will target **Docker Desktop** on macOS and Linux. Windows support is a non-goal for the first version.  
* The system will rely on the user having Docker installed and the Docker daemon running.

## **Data & Database changes**

### **Data model updates**

#### **Configuration & State Data Models**

The following Go structs will be used to parse the YAML configuration and manage the orchestrator's in-memory state. These should be defined in a shared pkg directory.

package types

import "time"

// MCPSuite is the top-level structure for the claude-mcp-suite.yaml file.  
type MCPSuite struct {  
	Version      string                \`yaml:"version"\`  
	Orchestrator OrchestratorConfig    \`yaml:"orchestrator"\`  
	Services     map\[string\]MCPService \`yaml:"mcp\_services"\`  
}

// OrchestratorConfig holds global settings for the orchestrator itself.  
type OrchestratorConfig struct {  
	AllowedMountRoots \[\]string \`yaml:"allowed\_mount\_roots"\`  
}

// MCPService defines a single, orchestrable MCP agent.  
type MCPService struct {  
	Image   string                 \`yaml:"image"\`  
	Config  map\[string\]interface{} \`yaml:"config,omitempty"\`  
	Timeout string                 \`yaml:"timeout,omitempty"\` // e.g., "1m", "5m30s"  
}

// ClientContext holds the state for a single connected client session.  
type ClientContext struct {  
	SessionID string  
	Mounts    \[\]Mount  
}

// Mount defines a single volume mount.  
type Mount struct {  
	Source   string  
	Target   string  
	ReadOnly bool  
}

### **Data migration plan**

N/A.

## **API & Backend changes**

### **Data access pattern**

The orchestrator will interact with the Docker daemon via the official Docker Go SDK.

### **server actions**

The **Reactor-Fabric Orchestrator** will implement the following core logic:

* startSuite(configFile): Parses the YAML file, validates the configuration, and starts the main orchestrator loop.  
* handleClientConnection(connection): Manages an incoming MCP connection, creating a unique session ID and awaiting the fabric/registerClient call.  
* requestMcpService(sessionID, serviceName): The core orchestration function. It uses the sessionID to look up the correct client context (mounts), starts the container, and establishes a proxied connection.  
* cleanupContainer(containerID): Stops and removes a container after its task is complete or it times out.

### **API Routes & MCP Tools**

The orchestrator will expose a single MCP endpoint. Its primary API is the set of tools it exposes and the errors it returns.

#### **Custom MCP Tool: fabric/registerClient**

This tool must be called by a client as the first action in a new session to register its context.

* **Input Schema (JSON Schema):**  
  {  
    "type": "object",  
    "properties": {  
      "mounts": {  
        "type": "array",  
        "items": {  
          "type": "object",  
          "properties": {  
            "source": { "type": "string", "description": "The absolute path on the host machine." },  
            "target": { "type": "string", "description": "The absolute path inside the container." },  
            "readOnly": { "type": "boolean", "default": false }  
          },  
          "required": \["source", "target"\]  
        }  
      }  
    },  
    "required": \["mounts"\]  
  }

#### **Custom JSON-RPC 2.0 Error Codes**

| Code | Message | Meaning |
| :---- | :---- | :---- |
| \-32001 | Docker Daemon Unresponsive | The orchestrator cannot connect to the Docker socket. |
| \-32002 | Container Start Failure | The requested container failed to start. |
| \-32003 | Invalid Suite Configuration | The claude-mcp-suite.yaml is malformed or invalid. |
| \-32004 | Security Violation | An invalid operation was attempted (e.g., disallowed mount). |
| \-32005 | Service Not Found | The requested MCP service is not defined in the suite. |

## **frontend changes**

N/A.

## **Implementation plan**

### **Phase 0 \- Prerequisite Repository Refactoring**

* **Goal**: Restructure the existing claude-reactor repository to cleanly support multiple applications (claude-reactor and reactor-fabric) from a shared codebase. This phase must be completed before any new feature development begins.  
* \[ \] Move the main claude-reactor entrypoint and logic into cmd/claude-reactor/ and internal/reactor/ respectively.  
* \[ \] Update the Makefile to support distinct build targets for different applications (e.g., make build-reactor, make build-fabric). The default make build should build all applications.  
* \[ \] **Gate**: All existing unit, integration, and E2E tests for claude-reactor must run and pass after the refactoring is complete. No work on subsequent phases should begin until this is achieved.

### **phase 1 \- The Core Orchestrator**

* **Goal**: Create the foundational orchestrator capable of parsing a config and spawning a single, pre-defined container.  
* \[ \] Set up the new Go project structure for reactor-fabric under cmd/reactor-fabric/ and internal/fabric/.  
* \[ \] Implement the YAML parsing logic using the defined Go structs.  
* \[ \] Implement the core logic to start and stop containers using the Docker Go SDK.  
* \[ \] Add reactor-fabric targets to the Makefile for building and running.

### **phase 2 \- MCP Proxying & Client Integration**

* **Goal**: Enable a claude-reactor client to register and use a dynamically spawned MCP service.  
* \[ \] Implement the MCP server endpoint on the orchestrator, capable of managing multiple concurrent sessions.  
* \[ \] Implement the fabric/registerClient tool and its associated logic.  
* \[ \] Implement the connection proxying logic to route messages between a specific client and its dedicated backend container.  
* \[ \] Modify the container spawning logic to use the session's registered volume mounts.  
* \[ \] Test the end-to-end flow with a single client.

### **phase 3 \- Documentation & Operations**

* **Goal**: Harden the project with robust documentation, operational tooling, and multi-client support.  
* \[ \] Create docs/ structure with README.md, DEVELOPMENT.md, and USAGE.md guides.  
* \[ \] The USAGE.md guide must include the claude-mcp-suite.yaml schema and the initial service suite template.  
* \[ \] Implement the make validate-config and make test-onboarding targets.  
* \[ \] Update E2E tests to validate concurrent client scenarios.  
* \[ \] Document troubleshooting procedures.

## **Testing Strategy**

### **Unit Tests**

* Test the YAML parsing logic with valid and invalid configuration files.  
* Test the session management logic for multiple clients.  
* Test individual functions within the orchestrator, mocking the Docker SDK.

### **Integration Tests**

* Test the orchestrator's interaction with a live Docker daemon, including error handling when the daemon is unavailable.  
* Test the full MCP proxying logic with an embedded client and server.

### **End-to-End (E2E) Tests**

* The E2E test script will be updated to:  
  1. Start the orchestrator.  
  2. Start **Client A**, have it register and invoke a tool.  
  3. While Client A's container is running, start **Client B**, have it register with a *different* mount path and invoke a tool.  
  4. Assert that both clients receive correct responses and that two separate service containers were created with the correct, distinct mounts.  
  5. Assert that both containers are cleaned up correctly.

## **Security Considerations**

### **Authentication & Authorization**

For the initial version, the orchestrator's MCP endpoint will be unauthenticated and bound to localhost by default.

### **Data Validation & Sanitization**

* The orchestrator must strictly validate the claude-mcp-suite.yaml file.  
* **Mount Path Validation**: The orchestrator must have its own configuration defining an **allowlist** of permissible host directory roots (e.g., \["/Users/", "/home/"\]). When it receives a fabric/registerClient call, it must resolve the source path and verify it is a subdirectory of an allowed root. Any request to mount a path outside these roots must be rejected with error code \-32004.

### **Potential Vulnerabilities**

The primary vulnerability is **privilege escalation via Docker socket**. The orchestrator requires access to the host's Docker socket. The documentation must clearly state this security consideration. The orchestrator code must be carefully reviewed to ensure it only performs the intended actions.

## **Initial MCP Service Suite**

To provide a useful out-of-the-box experience, the project documentation will include a template claude-mcp-suite.yaml file with definitions for several popular, high-quality, third-party MCP servers.

### **Example claude-mcp-suite.yaml**

\# Reactor-Fabric Service Suite Configuration  
version: "1.0"

\# Global settings for the orchestrator  
orchestrator:  
  \# Allowed host paths that clients can request to mount.  
  \# IMPORTANT: For security, keep this as restrictive as possible.  
  allowed\_mount\_roots:  
    \- "/home/"  
    \- "/Users/"  
    \- "/mnt/"  
    \- "/media/"  
    \- "/projects/"

\# Available MCP service definitions  
mcp\_services:  
  \# Provides basic filesystem operations.  
  filesystem:  
    image: "ghcr.io/modelcontextprotocol/server-filesystem:latest"  
    timeout: "1m" \# Default idle timeout

  \# Provides tools for interacting with a Git repository.  
  git:  
    image: "ghcr.io/modelcontextprotocol/server-git:latest"  
    timeout: "5m"

  \# Provides tools for running shell commands.  
  \# Note: This is powerful and potentially dangerous.  
  shell:  
    image: "ghcr.io/modelcontextprotocol/server-shell:latest"  
    timeout: "1m"

  \# An example of a claude-reactor based service.  
  \# This agent would be specialised for Python development.  
  python\_expert:  
    image: "ghcr.io/dyluth/claude-reactor/python:latest"  
    timeout: "10m"  
    config:  
      account: "work\_account"  
      danger\_mode: false  
      specialty: "Expert in Python, Django, and data analysis."

## **Rollout & Deployment**

### **Deployment Steps**

Deployment will consist of building the Go binary for reactor-fabric and providing it to users, likely via a GitHub Release.

### **Rollback Plan**

N/A for the initial release. The additive nature of the feature means users can simply not run the reactor-fabric binary to revert to the classic claude-reactor workflow.

## **Enhanced Technical Architecture**

### **Architecture Clarifications**

**Current claude-reactor**: The existing system is a Go application (not a bash script) built from `cmd/` directory with comprehensive CLI functionality, account isolation, and professional automation. The INSTALL script is a shell utility for installation only.

**Repository Strategy**: The system will be refactored to support multiple Go applications from a shared codebase:
- `cmd/claude-reactor/` - Existing claude-reactor entrypoint 
- `cmd/reactor-fabric/` - New orchestrator entrypoint
- `internal/reactor/` - Claude-reactor specific logic
- `internal/fabric/` - Orchestrator specific logic  
- `pkg/` - Shared data structures and utilities

**Backward Compatibility**: New functionality is purely additive. Claude-reactor must continue to function perfectly in standalone mode when reactor-fabric is not available.

### **MCP Integration Strategy**

**Go Libraries**: Official Go SDKs and reference implementations from the Model Context Protocol project (`modelcontextprotocol/servers` repository patterns).

**Transparent Proxying**: The orchestrator implements a transport-level proxy that only inspects the first message from new connections (to handle `fabric/registerClient`). Subsequent messages are routed based on session ID without content inspection.

**Protocol Flow**:
1. Claude-reactor client connects to reactor-fabric MCP endpoint
2. Client immediately calls `fabric/registerClient` with volume mount information
3. Orchestrator validates mounts against `allowed_mount_roots`, creates session ID, stores context
4. Client sends tool request (e.g., git tool)
5. Orchestrator identifies service, spawns container with session mounts, proxies request
6. Response proxied back to client, idle timeout timer started

### **Discovery & Fallback Strategy**

**Discovery**: Environment variable `REACTOR_FABRIC_ENDPOINT` for orchestrator location. If unset, operates in standalone mode.

**Graceful Fallback**: If `REACTOR_FABRIC_ENDPOINT` is set but orchestrator unavailable, claude-reactor prints warning ("Warning: Could not connect to Reactor-Fabric at [endpoint]. Falling back to standalone mode.") and continues normally.

### **Security Implementation**

**Mount Path Validation**: `allowed_mount_roots` must be user-configurable in `claude-mcp-suite.yaml` orchestrator section. Paths validated against explicit security boundaries.

**Docker Socket Security**: All Docker SDK calls use Go context with 30-second timeout to prevent DoS via stalled daemon operations. Code requires strict security review.

### **Testing Strategy (TDD)**

**Phase-Aligned Testing**:
- **Phase 0**: 100% existing test pass rate after refactoring (gate condition)
- **Phase 1**: Unit tests for YAML parsing and container lifecycle (mocked Docker SDK)
- **Phase 2**: Integration tests for `fabric/registerClient` and MCP proxy logic
- **Phase 3**: End-to-end tests including concurrent client scenarios

## **Assumptions**

* It is assumed that the user is running on a system with Docker Desktop and has sufficient permissions to interact with the Docker daemon.  
* It is assumed that the performance of dynamic container startup on a modern development machine will be acceptable for an interactive workflow.