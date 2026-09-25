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

package xpcli

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
)

// Detect probes the crossplane binary and returns a fully populated XPCLI,
// including the detected CLI version string.
func Detect(crossplaneBin string) XPCLI {
	var tier Tier

	switch {
	case detectV1CLI(crossplaneBin):
		tier = TierV1
	case detectLegacyCLI(crossplaneBin):
		tier = TierV2Legacy
	default:
		tier = TierV2
	}

	x := InitXPCLI(tier)
	x.Version = detectVersion(crossplaneBin)

	return x
}

func detectVersion(crossplaneBin string) string {
	probe := exec.Command(crossplaneBin, "version", "--client")

	var out bytes.Buffer

	probe.Stdout = &out
	probe.Stderr = io.Discard
	_ = probe.Run()

	// Output is e.g. "Client Version: v2.5.0"; take the last whitespace-separated token.
	fields := strings.Fields(out.String())
	if len(fields) == 0 {
		return ""
	}

	return fields[len(fields)-1]
}

func detectV1CLI(crossplaneBin string) bool {
	probe := exec.Command(crossplaneBin, "render", "--help")

	var out bytes.Buffer

	probe.Stdout = &out
	probe.Stderr = &out
	_ = probe.Run()

	return !strings.Contains(out.String(), "--xrd")
}

func detectLegacyCLI(crossplaneBin string) bool {
	probe := exec.Command(crossplaneBin, "resource", "validate", "--help")
	probe.Stdout = io.Discard
	probe.Stderr = io.Discard

	return probe.Run() != nil
}
