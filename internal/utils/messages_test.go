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

package utils

import (
	"strings"
	"testing"

	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
)

func TestWarningPrintf(t *testing.T) {
	out := unittestsUtils.CaptureStderr(func() {
		WarningPrintf("always: %d %s", 7, "baz")
	})
	if !strings.Contains(out, "always: 7 baz") {
		t.Errorf("WarningPrintf did not print expected output, got: %q", out)
	}

	if !strings.HasPrefix(out, "WARNING: ") {
		t.Errorf("WarningPrintf output did not start with 'WARNING: ', got: %q", out)
	}
}

func TestDebugPrintf(t *testing.T) {
	out := unittestsUtils.CaptureStderr(func() {
		DebugPrintf("always: %d %s", 7, "baz")
	})
	if !strings.Contains(out, "always: 7 baz") {
		t.Errorf("DebugPrintf did not print expected output, got: %q", out)
	}

	if !strings.HasPrefix(out, "DEBUG: ") {
		t.Errorf("WarningPrintf output did not start with 'DEBUG: ', got: %q", out)
	}
}
