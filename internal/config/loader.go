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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

// Config represents the main configuration structure.
type Config struct {
	Dependencies map[string]string `yaml:"dependencies"`
	Subcommands  *Subcommands      `yaml:"subcommands"`
	Repositories map[string]string `yaml:"repositories"`
}

// Subcommands holds the subcommand configurations.
type Subcommands struct {
	Render   string `yaml:"render"`
	Validate string `yaml:"validate"`
}

const (
	// CrossplaneCmd is the default name of the Crossplane command.
	CrossplaneCmd = "crossplane"

	// RenderSubcommand is the default name of the crossplane render subcommand.
	RenderSubcommand = "render"
	// RenderFlags are the default flags for the crossplane render subcommand.
	RenderFlags = "--include-full-xr"
	// DefaultRenderCmd is the default command for the crossplane render subcommand.
	DefaultRenderCmd = RenderSubcommand + " " + RenderFlags

	// ValidateSubcommand is the default name of the crossplane validate subcommand.
	ValidateSubcommand = "beta validate"
	// ValidateFlags are the default flags for the crossplane validate subcommand.
	ValidateFlags = "--error-on-missing-schemas"
	// DefaultValidateCmd is the default command for the crossplane validate subcommand.
	DefaultValidateCmd = ValidateSubcommand + " " + ValidateFlags
)

// Load loads and validates an xprin configuration file.
func Load(fs afero.Fs, configPath string) (*Config, error) {
	if !strings.HasSuffix(configPath, ".yaml") {
		return nil, fmt.Errorf("Config file must have .yaml extension")
	}

	expandedPath, err := utils.ExpandTildeAbs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand config path: %w", err)
	}

	data, err := afero.ReadFile(fs, expandedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}

		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Validate the YAML before unmarshalling
	if err := utils.ValidateYAML(data); err != nil {
		return nil, fmt.Errorf("invalid YAML in config file %s: %w", data, err)
	}

	// Try to parse with structure validation first
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Initialize maps if they're nil
	if cfg.Dependencies == nil {
		cfg.Dependencies = make(map[string]string)
	}

	if cfg.Subcommands == nil {
		cfg.Subcommands = &Subcommands{}
	}

	if cfg.Repositories == nil {
		cfg.Repositories = make(map[string]string)
	}

	// Set default subcommands if not provided
	if cfg.Subcommands.Render == "" {
		cfg.Subcommands.Render = DefaultRenderCmd
	}

	if cfg.Subcommands.Validate == "" {
		cfg.Subcommands.Validate = DefaultValidateCmd
	}

	return &cfg, nil
}

// Fallback returns a config that uses only binaries from PATH.
func Fallback() (*Config, error) {
	mandatoryDeps := []string{CrossplaneCmd}

	foundDeps := make(map[string]string)

	var missingMandatoryDeps []string

	// Check mandatory dependencies
	for _, dep := range mandatoryDeps {
		if _, err := exec.LookPath(dep); err != nil {
			missingMandatoryDeps = append(missingMandatoryDeps, dep)
		} else {
			foundDeps[dep] = dep
		}
	}

	// Fail only if mandatory dependencies are missing
	if len(missingMandatoryDeps) > 0 {
		return nil, fmt.Errorf("missing required dependencies from PATH (%s)", strings.Join(missingMandatoryDeps, ", "))
	}

	return &Config{
		Dependencies: foundDeps,
		Subcommands: &Subcommands{
			Render:   DefaultRenderCmd,
			Validate: DefaultValidateCmd,
		},
		Repositories: make(map[string]string),
	}, nil
}
