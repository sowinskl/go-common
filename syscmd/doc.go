// Package syscmd provides a fluent interface for executing system commands
// with built-in support for timeouts, retries, and context cancellation.
//
// The package is designed to make system command execution more reliable
// and easier to configure through a fluent API pattern.
//
// Basic usage:
//
//	err := syscmd.Run("echo", "hello")
//	output, err := syscmd.Output("ls", "-la")
//
// Advanced usage with configuration:
//
//	cmd := syscmd.New().
//		Timeout(10 * time.Second).
//		Retry(3, 2*time.Second).
//		Quiet()
//
//	err := cmd.ExecuteQuiet("systemctl", "restart", "nginx")
//
// Predefined configurations:
//
//	err := syscmd.Quick().ExecuteQuiet("fast-command")     // 5s timeout
//	err := syscmd.Resilient().ExecuteQuiet("slow-command") // 30s timeout, 3 retries
package syscmd
