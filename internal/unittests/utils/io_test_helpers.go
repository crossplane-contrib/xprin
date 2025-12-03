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
	"bytes"
	"io"
	"os"
)

// CaptureStderr captures output written to os.Stderr during the execution of function f.
// This is useful for testing functions that write output directly to stderr.
// This function also safely handles panics by ensuring os.Stderr is properly restored.
// Example:
//
//	output := testutils.CaptureStderr(func() {
//	   utils.WarningPrintf("Warning message")
//	})
//	assert.Contains(t, output, "Warning message")
func CaptureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Use defer to ensure everything gets properly restored and closed
	// even if f() panics
	var buf bytes.Buffer

	defer func() {
		// First restore the original stderr
		os.Stderr = old
		// Then recover from panic if any
		_ = recover()
	}()

	f()

	// Close the write end of the pipe to unblock the reader
	w.Close() //nolint:errcheck // cleanup function, error handling not practical
	// Copy the captured output to our buffer
	_, _ = io.Copy(&buf, r)

	return buf.String()
}

// CaptureStdout captures output written to os.Stdout during the execution of function f.
// This is useful for testing functions that write output directly to stdout.
// This function also safely handles panics by ensuring os.Stdout is properly restored.
// Example:
//
//	output := testutils.CaptureStdout(func() {
//	   fmt.Printf("Hello world")
//	})
//	assert.Contains(t, output, "Hello world")
func CaptureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Use defer to ensure everything gets properly restored and closed
	// even if f() panics
	var buf bytes.Buffer

	defer func() {
		// First restore the original stdout
		os.Stdout = old
		// Then recover from panic if any
		_ = recover()
	}()

	f()

	// Close the write end of the pipe to unblock the reader
	w.Close() //nolint:errcheck // cleanup function, error handling not practical
	// Copy the captured output to our buffer
	_, _ = io.Copy(&buf, r)

	return buf.String()
}

// CapturedOutput represents the captured stdout and stderr output from a function.
type CapturedOutput struct {
	Stdout string
	Stderr string
}

// CaptureOutput captures both stdout and stderr output simultaneously during the execution of function f.
// This is useful for testing functions that write output to both stdout and stderr.
// This function also safely handles panics by ensuring both stdout and stderr are properly restored.
// Example:
//
//	output := testutils.CaptureOutput(func() {
//	   fmt.Printf("Standard output")
//	   fmt.Fprintf(os.Stderr, "Error output")
//	})
//	assert.Contains(t, output.Stdout, "Standard output")
//	assert.Contains(t, output.Stderr, "Error output")
func CaptureOutput(f func()) CapturedOutput {
	// Create buffer for stderr
	var stderrBuf bytes.Buffer

	// Capture stdout and run the function that captures stderr
	output := CapturedOutput{
		Stdout: CaptureStdout(func() {
			output := CaptureStderr(f)
			stderrBuf.WriteString(output) // Save stderr output
		}),
	}

	// Add stderr to our result
	output.Stderr = stderrBuf.String()

	return output
}
