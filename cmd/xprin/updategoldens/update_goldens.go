/*
Copyright 2026 The Crossplane Authors.

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

// Package updategoldens provides the update-goldens subcommand for the xprin tool.
package updategoldens

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	internalcfg "github.com/crossplane-contrib/xprin/internal/config"
	"github.com/crossplane-contrib/xprin/internal/testexecution/processor"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	"github.com/gonvenience/bunt"
	"github.com/spf13/afero"
)

// Cmd represents the update-goldens subcommand.
type Cmd struct {
	Targets           []string            `arg:""                                                                                       help:"One or more test targets: individual files (e.g., 'tests/aws_xprin.yaml'), directories (e.g., 'tests/aws/'), or recursive directories (e.g., 'tests/aws/...'). Files must be named 'xprin.yaml' or '*_xprin.yaml'"`
	Verbose           bool                `help:"Show verbose output (RUN, PASS/FAIL/SKIP, written files)."                             short:"v"`
	Quiet             bool                `help:"Suppress '[no testsuite files]' and '[no test cases found]' messages."                 name:"quiet"                                                                                                                                                                                                             short:"q"`
	Debug             bool                `help:"Show detailed debug information about test discovery, path resolution, and execution."`
	Color             string              `default:"auto"                                                                               enum:"on,off,auto"                                                                                                                                                                                                       help:"Specify color usage: on, off, or auto (default auto)." name:"color"`
	CrossplaneVersion string              `help:"Version of the Crossplane controller image, passed to crossplane render."              name:"crossplane-version"`
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

// Run executes the update-goldens subcommand.
func (c *Cmd) Run(_ *kong.Context) error {
	switch c.Color {
	case "on":
		bunt.SetColorSettings(bunt.ON, bunt.ON)
	case "off":
		bunt.SetColorSettings(bunt.OFF, bunt.OFF)
	default: // "auto"
		bunt.SetColorSettings(bunt.AUTO, bunt.AUTO)
	}

	return processor.ProcessTargets(c.fs, c.Targets, c.newOptions(c.Config))
}

// newOptions creates a testexecutionUtils.Options struct from the command and config.
func (c *Cmd) newOptions(cfg *internalcfg.Config) *testexecutionUtils.Options {
	var render, validate []string

	if cfg.Subcommands != nil {
		render = strings.Fields(cfg.Subcommands.Render)
		validate = strings.Fields(cfg.Subcommands.Validate)
	}

	return &testexecutionUtils.Options{
		Dependencies:      cfg.Dependencies,
		Repositories:      cfg.Repositories,
		Verbose:           c.Verbose,
		Quiet:             c.Quiet,
		Debug:             c.Debug,
		Color:             bunt.UseColors(),
		Render:            render,
		Validate:          validate,
		XPCLI:             cfg.XPCLI,
		CrossplaneVersion: c.CrossplaneVersion,
		WorkingDir:        c.cwd,
		UpdateGoldens:     true,
	}
}
