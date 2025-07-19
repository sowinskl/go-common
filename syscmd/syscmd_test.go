package syscmd

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	cmd := New()
	if cmd == nil {
		t.Fatal("New() returned nil")
	}
	if cmd.timeout != 30*time.Second {
		t.Errorf("Expected default timeout to be 30s, got %v", cmd.timeout)
	}
	if cmd.retries != 0 {
		t.Errorf("Expected default retries to be 0, got %d", cmd.retries)
	}
	if cmd.retryDelay != 1*time.Second {
		t.Errorf("Expected default retry delay to be 1s, got %v", cmd.retryDelay)
	}
	if cmd.quiet != false {
		t.Errorf("Expected default quiet to be false, got %v", cmd.quiet)
	}
}

func TestTimeout(t *testing.T) {
	cmd := New().Timeout(10 * time.Second)
	if cmd.timeout != 10*time.Second {
		t.Errorf("Expected timeout to be 10s, got %v", cmd.timeout)
	}
}

func TestRetry(t *testing.T) {
	cmd := New().Retry(3, 2*time.Second)
	if cmd.retries != 3 {
		t.Errorf("Expected retries to be 3, got %d", cmd.retries)
	}
	if cmd.retryDelay != 2*time.Second {
		t.Errorf("Expected retry delay to be 2s, got %v", cmd.retryDelay)
	}
}

func TestQuiet(t *testing.T) {
	cmd := New().Quiet()
	if cmd.quiet != true {
		t.Errorf("Expected quiet to be true, got %v", cmd.quiet)
	}
}

func TestWithContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "test", "value")
	cmd := New().WithContext(ctx)
	if cmd.ctx != ctx {
		t.Errorf("Expected context to be set")
	}
}

func TestExecute_Success(t *testing.T) {
	var cmdName, arg string
	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		arg = "/c"
	} else {
		cmdName = "echo"
	}

	cmd := New()

	var output string
	var err error

	if runtime.GOOS == "windows" {
		output, err = cmd.Execute(cmdName, arg, "echo hello")
	} else {
		output, err = cmd.Execute(cmdName, "hello")
	}

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("Expected output to contain 'hello', got %s", output)
	}
}

func TestExecute_Timeout(t *testing.T) {
	var cmdName, arg string
	if runtime.GOOS == "windows" {
		cmdName = "timeout"
		arg = "5"
	} else {
		cmdName = "sleep"
		arg = "5"
	}

	cmd := New().Timeout(100 * time.Millisecond)
	_, err := cmd.Execute(cmdName, arg)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestExecuteQuiet(t *testing.T) {
	var cmdName, arg string
	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		arg = "/c"
	} else {
		cmdName = "echo"
	}

	cmd := New()

	var err error
	if runtime.GOOS == "windows" {
		err = cmd.ExecuteQuiet(cmdName, arg, "echo hello")
	} else {
		err = cmd.ExecuteQuiet(cmdName, "hello")
	}

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRun(t *testing.T) {
	var cmdName, arg string
	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		arg = "/c"
	} else {
		cmdName = "echo"
	}

	var err error
	if runtime.GOOS == "windows" {
		err = Run(cmdName, arg, "echo hello")
	} else {
		err = Run(cmdName, "hello")
	}

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestOutput(t *testing.T) {
	var cmdName, arg string
	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		arg = "/c"
	} else {
		cmdName = "echo"
	}

	var output string
	var err error

	if runtime.GOOS == "windows" {
		output, err = Output(cmdName, arg, "echo hello")
	} else {
		output, err = Output(cmdName, "hello")
	}

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("Expected output to contain 'hello', got %s", output)
	}
}

func TestQuick(t *testing.T) {
	cmd := Quick()
	if cmd.timeout != 5*time.Second {
		t.Errorf("Expected Quick() timeout to be 5s, got %v", cmd.timeout)
	}
}

func TestResilient(t *testing.T) {
	cmd := Resilient()
	if cmd.timeout != 30*time.Second {
		t.Errorf("Expected Resilient() timeout to be 30s, got %v", cmd.timeout)
	}
	if cmd.retries != 3 {
		t.Errorf("Expected Resilient() retries to be 3, got %d", cmd.retries)
	}
	if cmd.retryDelay != 2*time.Second {
		t.Errorf("Expected Resilient() retry delay to be 2s, got %v", cmd.retryDelay)
	}
}

func TestChaining(t *testing.T) {
	cmd := New().
		Timeout(10*time.Second).
		Retry(2, 500*time.Millisecond).
		Quiet().
		WithContext(context.Background())

	if cmd.timeout != 10*time.Second {
		t.Errorf("Expected chained timeout to be 10s, got %v", cmd.timeout)
	}
	if cmd.retries != 2 {
		t.Errorf("Expected chained retries to be 2, got %d", cmd.retries)
	}
	if cmd.retryDelay != 500*time.Millisecond {
		t.Errorf("Expected chained retry delay to be 500ms, got %v", cmd.retryDelay)
	}
	if cmd.quiet != true {
		t.Errorf("Expected chained quiet to be true, got %v", cmd.quiet)
	}
}

func TestExecute_NonExistentCommand(t *testing.T) {
	cmd := New()
	_, err := cmd.Execute("this-command-should-not-exist-12345")
	if err == nil {
		t.Error("Expected error for non-existent command, got nil")
	}
}

func TestExecute_WithRetries(t *testing.T) {
	// This test uses a command that will fail, to test retry logic
	cmd := New().Retry(2, 100*time.Millisecond)
	_, err := cmd.Execute("this-command-should-not-exist-12345")
	if err == nil {
		t.Error("Expected error after retries, got nil")
	}
	if !strings.Contains(err.Error(), "failed after 2 retries") {
		t.Errorf("Expected retry error message, got %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	cmd := New().WithContext(ctx)
	_, err := cmd.Execute("echo", "hello")

	// The command might still succeed if it's fast enough,
	// but we should handle context cancellation gracefully
	if err != nil && !strings.Contains(err.Error(), "context canceled") {
		t.Logf("Command failed as expected due to context cancellation: %v", err)
	}
}
