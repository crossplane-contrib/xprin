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

// Package config provides the config subcommand for the xprin tool.
package config

import (
	"github.com/alecthomas/kong"
	configtypes "github.com/crossplane-contrib/xprin/internal/config"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/spf13/afero"
)

// Cmd represents the config subcommand.
type Cmd struct {
	Check      bool                `help:"Check dependencies and repository configuration"`
	Config     *configtypes.Config `kong:"-"`
	ConfigPath string              `kong:"-"`
	fs         afero.Fs
}

// AfterApply implements kong.AfterApply.
func (c *Cmd) AfterApply() error {
	c.fs = afero.NewOsFs()
	return nil
}

// Run executes the config subcommand.
func (c *Cmd) Run(_ *kong.Context) error {
	if c.Check {
		return configtypes.RunCheck(c.Config, c.ConfigPath, false)
	}

	if c.ConfigPath == "" {
		utils.OutputPrintf("No configuration file provided.\n")
		return nil
	}

	utils.OutputPrintf("Configuration file: %s\n", c.ConfigPath)

	if len(c.Config.Dependencies) > 0 {
		utils.OutputPrintf("\nDependencies:\n")

		for name, value := range c.Config.Dependencies {
			utils.OutputPrintf("- %s: %s\n", name, value)
		}
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
