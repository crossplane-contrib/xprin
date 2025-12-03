/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert" //nolint:depguard // testify is widely used for testing
)

func TestCaptureStderr(t *testing.T) {
	t.Run("basic output capture", func(t *testing.T) {
		// Test capturing stderr
		output := CaptureStderr(func() {
			fmt.Fprintf(os.Stderr, "test error output")
		})

		assert.Equal(t, "test error output", output)
	})

	t.Run("empty output", func(t *testing.T) {
		// Test capturing when nothing is written
		output := CaptureStderr(func() {
			// No output
		})

		assert.Empty(t, output)
	})

	t.Run("multiple writes", func(t *testing.T) {
		// Test capturing with multiple write operations
		output := CaptureStderr(func() {
			fmt.Fprintf(os.Stderr, "first ")
			fmt.Fprintf(os.Stderr, "second ")
			fmt.Fprintf(os.Stderr, "third")
		})

		assert.Equal(t, "first second third", output)
	})
}

func TestCaptureStdout(t *testing.T) {
	t.Run("basic output capture", func(t *testing.T) {
		// Test capturing stdout
		output := CaptureStdout(func() {
			fmt.Fprintf(os.Stdout, "test standard output")
		})

		assert.Equal(t, "test standard output", output)
	})

	t.Run("empty output", func(t *testing.T) {
		// Test capturing when nothing is written
		output := CaptureStdout(func() {
			// No output
		})

		assert.Empty(t, output)
	})

	t.Run("with formatting", func(t *testing.T) {
		// Test capturing with formatting
		output := CaptureStdout(func() {
			fmt.Printf("Number: %d, String: %s, Float: %.2f", 42, "test", 3.14159) //nolint:forbidigo // testing fmt.Printf capture
		})

		assert.Equal(t, "Number: 42, String: test, Float: 3.14", output)
	})
}

func TestCaptureWithMultipleLines(t *testing.T) {
	t.Run("stderr multiple lines", func(t *testing.T) {
		// Test capturing multiple lines
		output := CaptureStderr(func() {
			fmt.Fprintf(os.Stderr, "line 1\nline 2\nline 3")
		})

		lines := strings.Split(output, "\n")
		assert.Len(t, lines, 3)
		assert.Equal(t, "line 1", lines[0])
		assert.Equal(t, "line 2", lines[1])
		assert.Equal(t, "line 3", lines[2])
	})

	t.Run("stdout multiple lines", func(t *testing.T) {
		// Test capturing multiple lines
		output := CaptureStdout(func() {
			fmt.Fprintf(os.Stdout, "line 1\nline 2\nline 3")
		})

		lines := strings.Split(output, "\n")
		assert.Len(t, lines, 3)
		assert.Equal(t, "line 1", lines[0])
		assert.Equal(t, "line 2", lines[1])
		assert.Equal(t, "line 3", lines[2])
	})
}

func TestCaptureOutput(t *testing.T) {
	t.Run("basic output capture", func(t *testing.T) {
		// Test capturing both stdout and stderr
		output := CaptureOutput(func() {
			fmt.Fprintf(os.Stdout, "standard output")
			fmt.Fprintf(os.Stderr, "error output")
		})

		assert.Equal(t, "standard output", output.Stdout)
		assert.Equal(t, "error output", output.Stderr)
	})

	t.Run("empty output", func(t *testing.T) {
		// Test capturing when nothing is written
		output := CaptureOutput(func() {
			// No output
		})

		assert.Empty(t, output.Stdout)
		assert.Empty(t, output.Stderr)
	})

	t.Run("stdout only", func(t *testing.T) {
		// Test with only stdout output
		output := CaptureOutput(func() {
			fmt.Fprintf(os.Stdout, "only stdout")
		})

		assert.Equal(t, "only stdout", output.Stdout)
		assert.Empty(t, output.Stderr)
	})

	t.Run("stderr only", func(t *testing.T) {
		// Test with only stderr output
		output := CaptureOutput(func() {
			fmt.Fprintf(os.Stderr, "only stderr")
		})

		assert.Empty(t, output.Stdout)
		assert.Equal(t, "only stderr", output.Stderr)
	})

	t.Run("multiple lines", func(t *testing.T) {
		// Test with multiple lines in both streams
		output := CaptureOutput(func() {
			fmt.Fprintf(os.Stdout, "line 1\nline 2\nline 3")
			fmt.Fprintf(os.Stderr, "error 1\nerror 2\nerror 3")
		})

		stdoutLines := strings.Split(output.Stdout, "\n")
		assert.Len(t, stdoutLines, 3)
		assert.Equal(t, "line 1", stdoutLines[0])
		assert.Equal(t, "line 2", stdoutLines[1])
		assert.Equal(t, "line 3", stdoutLines[2])

		stderrLines := strings.Split(output.Stderr, "\n")
		assert.Len(t, stderrLines, 3)
		assert.Equal(t, "error 1", stderrLines[0])
		assert.Equal(t, "error 2", stderrLines[1])
		assert.Equal(t, "error 3", stderrLines[2])
	})

	t.Run("interleaved writes", func(t *testing.T) {
		// Test with interleaved writes to stdout and stderr
		output := CaptureOutput(func() {
			fmt.Fprintf(os.Stdout, "A")
			fmt.Fprintf(os.Stderr, "1")
			fmt.Fprintf(os.Stdout, "B")
			fmt.Fprintf(os.Stderr, "2")
			fmt.Fprintf(os.Stdout, "C")
			fmt.Fprintf(os.Stderr, "3")
		})

		assert.Equal(t, "ABC", output.Stdout)
		assert.Equal(t, "123", output.Stderr)
	})
}

func TestCaptureOutputPanicHandling(t *testing.T) {
	// Save the original stdout and stderr
	origStdout := os.Stdout
	origStderr := os.Stderr

	// Test that the function doesn't allow panics to propagate
	panicked := false

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()

		CaptureOutput(func() {
			panic("test panic")
		})
	}()

	// Verify that no panic escaped the CaptureOutput function
	assert.False(t, panicked, "CaptureOutput should recover from panics")

	// Verify that stdout and stderr were properly restored
	assert.Equal(t, origStdout, os.Stdout, "Stdout should be restored after panic")
	assert.Equal(t, origStderr, os.Stderr, "Stderr should be restored after panic")

	// Test that we can still use stdout and stderr after recovery from a panic
	output := CaptureOutput(func() {
		fmt.Fprint(os.Stdout, "stdout works")
		fmt.Fprint(os.Stderr, "stderr works")
	})

	assert.Equal(t, "stdout works", output.Stdout)
	assert.Equal(t, "stderr works", output.Stderr)
}

func TestSimultaneousCapture(t *testing.T) {
	// Test capturing both stdout and stderr simultaneously
	stdoutOutput := CaptureStdout(func() {
		stderrOutput := CaptureStderr(func() {
			fmt.Fprintf(os.Stdout, "to stdout")
			fmt.Fprintf(os.Stderr, "to stderr")
		})

		assert.Equal(t, "to stderr", stderrOutput)
		fmt.Fprintf(os.Stdout, " and more")
	})

	assert.Equal(t, "to stdout and more", stdoutOutput)
}

func TestNestedCaptures(t *testing.T) {
	// Test nested captures to ensure they work correctly
	outerOutput := CaptureStderr(func() {
		fmt.Fprintf(os.Stderr, "outer start, ")

		innerOutput := CaptureStderr(func() {
			fmt.Fprintf(os.Stderr, "inner content")
		})

		assert.Equal(t, "inner content", innerOutput)
		fmt.Fprintf(os.Stderr, "outer end")
	})

	assert.Equal(t, "outer start, outer end", outerOutput)
}

func TestCaptureOriginalRestored(t *testing.T) {
	// Verify that the original stdout/stderr are restored after capturing
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	_ = CaptureStdout(func() {
		_ = CaptureStderr(func() {
			// Nested captures
		})
	})

	// Verify that the original stdout and stderr are restored
	assert.Same(t, originalStdout, os.Stdout)
	assert.Same(t, originalStderr, os.Stderr)
}

func TestCaptureErrorHandling(t *testing.T) {
	t.Run("stderr restore guarantees", func(t *testing.T) {
		// Store the original stderr for comparison
		originalStderr := os.Stderr

		// Call CaptureStderr with an empty function
		output := CaptureStderr(func() {})

		// Output should be empty, and os.Stderr should be restored
		assert.Empty(t, output)
		assert.Same(t, originalStderr, os.Stderr, "os.Stderr should be properly restored")
	})

	t.Run("stdout restore guarantees", func(t *testing.T) {
		// Store the original stdout for comparison
		originalStdout := os.Stdout

		// Call CaptureStdout with an empty function
		output := CaptureStdout(func() {})

		// Output should be empty, and os.Stdout should be restored
		assert.Empty(t, output)
		assert.Same(t, originalStdout, os.Stdout, "os.Stdout should be properly restored")
	})
}

func TestCaptureConcurrentWrites(t *testing.T) {
	t.Run("concurrent stdout writes", func(t *testing.T) {
		output := CaptureStdout(func() {
			// Simulate concurrent writes with multiple goroutines
			done := make(chan bool)

			for i := range 10 {
				go func(idx int) {
					fmt.Fprintf(os.Stdout, "concurrent-%d ", idx)

					done <- true
				}(i)
			}
			// Wait for all goroutines to finish
			for range 10 {
				<-done
			}
		})

		// Check that all writes were captured (order may vary)
		for i := range 10 {
			assert.Contains(t, output, fmt.Sprintf("concurrent-%d ", i))
		}
	})
}

func TestCaptureDifferentFormatTypes(t *testing.T) {
	t.Run("various formatting verbs", func(t *testing.T) {
		output := CaptureStderr(func() {
			fmt.Fprintf(os.Stderr, "Integer: %d", 42)
			fmt.Fprintf(os.Stderr, " String: %s", "test")
			fmt.Fprintf(os.Stderr, " Bool: %t", true)
			fmt.Fprintf(os.Stderr, " Float: %.2f", 3.14159)
		})

		assert.Equal(t, "Integer: 42 String: test Bool: true Float: 3.14", output)
	})
	t.Run("special characters", func(t *testing.T) {
		special := "Tab:\t Newline:\n Quote:\" Backslash:\\"
		output := CaptureStdout(func() {
			fmt.Fprint(os.Stdout, special)
		})

		assert.Equal(t, special, output)
	})
}

func TestCaptureWithInterfaces(t *testing.T) {
	// Test that capturing works with Writer interfaces, not just os.Stderr/os.Stdout directly
	output := CaptureStderr(func() {
		// Create a variable of type io.Writer that points to os.Stderr
		var w io.Writer = os.Stderr
		fmt.Fprint(w, "through interface")
	})

	assert.Equal(t, "through interface", output)
}
