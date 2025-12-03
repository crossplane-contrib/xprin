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

// Package check provides the check subcommand for the xprin tool.
package check

import (
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
	configtypes "github.com/crossplane-contrib/xprin/internal/config"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/spf13/afero"
)

// Cmd represents the check subcommand.
type Cmd struct {
	Config     *configtypes.Config `kong:"-"`
	ConfigPath string              `kong:"-"`
	fs         afero.Fs
}

// AfterApply implements kong.AfterApply.
func (c *Cmd) AfterApply() error {
	c.fs = afero.NewOsFs()
	return nil
}

// Run executes the check subcommand.
func (c *Cmd) Run(_ *kong.Context) error {
	// combine all error messages
	var allErrors []string

	if c.ConfigPath == "" {
		utils.OutputPrintf("No configuration file provided, using detected dependencies\n")
	} else {
		utils.OutputPrintf("Configuration file: %s\n\n", c.ConfigPath)
	}

	// Always check dependencies, subcommands, and repositories
	if err := c.Config.CheckDependencies(); err != nil {
		allErrors = append(allErrors, err.Error())
	}

	if err := c.Config.CheckSubcommands(); err != nil {
		allErrors = append(allErrors, err.Error())
	}

	if err := c.Config.CheckRepositories(); err != nil {
		allErrors = append(allErrors, err.Error())
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("configuration check failed:\n%s", strings.Join(allErrors, "\n"))
	}

	utils.OutputPrintf("Configuration check successful\n")

	utils.OutputPrintf("\nDependencies:\n")

	for name, value := range c.Config.Dependencies {
		utils.OutputPrintf("- %s: %s\n", name, value)
	}

	if c.Config.Subcommands != nil {
		utils.OutputPrintf("\nSubcommands:\n")

		if c.Config.Subcommands.Render != "" {
			utils.OutputPrintf("- render: %s\n", c.Config.Subcommands.Render)
		}

		if c.Config.Subcommands.Validate != "" {
			utils.OutputPrintf("- validate: %s\n", c.Config.Subcommands.Validate)
		}
	}

	if len(c.Config.Repositories) > 0 {
		utils.OutputPrintf("\nRepositories:\n")

		for name, path := range c.Config.Repositories {
			utils.OutputPrintf("- %s: %s\n", name, path)
		}
	}

	return nil
}
