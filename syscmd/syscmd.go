package syscmd

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Executor defines the behavior required to run system commands.
type Command interface {
	Execute(name string, args ...string) (string, error)
}

// Command provides a fluent interface for executing system commands with timeout and retry
type Process struct {
	ctx        context.Context
	timeout    time.Duration
	retries    int
	retryDelay time.Duration
}

// Ensure Command implements Executor at compile time
var _ Command = (*Process)(nil)

// New creates a new system command instance with context and default values
func New(ctx context.Context) *Process {
	return &Process{
		ctx:        ctx,
		timeout:    30 * time.Second, // default timeout
		retries:    0,                // no retries by default
		retryDelay: 1 * time.Second,  // default retry delay
	}
}

// Timeout sets the timeout for command execution
func (c *Process) Timeout(timeout time.Duration) *Process {
	c.timeout = timeout
	return c
}

// Retry sets the number of retries and delay between retries
func (c *Process) Retry(retries int, delay time.Duration) *Process {
	c.retries = retries
	c.retryDelay = delay
	return c
}

// Execute runs the command with the configured timeout and retry settings
func (c *Process) Execute(name string, args ...string) (string, error) {
	var finalOutput string

	operation := func() error {
		var execCtx context.Context
		var cancel context.CancelFunc

		if c.timeout > 0 {
			execCtx, cancel = context.WithTimeout(c.ctx, c.timeout)
		} else {
			execCtx, cancel = context.WithCancel(c.ctx)
		}
		defer cancel()

		cmd := exec.CommandContext(execCtx, name, args...)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("command failed: %w, output: %s", err, string(output))
		}

		finalOutput = string(output)
		return nil
	}

	if c.retries > 0 {
		b := backoff.NewExponentialBackOff()
		b.InitialInterval = c.retryDelay
		b.MaxElapsedTime = time.Duration(c.retries+1) * c.retryDelay * 2
		contextBackoff := backoff.WithContext(b, c.ctx)
		err := backoff.Retry(operation, backoff.WithMaxRetries(contextBackoff, uint64(c.retries)))
		if err != nil {
			return "", fmt.Errorf("command failed after %d retries: %w", c.retries, err)
		}
	} else {
		if err := operation(); err != nil {
			return "", err
		}
	}

	return finalOutput, nil
}

// Run is a convenience function for simple command execution
func Run(ctx context.Context, name string, args ...string) (string, error) {
	return New(ctx).Execute(name, args...)
}
