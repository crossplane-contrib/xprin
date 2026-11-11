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

package test

import (
	"runtime"
	"testing"

	"github.com/alecthomas/kong"
	internalcfg "github.com/crossplane-contrib/xprin/internal/config"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func parseTest(t *testing.T, args ...string) (*Cmd, error) {
	t.Helper()

	var cli struct {
		Test Cmd `cmd:""`
	}

	parser, err := kong.New(&cli)
	require.NoError(t, err)

	_, err = parser.Parse(append([]string{"test"}, args...))

	return &cli.Test, err
}

func TestParallel_Parse(t *testing.T) {
	cpus := Parallelism(runtime.GOMAXPROCS(0))

	tests := map[string]struct {
		args        []string
		want        Parallelism
		wantTargets []string
	}{
		"not given: one at a time":           {[]string{"tests/"}, 0, []string{"tests/"}},
		"a number":                           {[]string{"--parallel", "4", "tests/"}, 4, []string{"tests/"}},
		"the number with =":                  {[]string{"--parallel=8", "tests/"}, 8, []string{"tests/"}},
		"the short flag":                     {[]string{"-p", "2", "tests/"}, 2, []string{"tests/"}},
		"attached to the short flag":         {[]string{"-p4", "tests/"}, 4, []string{"tests/"}},
		"one":                                {[]string{"-p", "1", "tests/"}, 1, []string{"tests/"}},
		"the number after the targets":       {[]string{"tests/", "other/", "-p", "3"}, 3, []string{"tests/", "other/"}},
		"without a number: the CPUs":         {[]string{"--parallel", "tests/"}, cpus, []string{"tests/"}},
		"the short flag without a number":    {[]string{"-p", "tests/"}, cpus, []string{"tests/"}},
		"without a number, after the target": {[]string{"tests/", "--parallel"}, cpus, []string{"tests/"}},
		"without a number, with other flags": {[]string{"-p", "-v", "tests/"}, cpus, []string{"tests/"}},
		"auto is not a number, but a target": {[]string{"--parallel", "auto", "tests/"}, cpus, []string{"auto", "tests/"}},
		"a target that is a number, as ./4":  {[]string{"--parallel", "./4"}, cpus, []string{"./4"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cmd, err := parseTest(t, tc.args...)

			require.NoError(t, err)
			assert.Equal(t, tc.want, cmd.Parallel)
			assert.Equal(t, tc.wantTargets, cmd.Targets)
		})
	}
}

func TestParallel_ParseErrors(t *testing.T) {
	for name, args := range map[string][]string{
		"zero":                {"--parallel=0", "tests/"},
		"zero after the flag": {"--parallel", "0", "tests/"},
		"negative":            {"--parallel=-1", "tests/"},
		"text":                {"--parallel=many", "tests/"},
		"auto is not a value": {"--parallel=auto", "tests/"},
		"decimal":             {"--parallel=1.5", "tests/"},
		"empty":               {"--parallel=", "tests/"},
		"percent":             {"--parallel=50%", "tests/"},
		"short flag, zero":    {"-p", "0", "tests/"},
		"huge, not a number":  {"--parallel=99999999999999999999", "tests/"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseTest(t, args...)

			require.Error(t, err)
		})
	}

	_, err := parseTest(t, "--parallel=0", "tests/")
	require.ErrorContains(t, err, "expected a number of test cases greater than 0")
}

func TestNewOptions_Parallel(t *testing.T) {
	cfg := &internalcfg.Config{}

	assert.Equal(t, 0, (&Cmd{}).newOptions(cfg).Parallel)
	assert.Equal(t, 6, (&Cmd{Parallel: 6}).newOptions(cfg).Parallel)
}
