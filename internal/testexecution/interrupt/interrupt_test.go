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

package interrupt

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

func send(t *testing.T) {
	t.Helper()

	if err := syscall.Kill(os.Getpid(), syscall.SIGUSR1); err != nil {
		t.Fatal(err)
	}
}

func TestNotify(t *testing.T) {
	t.Run("the first signal cancels the context and does not exit", func(t *testing.T) {
		exited := make(chan int, 1)

		defer func(old func(int)) { exit = old }(exit)

		exit = func(code int) { exited <- code }

		ctx, stop := notify(context.Background(), syscall.SIGUSR1)
		defer stop()

		send(t)

		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
			t.Fatal("the context was not canceled")
		}

		select {
		case code := <-exited:
			t.Fatalf("exited with %d on the first signal", code)
		case <-time.After(100 * time.Millisecond):
		}
	})

	t.Run("the second signal exits with 130", func(t *testing.T) {
		exited := make(chan int, 1)

		defer func(old func(int)) { exit = old }(exit)

		exit = func(code int) { exited <- code }

		ctx, stop := notify(context.Background(), syscall.SIGUSR1)
		defer stop()

		send(t)
		<-ctx.Done()
		send(t)

		select {
		case code := <-exited:
			if code != 130 {
				t.Fatalf("exit code = %d, want 130", code)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("did not exit")
		}
	})

	t.Run("stop cancels the context without a signal", func(t *testing.T) {
		ctx, stop := notify(context.Background(), syscall.SIGUSR1)
		stop()

		if ctx.Err() == nil {
			t.Fatal("the context is not canceled")
		}
	})
}
