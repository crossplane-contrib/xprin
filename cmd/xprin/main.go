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

// Package main is the main package for the xprin tool.
package main

import (
	"errors"
	"log"
	"os"
	"slices"

	"github.com/alecthomas/kong"
	checkCmd "github.com/crossplane-contrib/xprin/cmd/xprin/check"
	configCmd "github.com/crossplane-contrib/xprin/cmd/xprin/config"
	"github.com/crossplane-contrib/xprin/cmd/xprin/test"
	"github.com/crossplane-contrib/xprin/cmd/xprin/version"
	internalConfig "github.com/crossplane-contrib/xprin/internal/config"
	"github.com/spf13/afero"
)

// CLI represents the command-line interface.
type CLI struct {
	ConfigFile string        `default:"~/.config/xprin.yaml" help:"Path to xprin config file"            short:"c" type:"path"`
	Check      checkCmd.Cmd  `cmd:""                         help:"Check dependencies and configuration"`
	Config     configCmd.Cmd `cmd:""                         help:"Manage xprin configuration"`
	Test       test.Cmd      `cmd:""                         help:"Run Crossplane tests"`
	Version    version.Cmd   `cmd:""                         help:"Print the version of xprin"`
}

// commandsNotRequiringConfig returns subcommand names that do not need config (e.g. version).
func commandsNotRequiringConfig() []string {
	return []string{"version"}
}

// loadConfigAndInject loads config (from file or Fallback) and injects it into the CLI command structs.
// Only called when the selected command is not in commandsNotRequiringConfig. Add new config-requiring commands here.
func loadConfigAndInject(cli *CLI, fs afero.Fs) error {
	configPath := cli.ConfigFile

	cfg, err := internalConfig.Load(fs, configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			configPath = ""
			cfg, err = internalConfig.Fallback()
		}

		if err != nil {
			return err
		}
	}

	cli.Check.Config = cfg
	cli.Check.ConfigPath = configPath
	cli.Config.Config = cfg
	cli.Config.ConfigPath = configPath
	cli.Test.Config = cfg

	return nil
}

func main() {
	var cli CLI

	ctx := kong.Parse(&cli,
		kong.Name("xprin"),
		kong.Description("A Crossplane Testing Framework."),
		kong.UsageOnError(),
	)

	if !slices.Contains(commandsNotRequiringConfig(), ctx.Command()) {
		if err := loadConfigAndInject(&cli, afero.NewOsFs()); err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	}

	if err := ctx.Run(); err != nil {
		log.Fatalf("%v", err)
	}
}
