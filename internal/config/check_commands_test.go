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

package config

import (
	"strings"
	"testing"
)

func TestCheckSubcommands(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "nil subcommands section",
			cfg:  &Config{},
		},
		{
			name: "empty subcommands",
			cfg:  &Config{Subcommands: &Subcommands{}},
		},
		{
			name: "valid render and validate",
			cfg:  &Config{Subcommands: &Subcommands{Render: "render --foo", Validate: "validate --bar"}},
		},
		{
			name: "valid beta render and beta validate",
			cfg:  &Config{Subcommands: &Subcommands{Render: "beta render --foo", Validate: "beta validate --bar"}},
		},
		{
			name:    "invalid render prefix",
			cfg:     &Config{Subcommands: &Subcommands{Render: "foo --bar", Validate: "validate --bar"}},
			wantErr: "subcommands.render must start with 'render' or 'beta render'",
		},
		{
			name:    "invalid validate prefix",
			cfg:     &Config{Subcommands: &Subcommands{Render: "render --foo", Validate: "foo --bar"}},
			wantErr: "subcommands.validate must start with 'validate' or 'beta validate'",
		},
		{
			name:    "render with non-flag argument",
			cfg:     &Config{Subcommands: &Subcommands{Render: "render notaflag --foo", Validate: "validate --bar"}},
			wantErr: "subcommands.render: argument 1 ('notaflag') must be a flag",
		},
		{
			name:    "validate with non-flag argument",
			cfg:     &Config{Subcommands: &Subcommands{Render: "render --foo", Validate: "validate notaflag --bar"}},
			wantErr: "subcommands.validate: argument 1 ('notaflag') must be a flag",
		},
		{
			name:    "beta render with non-flag argument",
			cfg:     &Config{Subcommands: &Subcommands{Render: "beta render notaflag --foo", Validate: "validate --bar"}},
			wantErr: "subcommands.render: argument 2 ('notaflag') must be a flag",
		},
		{
			name:    "beta validate with non-flag argument",
			cfg:     &Config{Subcommands: &Subcommands{Render: "render --foo", Validate: "beta validate notaflag --bar"}},
			wantErr: "subcommands.validate: argument 2 ('notaflag') must be a flag",
		},
		{
			name: "render with short flag",
			cfg:  &Config{Subcommands: &Subcommands{Render: "render -x", Validate: "validate --bar"}},
		},
		{
			name: "beta render with short flag",
			cfg:  &Config{Subcommands: &Subcommands{Render: "beta render -x", Validate: "validate --bar"}},
		},
		{
			name: "validate with short flag",
			cfg:  &Config{Subcommands: &Subcommands{Render: "render --foo", Validate: "validate -y"}},
		},
		{
			name: "beta validate with short flag",
			cfg:  &Config{Subcommands: &Subcommands{Render: "render --foo", Validate: "beta validate -y"}},
		},
		{
			name: "only render set valid",
			cfg:  &Config{Subcommands: &Subcommands{Render: "render --foo"}},
		},
		{
			name:    "only render set invalid",
			cfg:     &Config{Subcommands: &Subcommands{Render: "foo --bar"}},
			wantErr: "subcommands.render must start with 'render' or 'beta render'",
		},
		{
			name: "only validate set valid",
			cfg:  &Config{Subcommands: &Subcommands{Validate: "validate --bar"}},
		},
		{
			name:    "only validate set invalid",
			cfg:     &Config{Subcommands: &Subcommands{Validate: "foo --bar"}},
			wantErr: "subcommands.validate must start with 'validate' or 'beta validate'",
		},
		{
			name: "render set without flags",
			cfg:  &Config{Subcommands: &Subcommands{Render: "render"}},
		},
		{
			name: "beta render set without flags",
			cfg:  &Config{Subcommands: &Subcommands{Render: "beta render"}},
		},
		{
			name: "validate set without flags",
			cfg:  &Config{Subcommands: &Subcommands{Validate: "validate"}},
		},
		{
			name: "beta validate set without flags",
			cfg:  &Config{Subcommands: &Subcommands{Validate: "beta validate"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.CheckSubcommands()
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("CheckSubcommands() error = nil, wantErr %q", tt.wantErr)
					return
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("CheckSubcommands() error = %v, wantErr %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("CheckSubcommands() error = %v, wantErr nil", err)
			}
		})
	}
}
