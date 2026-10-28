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

// Package testsuiteneeds implements the command that checks that the test cases of the testsuite files list in
// `needs` the test cases whose outputs they use (`.Tests`).
package testsuiteneeds

import (
	"bufio"
	"errors"
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/crossplane-contrib/xprin/internal/discovery"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/spf13/afero"
)

// Cmd arguments and flags for checking the `needs` of testsuite files.
type Cmd struct {
	// Arguments.
	Targets []string `arg:"" help:"One or more test targets: individual files (e.g., 'tests/aws_xprin.yaml'), directories (e.g., 'tests/aws/'), or recursive directories (e.g., 'tests/aws/...'). Files must be named 'xprin.yaml' or '*_xprin.yaml'" predictor:"file"`

	// Flags.
	Verbose bool `help:"Show every test case, also the ones that have nothing to say."         name:"verbose" short:"v"`
	Quiet   bool `help:"Suppress '[no testsuite files]' and '[no test cases found]' messages." name:"quiet"   short:"q"`

	fs afero.Fs
}

// Help returns help message for the check-testsuite-needs command.
func (c *Cmd) Help() string {
	return `
Check that the test cases of the testsuite files list in 'needs' the test cases whose outputs they use.

A test case that uses the outputs of another one ('.Tests.<id>' in a template) has to list it in
'needs'. xprin used to find them by reading the templates, and since 0.7.0 it does not. This
command checks every test case of every file, also in the files that already use 'needs', and shows
the IDs that are missing in the 'needs' of the test cases. It exits with 1 if anything is missing or
has to be fixed, so it can be used to check that the files are migrated. A 'needs' that is not used
through '.Tests' is fine: it is a legitimate way to order test cases.

The output is the one of 'xprin test': a result for each test case, and a line for each file.
A test case fails when it misses needs (the 'Missing Needs:' section, with a '[x]' for each) or
when a person has to fix something in it ('[!]'). It passes when it uses other test cases and lists
all it uses, and is skipped when it uses none. A skip that has a reason ('[!]') is shown too: it uses
'.Tests' in a way that can't be traced, and it lists needs that can't be checked. With -v all the test
cases are shown.

What can't be decided is reported for a person to fix: a use of '.Tests' that can't be traced to a
test case (for example 'range .Tests'), an ID that no test case has, a test case that uses its own
outputs, and needs that would make test cases need each other.

Examples:

  # Check a testsuite file
  xprin-helpers check-testsuite-needs tests/aws_xprin.yaml

  # Check all the testsuite files under tests/, for example in a pipeline
  xprin-helpers check-testsuite-needs tests/...
`
}

// AfterApply implements kong.AfterApply.
func (c *Cmd) AfterApply() error {
	c.fs = afero.NewOsFs()

	return nil
}

// Run runs the check-testsuite-needs command. The error is set when anything is missing, or requires a person.
func (c *Cmd) Run(k *kong.Context) error {
	var missNeeds, requireAPerson int

	errOut := bufio.NewWriter(k.Stderr)

	walkErr := discovery.Walk(c.fs, c.Targets, discovery.Options{Quiet: c.Quiet}, func(testSuiteFile string) error {
		// Created before the file is loaded, so that its duration is the time of everything
		testSuiteResult := engine.NewTestSuiteResult(testSuiteFile, c.Verbose)

		testSuite, err := discovery.Load(c.fs, testSuiteFile, discovery.Options{Quiet: c.Quiet})
		if err != nil || testSuite == nil {
			return err
		}

		res := checkFile(testSuite.Spec)

		// The errors of the writes in report are kept by the writer, and come out when it is flushed
		out := bufio.NewWriter(k.Stdout)
		report(out, testSuiteResult, res, c.Verbose)

		if err := out.Flush(); err != nil {
			return fmt.Errorf("failed to write the report of %s: %w", testSuiteFile, err)
		}

		if res.missesNeeds() {
			missNeeds++
		}

		// Every file that makes the run fail is an error here, so that the walk ends with a FAIL, as xprin test does
		switch {
		case res.requiresAPerson():
			requireAPerson++

			return errors.New("requires a person")
		case res.missesNeeds():
			return errors.New("misses needs")
		}

		return nil
	})

	writeSummary(errOut, missNeeds, requireAPerson)

	if err := errOut.Flush(); err != nil && walkErr == nil {
		return fmt.Errorf("failed to write the summary: %w", err)
	}

	return walkErr
}
