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

package config

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
)

// DetectVersion returns the client version string reported by the crossplane binary,
// or an empty string if the version cannot be determined.
func DetectVersion(crossplaneBin string) string {
	probe := exec.Command(crossplaneBin, "version", "--client")

	var out bytes.Buffer

	probe.Stdout = &out
	probe.Stderr = io.Discard
	_ = probe.Run()

	return strings.TrimSpace(out.String())
}

// DetectV1CLI returns true when the given crossplane binary is a v1 CLI
// by probing whether "crossplane render --help" mentions the --xrd flag.
// If --xrd is absent, the binary is a v1 CLI that does not support passing an XRD to render.
func DetectV1CLI(crossplaneBin string) bool {
	probe := exec.Command(crossplaneBin, "render", "--help")

	var out bytes.Buffer

	probe.Stdout = &out
	probe.Stderr = &out
	_ = probe.Run()

	return !strings.Contains(out.String(), "--xrd")
}

// DetectLegacyCLI returns true when the given crossplane binary is a legacy CLI (< v2.3.0)
// by probing whether "crossplane resource validate --help" succeeds.
// Success → new CLI (false); failure → legacy CLI (true).
func DetectLegacyCLI(crossplaneBin string) bool {
	probe := exec.Command(crossplaneBin, "resource", "validate", "--help")
	probe.Stdout = io.Discard
	probe.Stderr = io.Discard

	return probe.Run() != nil
}
