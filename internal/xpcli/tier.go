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

// Package xpcli provides detection and capability classification for the crossplane CLI binary.
package xpcli

const (
	tierV2       = "v2"
	tierV2Legacy = "v2legacy"
	tierV1       = "v1"
)

// Tier identifies a crossplane CLI capability tier by name and description.
// Its fields are exported, so two Tier values are comparable with ==.
type Tier struct {
	Name        string // "v1", "v2", or "v2legacy"
	Description string // human-readable, e.g. "v2+, current (>= v2.3.0)"
}

// String returns the canonical tier name.
func (t Tier) String() string { return t.Name }

// Predefined Tier sentinels — use == for comparison.
//
//nolint:gochecknoglobals // Struct types cannot be Go constants; these are effectively immutable sentinel values.
var (
	// TierV2 is a v2+ CLI at >= v2.3.0 (supports --xrd and --crossplane-version in render).
	TierV2 = Tier{Name: tierV2, Description: "v2+, current (>= v2.3.0)"}
	// TierV2Legacy is a v2+ CLI at < v2.3.0 (supports --xrd but not --crossplane-version; uses beta validate).
	TierV2Legacy = Tier{Name: tierV2Legacy, Description: "v2+, legacy (< v2.3.0)"}
	// TierV1 is a v1 CLI (no --xrd support; uses beta validate).
	TierV1 = Tier{Name: tierV1, Description: tierV1}
)

// XPCLI represents the detected crossplane CLI binary with its capability tier and derived settings.
// Use InitXPCLI to construct from a Tier sentinel, or Detect to probe a binary directly.
type XPCLI struct {
	Tier                  Tier
	validateSubcommand    string
	supportsRenderXRD     bool
	supportsRenderVersion bool
	// Version is the client version string reported by the binary (e.g. "v2.4.0").
	// Set only when constructed via Detect; empty for values from InitXPCLI.
	Version string
}

// InitXPCLI constructs an XPCLI from a Tier sentinel, deriving all capability fields from it.
func InitXPCLI(t Tier) XPCLI {
	x := XPCLI{Tier: t}
	setValidateSubcommand(&x)
	setSupportsRenderXRD(&x)
	setSupportsRenderVersion(&x)

	return x
}

func setValidateSubcommand(x *XPCLI) {
	if x.Tier.Name == tierV1 || x.Tier.Name == tierV2Legacy {
		x.validateSubcommand = "beta validate"
	} else {
		x.validateSubcommand = "resource validate"
	}
}

func setSupportsRenderXRD(x *XPCLI) {
	x.supportsRenderXRD = x.Tier.Name == tierV2 || x.Tier.Name == tierV2Legacy
}

func setSupportsRenderVersion(x *XPCLI) {
	x.supportsRenderVersion = x.Tier.Name == tierV2
}

// SupportsRenderXRD reports whether this CLI supports the --xrd flag in crossplane render.
func (x XPCLI) SupportsRenderXRD() bool { return x.supportsRenderXRD }

// SupportsRenderVersion reports whether this CLI supports --crossplane-version in crossplane render.
func (x XPCLI) SupportsRenderVersion() bool { return x.supportsRenderVersion }

// DefaultValidateSubcommand returns the validate subcommand string for this tier.
func (x XPCLI) DefaultValidateSubcommand() string { return x.validateSubcommand }
