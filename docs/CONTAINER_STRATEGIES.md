# Container Strategy Guide

This document explains the three container lifecycle strategies available in Reactor-Fabric Phase 3.

## Overview

Reactor-Fabric automatically manages container lifecycles based on intelligent defaults, but you can customize the behavior for optimal performance and resource usage.

## Strategy Types

### 1. `fresh_per_call` 
**When to Use**: LLM agents, security-sensitive operations
**Behavior**: Creates a new container for every tool call
**Pros**: 
- Fresh context window for LLM agents
- Maximum security isolation
- No state contamination between calls
**Cons**: 
- Higher latency (container startup time)
- Higher resource usage
**Default for**: Auto-detected LLM agent services

```yaml
python_expert:
  image: "claude-reactor:python"
  container_strategy: "fresh_per_call"
  # New container every time = fresh context window
```

### 2. `reuse_per_session`
**When to Use**: Traditional tool services, iterative development
**Behavior**: Reuses the same container within a client session
**Pros**:
- Fast response times (no startup delay)
- Efficient resource usage
- Good for stateful operations
**Cons**:
- Potential state contamination
- Memory usage can grow over time
**Default for**: Auto-detected tool services

```yaml
filesystem:
  image: "ghcr.io/modelcontextprotocol/server-filesystem:latest"
  container_strategy: "reuse_per_session"
  # Same container reused for all filesystem operations in session
```

### 3. `smart_refresh`
**When to Use**: Balanced performance/freshness requirements
**Behavior**: Intelligently refreshes containers based on configurable criteria
**Pros**:
- Balances performance and freshness
- Configurable thresholds
- Prevents resource exhaustion
**Cons**:
- More complex configuration
- Requires monitoring and tuning

```yaml
git_service:
  image: "ghcr.io/modelcontextprotocol/server-git:latest"
  container_strategy: "smart_refresh"
  max_calls_per_container: 15  # Refresh after 15 operations
  max_container_age: "25m"     # Or after 25 minutes
  memory_threshold: "300MB"    # Or when memory exceeds threshold
```

## Smart Refresh Thresholds

### `max_calls_per_container`
- **Purpose**: Limits the number of tool calls before container refresh
- **LLM Agent Default**: 5 calls (preserve context quality)
- **Tool Service Default**: 20 calls (favor performance)
- **Range**: 1-100 (recommended: 3-30)

### `max_container_age`
- **Purpose**: Limits how long a container can run before refresh
- **LLM Agent Default**: "20m" (preserve responsiveness)
- **Tool Service Default**: "30m" (favor efficiency)
- **Format**: Go duration strings ("1m", "30s", "2h")

### `memory_threshold`
- **Purpose**: Triggers refresh when container memory usage exceeds limit
- **LLM Agent Default**: "400MB" (prevent memory bloat)
- **Tool Service Default**: "300MB" (efficient resource usage)
- **Format**: Memory units ("100MB", "1GB", "512MB")
- **Note**: Memory monitoring implementation is future work

## Auto-Detection Rules

### LLM Agent Detection
Images containing any of these patterns are auto-detected as `llm_agent`:
- `claude-reactor`, `claude-`, `llm-`, `gpt-`
- `anthropic`, `openai`, `chatgpt`
- `assistant`, `agent`, `expert`

**Default Strategy**: `fresh_per_call` (fresh context window)

### Tool Service Detection
Images containing any of these patterns are auto-detected as `tool_service`:
- `server-filesystem`, `server-git`, `server-shell`
- `server-database`, `mcp-server`
- `tool-`, `util-`

**Default Strategy**: `reuse_per_session` (performance optimization)

## Configuration Examples

### High-Performance Tool Service
```yaml
database_service:
  image: "postgres-mcp-server:latest"
  container_strategy: "reuse_per_session"
  timeout: "5m"
  # Optimized for speed, reuses containers
```

### Security-Conscious LLM Agent
```yaml
security_expert:
  image: "claude-reactor:security"
  container_strategy: "fresh_per_call"
  timeout: "10m"
  # Fresh container = isolated analysis environment
```

### Balanced Hybrid Service
```yaml
code_reviewer:
  image: "claude-reactor:reviewer"
  container_strategy: "smart_refresh"
  max_calls_per_container: 8   # Review ~8 files before refresh
  max_container_age: "20m"     # Or refresh every 20 minutes
  memory_threshold: "350MB"    # Or when memory usage gets high
```

## Best Practices

1. **Use defaults when possible** - Auto-detection works well for most cases
2. **Fresh containers for LLMs** - Preserves context window quality
3. **Reuse for tools** - Faster performance for stateless operations
4. **Smart refresh for balance** - When you need both speed and freshness
5. **Monitor resource usage** - Adjust thresholds based on actual usage patterns
6. **Security-sensitive = fresh** - Always use fresh containers for security tools

## Performance Impact

| Strategy | Latency | Memory | CPU | Use Case |
|----------|---------|--------|-----|----------|
| fresh_per_call | High | Low | High | LLM agents, security |
| reuse_per_session | Low | Medium | Low | Tool services |
| smart_refresh | Medium | Medium | Medium | Balanced workloads |

## Troubleshooting

### High Latency
- Consider switching from `fresh_per_call` to `smart_refresh`
- Increase `max_calls_per_container` for smart refresh

### Memory Issues
- Decrease `memory_threshold` for smart refresh
- Switch to `fresh_per_call` for memory-intensive services

### Context Quality Issues (LLMs)
- Ensure LLM services use `fresh_per_call` or low `max_calls_per_container`
- Reduce `max_container_age` for smart refresh

### Resource Exhaustion
- Lower `max_calls_per_container` and `max_container_age`
- Switch high-usage services to `fresh_per_call`