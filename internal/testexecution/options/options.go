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

// Package options has the options of a test run, which the commands fill in and the test execution reads.
package options

import (
	"context"
	"io"
	"log"
	"os"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/config"
	"github.com/crossplane-contrib/xprin/internal/testexecution/containers"
	"github.com/crossplane-contrib/xprin/internal/xpcli"
)

// Options are the options of a test run, which the commands fill in.
type Options struct {
	Dependencies      map[string]string
	Repositories      map[string]string
	ShowRender        bool
	ShowValidate      bool
	ShowHooks         bool
	ShowAssertions    bool
	Verbose           bool
	Debug             bool
	DebugLog          *log.Logger
	Color             bool            // diff output is colorized (--color, resolved)
	Quiet             bool            // no "[no testsuite files]" and "[no test cases found]" messages
	Context           context.Context //nolint:containedctx // canceled when the run is interrupted, nil if it can't be
	Parallel          int             // how many test cases run at the same time, 0 is 1
	Render            []string
	Validate          []string
	XPCLI             xpcli.XPCLI
	CrossplaneVersion string
	WorkingDir        string                  // where `xprin test` started, for the paths it shows and the artifacts
	ArtifactsBaseDir  string                  // --artifacts-dir, empty disables the export
	ArtifactsRunDir   string                  // <ArtifactsBaseDir>/xprin-artifacts-<time>/, set on the first test run
	UpdateGoldens     bool                    // write the golden files instead of comparing (no validate, assertions or post-test hooks)
	Containers        *containers.Coordinator // the container reuse of the invocation, nil disables it
	NoContainerReuse  bool                    // --no-container-reuse, ProcessTargets makes no Containers
}

// FromConfig returns the options that the configuration decides: the dependencies, the repositories, the
// subcommands to render and validate with (split into words), and the Crossplane CLI. The command that runs
// fills in the rest, from its flags.
func FromConfig(cfg *config.Config) *Options {
	var render, validate []string

	if cfg.Subcommands != nil {
		render = strings.Fields(cfg.Subcommands.Render)
		validate = strings.Fields(cfg.Subcommands.Validate)
	}

	return &Options{
		Dependencies: cfg.Dependencies,
		Repositories: cfg.Repositories,
		Render:       render,
		Validate:     validate,
		XPCLI:        cfg.XPCLI,
	}
}

// Interrupted tells whether the run was interrupted.
func (o *Options) Interrupted() bool {
	return o.Context != nil && o.Context.Err() != nil
}

// SetDebug turns --debug on or off. DebugLog is where its lines go: stderr with it, nowhere without it.
func (o *Options) SetDebug(debug bool) {
	w := io.Discard
	if debug {
		w = os.Stderr
	}

	o.Debug = debug
	o.DebugLog = log.New(w, "DEBUG: ", 0)
}
