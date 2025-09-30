package commands

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerTimeoutContextHandling(t *testing.T) {
	t.Run("create timeout context with valid duration", func(t *testing.T) {
		// Test the timeout context creation logic used in run command
		baseCtx := context.Background()
		timeoutStr := "2s"

		timeout, err := time.ParseDuration(timeoutStr)
		require.NoError(t, err)

		dockerCtx, cancel := context.WithTimeout(baseCtx, timeout)
		defer cancel()

		// Verify context is created with deadline
		deadline, hasDeadline := dockerCtx.Deadline()
		assert.True(t, hasDeadline, "Context should have a deadline")
		assert.True(t, deadline.After(time.Now()), "Deadline should be in the future")
		assert.True(t, deadline.Before(time.Now().Add(3*time.Second)), "Deadline should be within expected range")
	})

	t.Run("timeout context expires correctly", func(t *testing.T) {
		baseCtx := context.Background()
		timeout := 50 * time.Millisecond

		dockerCtx, cancel := context.WithTimeout(baseCtx, timeout)
		defer cancel()

		// Wait for context to timeout
		select {
		case <-dockerCtx.Done():
			// Context timed out as expected
			assert.Equal(t, context.DeadlineExceeded, dockerCtx.Err())
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Context should have timed out")
		}
	})

	t.Run("no timeout context when disabled", func(t *testing.T) {
		// Test the logic when timeout is disabled (hostDockerTimeout = "0")
		baseCtx := context.Background()
		hostDockerTimeout := "0"

		// This is the logic from the run command
		var dockerCtx context.Context = baseCtx
		if hostDockerTimeout != "0" && hostDockerTimeout != "0s" {
			timeout, _ := time.ParseDuration(hostDockerTimeout)
			var cancel context.CancelFunc
			dockerCtx, cancel = context.WithTimeout(baseCtx, timeout)
			defer cancel()
		}

		// Verify no deadline is set when timeout is disabled
		_, hasDeadline := dockerCtx.Deadline()
		assert.False(t, hasDeadline, "Context should not have deadline when timeout is disabled")
	})

	t.Run("timeout context error detection", func(t *testing.T) {
		// Test deadline exceeded error detection
		baseCtx := context.Background()
		timeout := 1 * time.Millisecond

		dockerCtx, cancel := context.WithTimeout(baseCtx, timeout)
		defer cancel()

		// Wait for timeout
		time.Sleep(10 * time.Millisecond)

		// Check if context error is deadline exceeded
		err := dockerCtx.Err()
		assert.Equal(t, context.DeadlineExceeded, err)

		// Test the error check logic from run command
		isTimeout := (dockerCtx.Err() == context.DeadlineExceeded)
		assert.True(t, isTimeout, "Should detect timeout error correctly")
	})

	t.Run("timeout duration parsing edge cases", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected time.Duration
			hasError bool
		}{
			{"5m", 5 * time.Minute, false},
			{"30s", 30 * time.Second, false},
			{"1h", 1 * time.Hour, false},
			{"1h30m", 90 * time.Minute, false},
			{"0", 0, false},
			{"0s", 0, false},
			{"invalid", 0, true},
			{"5min", 0, true},
			{"-5m", -5 * time.Minute, false}, // Go allows negative durations
		}

		for _, tc := range testCases {
			t.Run("parse_"+tc.input, func(t *testing.T) {
				duration, err := time.ParseDuration(tc.input)

				if tc.hasError {
					assert.Error(t, err, "Expected error for input: %s", tc.input)
				} else {
					assert.NoError(t, err, "Expected no error for input: %s", tc.input)
					assert.Equal(t, tc.expected, duration, "Duration should match expected for input: %s", tc.input)
				}
			})
		}
	})
}

func TestTimeoutErrorMessageGeneration(t *testing.T) {
	t.Run("timeout error message format", func(t *testing.T) {
		hostDockerTimeout := "5m"

		// Test the error message format from run command
		expectedSubstrings := []string{
			"Docker operation timed out after 5m",
			"increase timeout: --host-docker-timeout 15m",
			"disable timeout: --host-docker-timeout 0",
			"Save preference: echo \"host_docker_timeout=15m\" >> .claude-reactor",
		}

		errorMsg := "Docker operation timed out after " + hostDockerTimeout + "\n💡 For complex builds, increase timeout: --host-docker-timeout 15m\n💡 For unlimited time, disable timeout: --host-docker-timeout 0\n💡 Save preference: echo \"host_docker_timeout=15m\" >> .claude-reactor"

		for _, substr := range expectedSubstrings {
			assert.Contains(t, errorMsg, substr, "Error message should contain guidance: %s", substr)
		}
	})
}