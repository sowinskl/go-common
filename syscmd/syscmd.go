package syscmd

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Command provides a fluent interface for executing system commands with timeout and retry
//
// Example usage:
//
//	cmd := syscmd.New().Retry(3, 3*time.Second).Timeout(10*time.Second).Quiet().WithContext(ctx)
//	err := cmd.Execute("systemctl", "restart", "nginx")
type Command struct {
	timeout    time.Duration
	retries    int
	retryDelay time.Duration
	quiet      bool
	ctx        context.Context
}

// New creates a new system command instance with default values
func New() *Command {
	return &Command{
		timeout:    30 * time.Second, // default timeout
		retries:    0,                // no retries by default
		retryDelay: 1 * time.Second,  // default retry delay
		quiet:      false,
		ctx:        context.Background(),
	}
}

// Timeout sets the timeout for command execution
func (c *Command) Timeout(timeout time.Duration) *Command {
	c.timeout = timeout
	return c
}

// Retry sets the number of retries and delay between retries
func (c *Command) Retry(retries int, delay time.Duration) *Command {
	c.retries = retries
	c.retryDelay = delay
	return c
}

// Quiet sets the command to run in quiet mode (no output)
func (c *Command) Quiet() *Command {
	c.quiet = true
	return c
}

// WithContext sets the context for command execution
func (c *Command) WithContext(ctx context.Context) *Command {
	c.ctx = ctx
	return c
}

// Execute runs the command with the configured timeout and retry settings
func (c *Command) Execute(name string, args ...string) (string, error) {
	var finalOutput string

	operation := func() error {
		// Create timeout context from the base context
		var ctx context.Context
		var cancel context.CancelFunc

		if c.timeout > 0 {
			ctx, cancel = context.WithTimeout(c.ctx, c.timeout)
		} else {
			ctx, cancel = context.WithCancel(c.ctx)
		}
		defer cancel()

		cmd := exec.CommandContext(ctx, name, args...)

		if c.quiet {
			return cmd.Run()
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("command failed: %w, output: %s", err, string(output))
		}

		finalOutput = string(output)
		return nil
	}

	if c.retries > 0 {
		// Use exponential backoff with jitter
		b := backoff.NewExponentialBackOff()
		b.InitialInterval = c.retryDelay
		b.MaxElapsedTime = time.Duration(c.retries+1) * c.retryDelay * 2 // reasonable max time

		err := backoff.Retry(operation, backoff.WithMaxRetries(b, uint64(c.retries)))
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

// ExecuteQuiet runs the command in quiet mode (equivalent to Quiet().Execute())
func (c *Command) ExecuteQuiet(name string, args ...string) error {
	_, err := c.Quiet().Execute(name, args...)
	return err
}

// Run is a convenience function for simple command execution without configuration
func Run(name string, args ...string) error {
	return New().ExecuteQuiet(name, args...)
}

// Output is a convenience function for getting command output without configuration
func Output(name string, args ...string) (string, error) {
	return New().Execute(name, args...)
}

// Quick creates a command with 5 second timeout for fast operations
func Quick() *Command {
	return New().Timeout(5 * time.Second)
}

// Resilient creates a command with 30 second timeout and 3 retries for reliable operations
func Resilient() *Command {
	return New().Timeout(30*time.Second).Retry(3, 2*time.Second)
}
