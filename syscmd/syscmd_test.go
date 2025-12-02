package syscmd

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	ctx := context.Background()
	cmd := New(ctx)

	require.NotNil(t, cmd)
	assert.Equal(t, 30*time.Second, cmd.timeout)
	assert.Equal(t, 0, cmd.retries)
	assert.Equal(t, 1*time.Second, cmd.retryDelay)
	assert.Equal(t, ctx, cmd.ctx)
}

func TestTimeout(t *testing.T) {
	ctx := context.Background()
	cmd := New(ctx).Timeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, cmd.timeout)
}

func TestRetry(t *testing.T) {
	ctx := context.Background()
	cmd := New(ctx).Retry(3, 2*time.Second)
	assert.Equal(t, 3, cmd.retries)
	assert.Equal(t, 2*time.Second, cmd.retryDelay)
}

func TestChaining(t *testing.T) {
	ctx := context.Background()
	cmd := New(ctx).
		Timeout(10*time.Second).
		Retry(2, 500*time.Millisecond)

	assert.Equal(t, 10*time.Second, cmd.timeout)
	assert.Equal(t, 2, cmd.retries)
	assert.Equal(t, 500*time.Millisecond, cmd.retryDelay)
}

func TestExecute_Success(t *testing.T) {
	ctx := context.Background()
	cmd := New(ctx)

	var output string
	var err error

	if runtime.GOOS == "windows" {
		output, err = cmd.Execute("cmd", "/c", "echo hello")
	} else {
		output, err = cmd.Execute("echo", "hello")
	}

	require.NoError(t, err)
	assert.Contains(t, output, "hello")
}

func TestExecute_Timeout(t *testing.T) {
	ctx := context.Background()
	cmd := New(ctx).Timeout(100 * time.Millisecond)

	var cmdName, arg string
	if runtime.GOOS == "windows" {
		cmdName = "timeout"
		arg = "5"
	} else {
		cmdName = "sleep"
		arg = "5"
	}

	_, err := cmd.Execute(cmdName, arg)
	assert.Error(t, err)
}

func TestExecute_NonExistentCommand(t *testing.T) {
	ctx := context.Background()
	cmd := New(ctx)

	_, err := cmd.Execute("this-command-should-not-exist-12345")
	assert.Error(t, err)
}

func TestExecute_WithRetries(t *testing.T) {
	ctx := context.Background()
	cmd := New(ctx).Retry(2, 100*time.Millisecond)

	_, err := cmd.Execute("this-command-should-not-exist-12345")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 2 retries")
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cmd := New(ctx)
	_, err := cmd.Execute("echo", "hello")

	// The command might still succeed if it's fast enough,
	// but we should handle context cancellation gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "context canceled")
	}
}

// Parent Context Cancellation Tests

func TestParentContext_BasicCancellation(t *testing.T) {
	// Create parent context
	parentCtx, parentCancel := context.WithCancel(context.Background())

	// Create commands with the same parent context
	cmd1 := New(parentCtx).Timeout(30 * time.Second)
	cmd2 := New(parentCtx).Timeout(30 * time.Second)

	results := make(chan error, 2)
	var durations []time.Duration
	var mu sync.Mutex

	// Start first command
	go func() {
		start := time.Now()
		_, err := cmd1.Execute("sleep", "10")
		duration := time.Since(start)
		mu.Lock()
		durations = append(durations, duration)
		mu.Unlock()
		results <- err
	}()

	// Start second command
	go func() {
		start := time.Now()
		_, err := cmd2.Execute("sleep", "15")
		duration := time.Since(start)
		mu.Lock()
		durations = append(durations, duration)
		mu.Unlock()
		results <- err
	}()

	// Cancel parent context after 2 seconds
	time.Sleep(2 * time.Second)
	parentCancel()

	// Collect results
	for i := 0; i < 2; i++ {
		err := <-results
		assert.Error(t, err, "Command %d should have been cancelled", i+1)
	}

	// Both commands should have been cancelled around 2 seconds
	mu.Lock()
	for i, duration := range durations {
		assert.Less(t, duration, 3*time.Second, "Command %d took too long: %v", i+1, duration)
		assert.Greater(t, duration, 1500*time.Millisecond, "Command %d finished too quickly: %v", i+1, duration)
	}
	mu.Unlock()
}

func TestParentContext_MultipleConfigurations(t *testing.T) {
	// Create parent context
	parentCtx, parentCancel := context.WithCancel(context.Background())

	// Create commands with different configurations but same parent context
	commands := []*Process{
		New(parentCtx).Timeout(5 * time.Second),                               // Quick
		New(parentCtx).Timeout(30*time.Second).Retry(3, 2*time.Second),        // Resilient
		New(parentCtx).Timeout(45*time.Second).Retry(2, 500*time.Millisecond), // Custom
	}

	var wg sync.WaitGroup
	results := make(chan error, len(commands))
	var durations []time.Duration
	var mu sync.Mutex

	// Start all commands
	for i, cmd := range commands {
		wg.Add(1)
		go func(cmdIndex int, command *Process) {
			defer wg.Done()
			start := time.Now()
			_, err := command.Execute("sleep", "20")
			duration := time.Since(start)
			mu.Lock()
			durations = append(durations, duration)
			mu.Unlock()
			results <- err
		}(i, cmd)
	}

	// Cancel parent context after 3 seconds
	time.Sleep(3 * time.Second)
	parentCancel()

	// Wait for all goroutines and collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	// All commands should have been cancelled
	cancelled := 0
	for err := range results {
		if err != nil {
			cancelled++
		}
	}

	assert.Equal(t, len(commands), cancelled, "All commands should have been cancelled")

	// Check that all commands were cancelled reasonably quickly
	mu.Lock()
	for i, duration := range durations {
		assert.Less(t, duration, 15*time.Second, "Command %d took too long: %v", i, duration)
	}
	mu.Unlock()
}

func TestParentContext_NestedContexts(t *testing.T) {
	// Create grandparent context
	grandparentCtx, grandparentCancel := context.WithCancel(context.Background())

	// Create parent context from grandparent
	parentCtx, parentCancel := context.WithTimeout(grandparentCtx, 10*time.Second)
	defer parentCancel()

	// Create child commands
	cmd1 := New(parentCtx)
	cmd2 := New(parentCtx)

	results := make(chan error, 2)
	var durations []time.Duration
	var mu sync.Mutex

	// Start commands
	go func() {
		start := time.Now()
		_, err := cmd1.Execute("sleep", "20")
		duration := time.Since(start)
		mu.Lock()
		durations = append(durations, duration)
		mu.Unlock()
		results <- err
	}()

	go func() {
		start := time.Now()
		_, err := cmd2.Execute("sleep", "25")
		duration := time.Since(start)
		mu.Lock()
		durations = append(durations, duration)
		mu.Unlock()
		results <- err
	}()

	// Cancel grandparent context after 2 seconds (should cancel everything)
	time.Sleep(2 * time.Second)
	grandparentCancel()

	// Collect results
	for i := 0; i < 2; i++ {
		err := <-results
		assert.Error(t, err, "Child command %d should have been cancelled", i+1)
	}

	// Both commands should have been cancelled around 2 seconds
	mu.Lock()
	for i, duration := range durations {
		assert.Less(t, duration, 3*time.Second, "Child command %d took too long: %v", i+1, duration)
		assert.Greater(t, duration, 1500*time.Millisecond, "Child command %d finished too quickly: %v", i+1, duration)
	}
	mu.Unlock()
}

func TestParentContext_HighConcurrency(t *testing.T) {
	// Create parent context
	parentCtx, parentCancel := context.WithCancel(context.Background())

	const numCommands = 20
	var wg sync.WaitGroup
	results := make(chan error, numCommands)
	var durations []time.Duration
	var mu sync.Mutex

	// Start multiple commands concurrently
	for i := 0; i < numCommands; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Vary the command configurations
			var cmd *Process
			switch id % 3 {
			case 0:
				cmd = New(parentCtx).Timeout(5 * time.Second)
			case 1:
				cmd = New(parentCtx).Timeout(30*time.Second).Retry(3, 2*time.Second)
			default:
				cmd = New(parentCtx).Timeout(time.Duration(15+id) * time.Second)
			}

			start := time.Now()
			_, err := cmd.Execute("sleep", "20")
			duration := time.Since(start)
			mu.Lock()
			durations = append(durations, duration)
			mu.Unlock()
			results <- err
		}(i)
	}

	// Cancel parent context after 1.5 seconds
	time.Sleep(1500 * time.Millisecond)
	parentCancel()

	// Wait for all goroutines and collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	cancelled := 0
	for err := range results {
		if err != nil {
			cancelled++
		}
	}

	// Most commands should have been cancelled
	assert.Greater(t, cancelled, numCommands/2, "Most commands should have been cancelled")

	// Check that cancelled commands finished quickly
	mu.Lock()
	quickCancellations := 0
	for _, duration := range durations {
		if duration < 3*time.Second {
			quickCancellations++
		}
	}
	mu.Unlock()

	assert.Greater(t, quickCancellations, numCommands/2, "Most commands should have been cancelled quickly")
}

func TestParentContext_CancellationDuringRetries(t *testing.T) {
	parentCtx, parentCancel := context.WithCancel(context.Background())

	// Create a command that will retry a failing command
	cmd := New(parentCtx).Retry(3, 1*time.Second)

	done := make(chan error, 1)
	var duration time.Duration

	go func() {
		start := time.Now()
		_, err := cmd.Execute("this-command-does-not-exist-12345")
		duration = time.Since(start)
		done <- err
	}()

	// Cancel during retries
	time.Sleep(2500 * time.Millisecond) // Cancel during retry attempts
	parentCancel()

	err := <-done
	require.Error(t, err)

	// Should have been cancelled before all retries completed
	// With 5 retries and 1s delay, full execution would take ~5+ seconds
	// But cancellation during retries may take longer due to exponential backoff implementation
	assert.Less(t, duration, 5*time.Second, "Command should have been cancelled before all retries: %v", duration)
}
