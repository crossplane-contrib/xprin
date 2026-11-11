//go:build !windows

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

package runner

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"

	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestNew_RunsCommandsInAProcessGroupOfTheirOwn(t *testing.T) {
	// Ctrl-C goes to the process group of xprin: a command that leads a group of its own does not get it
	out, err := New(&testexecutionOptions.Options{}).runCommand(t.TempDir(), "sh", "-c", `ps -o pgid= -p $$; echo $$`)
	require.NoError(t, err)

	fields := strings.Fields(string(out))
	require.Len(t, fields, 2, string(out))
	assert.Equal(t, fields[1], fields[0], "the command must lead its process group")

	ours, err := syscall.Getpgid(os.Getpid())
	require.NoError(t, err)
	assert.NotEqual(t, strconv.Itoa(ours), fields[0])
}
