# Claude-Reactor Test Suite

Comprehensive testing framework for the claude-reactor Docker containerization system.

## Test Structure

```
tests/
├── test-runner.sh          # Main test orchestrator
├── demo.sh                 # Interactive feature demonstration
├── unit/
│   └── test-functions.sh   # Unit tests for script functions
├── integration/
│   └── test-variants.sh    # Integration tests for Docker containers
└── fixtures/               # Test data and temporary files
```

## Quick Start

```bash
# Run all tests
./tests/test-runner.sh

# Run only unit tests (fast)
./tests/test-runner.sh --unit

# Run integration tests without Docker builds (faster)
./tests/test-runner.sh --integration --quick

# Interactive demo
./tests/demo.sh

# Auto demo (no pauses)
./tests/demo.sh --auto

# Quick demo (no Docker builds)
./tests/demo.sh --quick
```

## Test Types

### Unit Tests (`tests/unit/test-functions.sh`)
- Configuration save/load functionality
- Auto-detection logic for different project types
- Variant validation and determination
- Configuration persistence across runs
- **Runtime**: ~5 seconds
- **Requirements**: bash, basic shell utilities

### Integration Tests (`tests/integration/test-variants.sh`)
- Docker container builds for each variant
- Container lifecycle management
- Tool availability verification
- Script option testing
- **Runtime**: 5-15 minutes (depending on variants)
- **Requirements**: Docker daemon running

### Demo Script (`tests/demo.sh`)
- Interactive showcase of all features
- Auto-detection demonstrations
- Configuration persistence examples  
- Container building workflow
- **Runtime**: 2-10 minutes (depending on mode)
- **Requirements**: Docker daemon (unless --quick mode)

## Test Options

### test-runner.sh Options
- `--unit` - Run only unit tests
- `--integration` - Run only integration tests
- `--quick` - Skip Docker builds (faster)
- `--verbose` - Enable detailed output
- `--clean` - Clean test artifacts before running

### demo.sh Options
- `--auto` - Run automatically without pauses
- `--quick` - Skip Docker builds, show functionality only

## Expected Test Results

### Unit Tests (should always pass)
- ✅ Configuration save/load
- ✅ Auto-detection logic
- ✅ Variant validation  
- ✅ Variant determination priority
- ✅ Configuration persistence

### Integration Tests (requires Docker)
- ✅ Docker availability
- ✅ Script options functionality
- ✅ Configuration file integration
- ✅ Build variant: base, go, full
- ✅ Container lifecycle for each variant
- ✅ Tools availability verification

## Troubleshooting

### Docker Issues
- Ensure Docker daemon is running: `docker info`
- Check Docker permissions: `docker ps`
- Free up disk space if builds fail

### Test Failures
- Run with `--verbose` for detailed output
- Check individual test scripts directly
- Use `--clean` to reset test environment

### Performance
- Use `--quick` to skip time-consuming Docker builds
- Run `--unit` only for fast validation
- Integration tests may take 10+ minutes for full suite

## Development

To add new tests:

1. **Unit tests**: Add functions to `tests/unit/test-functions.sh`
2. **Integration tests**: Add functions to `tests/integration/test-variants.sh`
3. **Demo features**: Extend `tests/demo.sh`

Follow the existing pattern:
```bash
test_my_feature() {
    # Test logic here
    return 0  # success
    return 1  # failure
}

run_test "My feature description" "test_my_feature"
```