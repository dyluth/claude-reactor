# Test Coverage Guidelines

## Coverage Requirements

This project enforces a minimum test coverage threshold of **35%** for all Go code.

## How It Works

- All Pull Requests and pushes to main trigger automated coverage checks
- The GitHub Actions workflow runs `go test -coverprofile=coverage.out ./...`
- If total coverage falls below 35%, the build will fail
- Coverage reports are automatically uploaded to Codecov for detailed analysis

## Current Coverage Status

As of the last measurement, the project achieves 35.0% test coverage across:

- `internal/reactor`: 92.9% (excellent coverage)
- `internal/reactor/config`: 91.4% (excellent coverage) 
- `internal/reactor/mount`: 94.1% (excellent coverage)
- `internal/reactor/auth`: 79.2% (good coverage)
- `internal/reactor/architecture`: 67.6% (good coverage)
- `internal/reactor/logging`: 63.6% (good coverage)
- `internal/reactor/docker`: 41.8% (improving)
- `internal/reactor/docker/validation`: 26.5% (improving)
- `cmd/claude-reactor`: 20.0% (needs improvement)
- `cmd/claude-reactor/commands`: 12.3% (needs improvement)

## Contributing Guidelines

When adding new code:

1. **Always include tests** for new functionality
2. **Aim for higher coverage** where practical (target 60%+ for new packages)
3. **Focus on critical path testing** - prioritize testing the most important code paths
4. **Use table-driven tests** for comprehensive scenario coverage
5. **Mock external dependencies** using testify/mock or similar patterns

## Running Coverage Locally

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage by function
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

## Coverage Improvement Strategies

1. **Start with utility functions** - easier to test, quick wins
2. **Add integration tests** for complex workflows  
3. **Test error conditions** - many packages have low coverage on error paths
4. **Use dependency injection** to make code more testable
5. **Separate business logic** from infrastructure concerns

## Exceptions

The coverage requirement may be temporarily waived for:
- Experimental features marked with build tags
- Code that requires significant infrastructure setup
- Generated code (protobuf, etc.)

Contact maintainers if you need an exception for a specific PR.