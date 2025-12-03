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

package engine

// AssertionResult represents the result of executing an assertion.
type AssertionResult struct {
	Name    string
	Status  Status // StatusPass or StatusFail
	Message string
}

// NewAssertionResult creates a new AssertionResult with the given parameters.
func NewAssertionResult(name string, status Status, message string) AssertionResult {
	return AssertionResult{
		Name:    name,
		Status:  status,
		Message: message,
	}
}
