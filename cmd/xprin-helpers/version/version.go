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

// Package version provides the version subcommand for xprin-helpers.
package version

import (
	"github.com/alecthomas/kong"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/crossplane-contrib/xprin/internal/version"
)

// Cmd represents the version subcommand.
type Cmd struct{}

// AfterApply implements kong.AfterApply.
func (c *Cmd) AfterApply(_ *kong.Context) error {
	// No setup needed, just return nil
	return nil
}

// Run executes the version subcommand.
func (c *Cmd) Run(_ *kong.Context) error {
	utils.OutputPrintf("%s\n", version.GetVersion())
	return nil
}
