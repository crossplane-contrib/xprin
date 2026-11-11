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

// Package interrupt handles the signals that stop a run: SIGINT (Ctrl-C) and SIGTERM.
package interrupt

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/crossplane-contrib/xprin/internal/utils"
)

// Mockable functions
//
//nolint:gochecknoglobals // Global variables for dependency injection in tests
var exit = os.Exit

// Notify returns a context that is canceled by the first SIGINT or SIGTERM, and a function that stops the handling.
// A second signal ends the process with exit code 130, without any cleanup.
func Notify(parent context.Context) (context.Context, context.CancelFunc) {
	return notify(parent, os.Interrupt, syscall.SIGTERM)
}

func notify(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	received := make(chan os.Signal, 2)
	signal.Notify(received, signals...)

	done := make(chan struct{})

	go func() {
		defer signal.Stop(received)

		select {
		case <-received:
			utils.WarningPrintf("Interrupted: finishing the test cases that run, and cleaning up. Press Ctrl-C again to quit now.\n")
			cancel()
		case <-done:
			return
		}

		select {
		case <-received:
			utils.WarningPrintf("Interrupted again: quitting without cleaning up.\n")
			exit(130)
		case <-done:
		}
	}()

	return ctx, func() {
		cancel()
		close(done)
	}
}
