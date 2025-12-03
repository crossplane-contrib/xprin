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

// Package patchxr implements the command for patching a Crossplane XR (Composite Resource).
package patchxr

import (
	"bufio"
	"os"

	"github.com/alecthomas/kong"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/crossplane/crossplane-runtime/pkg/errors"

	commonIO "github.com/crossplane/crossplane/cmd/crank/beta/convert/io"
)

// Cmd arguments and flags for patching a Crossplane XR (Composite Resource).
type Cmd struct {
	// Arguments.
	InputFile string `arg:"" default:"-" help:"The XR YAML file to be patched. If not specified or '-', stdin will be used." optional:"" predictor:"file" type:"path"`

	// Output Flags.
	OutputFile string `help:"The file to write the patched XR YAML to. If not specified, stdout will be used." placeholder:"PATH" predictor:"file" short:"o" type:"path"`

	// Patching Flags.
	AddConnectionSecret       bool   `help:"Add writeConnectionSecretToRef to the XR spec. Must be explicitly set to true when using connection-secret-name or connection-secret-namespace."                                        name:"add-connection-secret"`
	ConnectionSecretName      string `help:"Custom name for the connection secret. If not specified, it generates a random UUID. Requires --add-connection-secret=true."                                                            name:"connection-secret-name"      type:"string"`
	ConnectionSecretNamespace string `help:"Custom namespace for the connection secret. If not specified, 'default' will be used. Requires --add-connection-secret=true."                                                           name:"connection-secret-namespace" type:"string"`
	XRD                       string `help:"A YAML file specifying the CompositeResourceDefinition (XRD) that defines the XR's schema and properties. When provided, default values from the XRD schema will be applied to the XR." name:"xrd"                         placeholder:"PATH" predictor:"file" type:"path"`

	fs afero.Fs
}

// Help returns help message for the patch-xr command.
func (c *Cmd) Help() string {
	return `
Patch a Crossplane Composite Resource (XR) with additional configurations.

This command will:
1. Read the XR from the provided YAML file
2. Apply any requested patches to the XR (such as connection secret)
3. Apply default values from an XRD if provided

Examples:

  # Apply default values from an XRD
  xprin-helpers patch-xr xr.yaml --xrd=composite-resource-definition.yaml

  # Add connection secret to the XR
  xprin-helpers patch-xr xr.yaml --add-connection-secret

  # Add connection secret with custom name and namespace (requires explicit --add-connection-secret=true)
  xprin-helpers patch-xr xr.yaml --add-connection-secret=true --connection-secret-name=my-secret --connection-secret-namespace=my-namespace

  # Add connection secret with just custom name (requires explicit --add-connection-secret=true)
  xprin-helpers patch-xr xr.yaml --add-connection-secret=true --connection-secret-name=my-secret

  # Combine patching flags
  xprin-helpers patch-xr xr.yaml --add-connection-secret --xrd=xrd.yaml

  # Patch XR from stdin
  cat xr.yaml | xprin-helpers patch-xr - --add-connection-secret
`
}

// AfterApply implements kong.AfterApply.
func (c *Cmd) AfterApply() error {
	c.fs = afero.NewOsFs()
	return nil
}

// Run runs the patch-xr command.
func (c *Cmd) Run(k *kong.Context) error {
	// Validate that --add-connection-secret=true is explicitly set when using connection secret name/namespace
	if (c.ConnectionSecretName != "" || c.ConnectionSecretNamespace != "") && !c.AddConnectionSecret {
		return errors.New("--add-connection-secret=true is required when using --connection-secret-name or --connection-secret-namespace")
	}

	// Validate that at least one patching flag is provided
	if !c.hasPatchingFlags() {
		return errors.New("no patching flags provided - nothing to do")
	}

	xrData, err := commonIO.Read(c.fs, c.InputFile)
	if err != nil {
		return err
	}

	xr := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(xrData, xr); err != nil {
		return errors.Wrap(err, "Unmarshalling Error")
	}

	// Apply XRD defaults if XRD file is provided
	// Based on the `crossplane render --xrd` flag of Crossplane CLI v2
	// https://github.com/crossplane/crossplane/blob/v2.0.2/cmd/crank/render/cmd.go#L186-L200
	if c.XRD != "" {
		xrd, err := LoadXRD(c.fs, c.XRD)
		if err != nil {
			return err
		}

		if err := DefaultValuesFromXRD(xr.UnstructuredContent(), xr.GetAPIVersion(), *xrd); err != nil {
			return errors.Wrap(err, "failed to apply XRD defaults")
		}
	}

	// Add connection secret if requested
	if c.AddConnectionSecret {
		if err := AddConnectionSecret(xr, c.ConnectionSecretName, c.ConnectionSecretNamespace); err != nil {
			return errors.Wrap(err, "failed to add connection secret")
		}
	}

	b, err := yaml.Marshal(xr)
	if err != nil {
		return errors.Wrap(err, "Unable to marshal back to yaml")
	}

	output := k.Stdout

	if outputFileName := c.OutputFile; outputFileName != "" {
		f, err := c.fs.OpenFile(outputFileName, os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return errors.Wrap(err, "Unable to open output file")
		}

		defer func() { _ = f.Close() }()

		output = f
	}

	outputW := bufio.NewWriter(output)
	if _, err := outputW.WriteString("---\n"); err != nil {
		return errors.Wrap(err, "Writing YAML file header")
	}

	if _, err := outputW.Write(b); err != nil {
		return errors.Wrap(err, "Writing YAML file content")
	}

	if err := outputW.Flush(); err != nil {
		return errors.Wrap(err, "Flushing output")
	}

	return nil
}
