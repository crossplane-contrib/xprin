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
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	internalcfg "github.com/crossplane-contrib/xprin/internal/config"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testexecution/processor"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/crossplane-contrib/xprin/internal/xpcli"
	"github.com/gonvenience/bunt"
	"github.com/spf13/afero"
)

// Cmd represents the test subcommand.
type Cmd struct {
	Targets           []string            `arg:""                                                                                                                                                                                                            help:"One or more test targets: individual files (e.g., 'tests/aws_xprin.yaml'), directories (e.g., 'tests/aws/'), or recursive directories (e.g., 'tests/aws/...'). Files must be named 'xprin.yaml' or '*_xprin.yaml'"`
	ShowAll           bool                `help:"Enable all --show-* flags. Requires --verbose."                                                                                                                                                             name:"show-all"`
	ShowRender        bool                `help:"Display a list of the rendered resources in Kind/Name format. Requires --verbose."                                                                                                                          name:"show-render"`
	ShowValidate      bool                `help:"Display validation results for each resource. Requires --verbose."                                                                                                                                          name:"show-validate"`
	ShowHooks         bool                `help:"Display the execution hooks for each test case. Requires --verbose."                                                                                                                                        name:"show-hooks"`
	ShowAssertions    bool                `help:"Display assertion results for each test case. Requires --verbose."                                                                                                                                          name:"show-assertions"`
	Verbose           bool                `help:"Show verbose test output (RUN, PASS/FAIL, etc.). Independent of -q; can be used together."                                                                                                                  short:"v"`
	Quiet             bool                `help:"Suppress '[no testsuite files]' and '[no test cases found]' messages. Independent of -v; does not reduce verbose output."                                                                                   name:"quiet"                                                                                                                                                                                                             short:"q"`
	Debug             bool                `help:"Show detailed debug information about test discovery, path resolution, and execution"`
	Color             string              `default:"auto"                                                                                                                                                                                                    enum:"on,off,auto"                                                                                                                                                                                                       help:"Specify color usage: on, off, or auto (default auto)." name:"color"`
	CrossplaneVersion string              `help:"Version of the Crossplane controller image, passed at both render and validate."                                                                                                                            name:"crossplane-version"`
	NoContainerReuse  bool                `help:"Disable reuse of Docker containers for composition functions. Every render call creates and tears down a fresh container, instead of reusing one container per function image across the whole invocation." name:"no-container-reuse"`
	ArtifactsBaseDir  string              `help:"Parent directory for the exported external artifacts. Each invocation creates <dir>/xprin-artifacts-<YYYYMMDDHHMMSS>/ containing inputs/ and outputs/ for every test case."                                 name:"artifacts-dir"                                                                                                                                                                                                     short:"a"`
	Config            *internalcfg.Config `kong:"-"`
	cwd               string
	fs                afero.Fs
}

// AfterApply implements kong.AfterApply.
func (c *Cmd) AfterApply() error {
	c.fs = afero.NewOsFs()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	c.cwd = cwd

	return nil
}

// Run executes the test subcommand.
func (c *Cmd) Run(_ *kong.Context) error {
	// Warn if a show flag is used without -v
	if c.ShowAll && !c.Verbose {
		utils.WarningPrintf("--show-all requires -v (verbose mode) to display results.\n")
	}

	if c.ShowRender && !c.ShowAll && !c.Verbose {
		utils.WarningPrintf("--show-render requires -v (verbose mode) to display results.\n")
	}

	if c.ShowValidate && !c.ShowAll && !c.Verbose {
		utils.WarningPrintf("--show-validate requires -v (verbose mode) to display results.\n")
	}

	if c.ShowHooks && !c.ShowAll && !c.Verbose {
		utils.WarningPrintf("--show-hooks requires -v (verbose mode) to display results.\n")
	}

	if c.ShowAssertions && !c.ShowAll && !c.Verbose {
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

	if c.Debug {
		if c.Config.XPCLI.Tier == xpcli.TierV1 {
			utils.DebugPrintf("Crossplane CLI v1 detected; patches.xrd will not be passed as --xrd to render\n")
		}

		if c.CrossplaneVersion != "" && !c.Config.XPCLI.SupportsRenderVersion() {
			utils.DebugPrintf("--crossplane-version is not supported by the legacy Crossplane CLI (< v2.3.0); skipping for render, but --crossplane-image will still be passed to validate\n")
		}
	}

	// Process targets and run tests
	return processor.ProcessTargets(c.fs, c.Targets, options)
}

// newOptions creates a testexecutionOptions.Options struct from a Command and Config.
func (c *Cmd) newOptions(cfg *internalcfg.Config) *testexecutionOptions.Options {
	opts := testexecutionOptions.FromConfig(cfg)
	opts.ShowRender = c.ShowAll || c.ShowRender
	opts.ShowValidate = c.ShowAll || c.ShowValidate
	opts.ShowHooks = c.ShowAll || c.ShowHooks
	opts.ShowAssertions = c.ShowAll || c.ShowAssertions
	opts.Verbose = c.Verbose
	opts.Quiet = c.Quiet
	opts.Debug = c.Debug
	opts.Color = bunt.UseColors()
	opts.CrossplaneVersion = c.CrossplaneVersion
	opts.NoContainerReuse = c.NoContainerReuse
	opts.WorkingDir = c.cwd
	opts.ArtifactsBaseDir = c.ArtifactsBaseDir

	return opts
}
