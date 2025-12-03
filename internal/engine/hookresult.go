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

// Package engine provides the core functionality for running the tests.
package engine

// HookResult represents the result of executing a single hook.
type HookResult struct {
	Name    string // Hook name (optional)
	Command string // The command that was executed
	Stdout  []byte // Standard output
	Stderr  []byte // Standard error
	Error   error  // Execution error (nil if successful)
}

// NewHookResult creates a new HookResult with the given parameters.
func NewHookResult(name, command string, stdout, stderr []byte, err error) HookResult {
	return HookResult{
		Name:    name,
		Command: command,
		Stdout:  stdout,
		Stderr:  stderr,
		Error:   err,
	}
}
