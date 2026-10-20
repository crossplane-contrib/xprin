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

package options

import (
	"testing"

	"github.com/crossplane-contrib/xprin/internal/config"
	"github.com/stretchr/testify/assert" //nolint:depguard // testify is widely used for testing
)

func TestFromConfig(t *testing.T) {
	t.Run("what the configuration decides", func(t *testing.T) {
		cfg := &config.Config{
			Dependencies: map[string]string{"crossplane": "/usr/bin/crossplane"},
			Repositories: map[string]string{"myrepo": "/path/to/myrepo"},
			Subcommands:  &config.Subcommands{Render: "crossplane render --verbose", Validate: "crossplane beta validate"},
		}

		opts := FromConfig(cfg)

		assert.Equal(t, cfg.Dependencies, opts.Dependencies)
		assert.Equal(t, cfg.Repositories, opts.Repositories)
		assert.Equal(t, []string{"crossplane", "render", "--verbose"}, opts.Render, "split into words")
		assert.Equal(t, []string{"crossplane", "beta", "validate"}, opts.Validate)
		assert.Equal(t, cfg.XPCLI, opts.XPCLI)
	})

	t.Run("no subcommands: the defaults are used", func(t *testing.T) {
		opts := FromConfig(&config.Config{})

		assert.Nil(t, opts.Render)
		assert.Nil(t, opts.Validate)
	})

	t.Run("empty subcommands are no words", func(t *testing.T) {
		opts := FromConfig(&config.Config{Subcommands: &config.Subcommands{}})

		assert.Empty(t, opts.Render)
		assert.Empty(t, opts.Validate)
	})

	t.Run("what the command decides is not set", func(t *testing.T) {
		opts := FromConfig(&config.Config{})

		assert.False(t, opts.Verbose)
		assert.False(t, opts.UpdateGoldens)
		assert.Empty(t, opts.WorkingDir)
		assert.Nil(t, opts.Containers)
	})
}
