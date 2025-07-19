package syscmd_test

import (
	"context"
	"fmt"
	"time"

	"github.com/sowinskl/go-common/syscmd"
)

func ExampleRun() {
	err := syscmd.Run("echo", "hello world")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println("Command executed successfully")
	// Output: Command executed successfully
}

func ExampleOutput() {
	output, err := syscmd.Output("echo", "hello world")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Print(output)
	// Output: hello world
}

func ExampleNew() {
	cmd := syscmd.New().
		Timeout(10*time.Second).
		Retry(3, 2*time.Second).
		WithContext(context.Background())

	output, err := cmd.Execute("echo", "configured command")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Print(output)
	// Output: configured command
}

func ExampleQuick() {
	// Quick commands have a 5-second timeout
	err := syscmd.Quick().ExecuteQuiet("echo", "fast operation")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	// Output:
}

func ExampleResilient() {
	// Resilient commands have 30-second timeout and 3 retries
	err := syscmd.Resilient().ExecuteQuiet("echo", "reliable operation")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	// Output:
}

func ExampleCommand_Timeout() {
	cmd := syscmd.New().Timeout(5 * time.Second)

	output, err := cmd.Execute("echo", "with timeout")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Print(output)
	// Output: with timeout
}

func ExampleCommand_Retry() {
	cmd := syscmd.New().Retry(3, 1*time.Second)

	// This will retry up to 3 times with 1 second delay
	_, err := cmd.Execute("echo", "with retries")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println("Command with retry executed")
	// Output: Command with retry executed
}

func ExampleCommand_Quiet() {
	cmd := syscmd.New().Quiet()

	// This runs silently without capturing output
	err := cmd.ExecuteQuiet("echo", "silent operation")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	// Output:
}
