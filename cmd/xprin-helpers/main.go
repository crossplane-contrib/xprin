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

// Package main is the main package for the xprin-helpers tool.
package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/crossplane-contrib/xprin/cmd/xprin-helpers/claimtoxr"
	"github.com/crossplane-contrib/xprin/cmd/xprin-helpers/patchxr"
	"github.com/crossplane-contrib/xprin/cmd/xprin-helpers/version"
)

// CLI represents the command-line interface structure.
type CLI struct {
	ConvertClaimToXR claimtoxr.Cmd `cmd:"convert-claim-to-xr" help:"Convert a Crossplane Claim to an XR (Composite Resource)."`
	PatchXR          patchxr.Cmd   `cmd:""                    help:"Patch a Crossplane XR (Composite Resource) with additional configurations."`
	Version          version.Cmd   `cmd:""                    help:"Print the version of xprin-helpers"`
}

func main() {
	var cli CLI

	ctx := kong.Parse(&cli,
		kong.Name("xprin-helpers"),
		kong.Description("Crossplane helper utilities for converting and patching resources."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Summary: true,
		}),
	)

	// Run the selected command
	err := ctx.Run()
	if err != nil {
		os.Exit(1)
	}
}
