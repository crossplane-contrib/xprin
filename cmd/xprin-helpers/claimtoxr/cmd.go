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

// Package claimtoxr implements the command for converting a Crossplane Claim to an XR (Composite Resource).
package claimtoxr

import (
	"bufio"
	"os"

	"github.com/alecthomas/kong"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"

	commonIO "github.com/crossplane/crossplane/v2/cmd/crank/beta/convert/io"
)

// Cmd arguments and flags for converting a Crossplane Claim to an XR (Composite Resource).
type Cmd struct {
	// Arguments.
	InputFile string `arg:"" default:"-" help:"The Claim YAML file to be converted. If not specified or '-', stdin will be used." optional:"" predictor:"file" type:"path"`

	// Flags.
	OutputFile string `help:"The file to write the generated XR YAML to. If not specified, stdout will be used."                                                      placeholder:"PATH" predictor:"file" short:"o" type:"path"`
	Name       string `help:"The name to use for the XR. If empty, defaults to the Claim's name (direct mode) or the Claim's name with a random suffix (non-direct)." placeholder:"NAME" type:"string"`
	Kind       string `help:"The kind to use for the XR. If not specified, 'X' will be prepended to the Claim's kind (e.g. Infra -> XInfra)."                         placeholder:"KIND" type:"string"`
	Direct     bool   `help:"Create a direct XR without Claim references and suffix."                                                                                 name:"direct"      negatable:""`
	GenUID     bool   `help:"Set a fresh random metadata.uid on the generated XR."                                                                                    name:"gen-uid"`

	fs afero.Fs
}

// Help returns help message for the convert claim-to-xr command.
func (c *Cmd) Help() string {
	return `
Convert a Crossplane Claim YAML file to a Crossplane Composite Resource (XR) format.

This command will:
  - Read the Claim from the provided YAML file
  - Create an XR with the same spec as the Claim
  - Set appropriate API version and kind for the XR
  - Set up the Claim reference in the XR (unless --direct is used)
  - Copy any composition selector

Examples:

  # Convert claim.yaml to XR format and write to stdout (kind will be 'X' + Claim's kind)
  xprin-helpers convert-claim-to-xr claim.yaml

  # Convert claim.yaml to XR format and write to xr.yaml
  xprin-helpers convert-claim-to-xr claim.yaml -o xr.yaml

  # Convert claim.yaml using an explicit XR name (overrides the default suffix or claim name)
  xprin-helpers convert-claim-to-xr claim.yaml --name my-xr

  # Convert claim.yaml to XR format with a specific kind
  xprin-helpers convert-claim-to-xr claim.yaml --kind MyCompositeResource

  # Convert claim.yaml to a directly created XR (no Claim references, no name suffix)
  xprin-helpers convert-claim-to-xr claim.yaml --direct

  # Convert claim.yaml and assign a fresh random metadata.uid to the XR
  xprin-helpers convert-claim-to-xr claim.yaml --gen-uid

  # Convert Claim from stdin to XR format
  cat claim.yaml | xprin-helpers convert-claim-to-xr -
`
}

// AfterApply implements kong.AfterApply.
func (c *Cmd) AfterApply() error {
	c.fs = afero.NewOsFs()
	return nil
}

// Run runs the claim-to-xr command.
func (c *Cmd) Run(k *kong.Context) error {
	claimData, err := commonIO.Read(c.fs, c.InputFile)
	if err != nil {
		return err
	}

	claim := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(claimData, claim); err != nil {
		return errors.Wrap(err, "Unmarshalling Error")
	}

	// Convert to XR
	xr, err := ConvertClaimToXR(claim, Options{
		Name:        c.Name,
		Kind:        c.Kind,
		Direct:      c.Direct,
		GenerateUID: c.GenUID,
	})
	if err != nil {
		return errors.Wrap(err, "failed to convert Claim to XR")
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
