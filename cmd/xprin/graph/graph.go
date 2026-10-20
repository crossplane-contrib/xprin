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

// Package graph provides the graph subcommand for xprin.
package graph

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/crossplane-contrib/xprin/internal/discovery"
	"github.com/crossplane-contrib/xprin/internal/graph"
	"github.com/spf13/afero"
)

// Cmd represents the graph subcommand.
type Cmd struct {
	Targets []string `arg:""                                                                       help:"One or more test targets: individual files (e.g., 'tests/aws_xprin.yaml'), directories (e.g., 'tests/aws/'), or recursive directories (e.g., 'tests/aws/...'). Files must be named 'xprin.yaml' or '*_xprin.yaml'"`
	Quiet   bool     `help:"Suppress '[no testsuite files]' and '[no test cases found]' messages." name:"quiet"                                                                                                                                                                                                             short:"q"`
	fs      afero.Fs
}

// AfterApply implements kong.AfterApply.
func (c *Cmd) AfterApply() error {
	c.fs = afero.NewOsFs()

	return nil
}

// Run executes the graph subcommand.
func (c *Cmd) Run(_ *kong.Context) error {
	return graphTargets(c.fs, os.Stdout, c.Targets, c.Quiet)
}

// graphTargets writes the dependency tree of the testcases of every testsuite file that the targets resolve to.
func graphTargets(fs afero.Fs, out io.Writer, targets []string, quiet bool) error {
	options := discovery.Options{Quiet: quiet}
	first := true

	return discovery.Walk(fs, targets, options, func(testSuiteFile string) error {
		testSuite, err := discovery.Load(fs, testSuiteFile, options)
		if err != nil || testSuite == nil {
			return err
		}

		if !first {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
		}

		first = false

		return graph.WriteTree(out, testSuite.File, testSuite.Spec, testSuite.Graph)
	})
}
