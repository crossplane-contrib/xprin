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

// Package test provides the test subcommand for the xprin tool.
package test

import (
	"strings"

	"github.com/alecthomas/kong"
	internalcfg "github.com/crossplane-contrib/xprin/internal/config"
	"github.com/crossplane-contrib/xprin/internal/testexecution/processor"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/gonvenience/bunt"
	"github.com/spf13/afero"
)

// Cmd represents the test subcommand.
type Cmd struct {
	Targets        []string            `arg:""                                                                                      help:"One or more test targets: individual files (e.g., 'tests/aws_xprin.yaml'), directories (e.g., 'tests/aws/'), or recursive directories (e.g., 'tests/aws/...'). Files must be named 'xprin.yaml' or '*_xprin.yaml'"`
	ShowRender     bool                `help:"Display a list of the rendered resources in Kind/Name format. Requires --verbose."    name:"show-render"`
	ShowValidate   bool                `help:"Display validation results for each resource. Requires --verbose."                    name:"show-validate"`
	ShowHooks      bool                `help:"Display the execution hooks for each test case. Requires --verbose."                  name:"show-hooks"`
	ShowAssertions bool                `help:"Display assertion results for each test case. Requires --verbose."                    name:"show-assertions"`
	Verbose        bool                `help:"Show verbose test output and results (similar to go test -v)"                         short:"v"`
	Debug          bool                `help:"Show detailed debug information about test discovery, path resolution, and execution"`
	Color          string              `default:"auto"                                                                              enum:"on,off,auto"                                                                                                                                                                                                       help:"Specify color usage: on, off, or auto (default auto)." name:"color"`
	Config         *internalcfg.Config `kong:"-"`
	fs             afero.Fs
}

// AfterApply implements kong.AfterApply.
func (c *Cmd) AfterApply() error {
	c.fs = afero.NewOsFs()
	return nil
}

// Run executes the test subcommand.
func (c *Cmd) Run(_ *kong.Context) error {
	// Warn if render flag is used without -v
	if c.ShowRender && !c.Verbose {
		utils.WarningPrintf("--show-render requires -v (verbose mode) to display results.\n")
	}

	if c.ShowValidate && !c.Verbose {
		utils.WarningPrintf("--show-validate requires -v (verbose mode) to display results.\n")
	}

	if c.ShowHooks && !c.Verbose {
		utils.WarningPrintf("--show-hooks requires -v (verbose mode) to display results.\n")
	}

	if c.ShowAssertions && !c.Verbose {
		utils.WarningPrintf("--show-assertions requires -v (verbose mode) to display results.\n")
	}

	// Set bunt color state from CLI so diff/dyff and bunt.UseColors() stay in sync.
	switch c.Color {
	case "on":
		bunt.SetColorSettings(bunt.ON, bunt.ON)
	case "off":
		bunt.SetColorSettings(bunt.OFF, bunt.OFF)
	default: // "auto"
		bunt.SetColorSettings(bunt.AUTO, bunt.AUTO)
	}

	options := c.newOptions(c.Config)

	// Process targets and run tests
	return processor.ProcessTargets(c.fs, c.Targets, options)
}

// newOptions creates a testexecutionUtils.Options struct from a Command and Config.
func (c *Cmd) newOptions(cfg *internalcfg.Config) *testexecutionUtils.Options {
	var render, validate []string

	if cfg.Subcommands != nil {
		render = strings.Fields(cfg.Subcommands.Render)
		validate = strings.Fields(cfg.Subcommands.Validate)
	}

	return &testexecutionUtils.Options{
		Dependencies:   cfg.Dependencies,
		Repositories:   cfg.Repositories,
		ShowRender:     c.ShowRender,
		ShowValidate:   c.ShowValidate,
		ShowHooks:      c.ShowHooks,
		ShowAssertions: c.ShowAssertions,
		Verbose:        c.Verbose,
		Debug:          c.Debug,
		Color:          bunt.UseColors(),
		Render:         render,
		Validate:       validate,
	}
}
