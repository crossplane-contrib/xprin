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

package test

import (
	"fmt"
	"runtime"
	"strconv"

	"github.com/alecthomas/kong"
)

// Parallelism is the value of --parallel: the number of test cases to run at the same time, 1 or
// more. The flag without a number (--parallel) is the number of CPUs, as Go sees them, so it
// respects the CPU limit of a container, which is what you want in CI. 0 is the value when the flag
// is not given, and it is the same as 1: the test cases run one after the other.
type Parallelism int

// IsBool implements kong.BoolMapperValue: the flag can be given without a value.
func (p *Parallelism) IsBool() bool { return true }

// Decode implements kong.MapperValue. The flag is followed by its number, as --parallel=4, or as
// --parallel 4 or -p4 when the next argument is a number (like make -j); in any other case it is alone.
// So a target that is a number has to be written so that it is not one, such as ./4.
func (p *Parallelism) Decode(ctx *kong.DecodeContext) error {
	next := ctx.Scan.Peek()

	switch {
	case next.Type == kong.FlagValueToken:
	case next.Type == kong.UntypedToken && isNumber(fmt.Sprint(next.Value)):
	case next.Type == kong.ShortFlagTailToken && isNumber(fmt.Sprint(next.Value)):
	default:
		*p = Parallelism(runtime.GOMAXPROCS(0))

		return nil
	}

	n, err := parseParallelism(fmt.Sprint(ctx.Scan.Pop().Value))
	if err != nil {
		return err
	}

	*p = Parallelism(n)

	return nil
}

func isNumber(value string) bool {
	_, err := strconv.Atoi(value)

	return err == nil
}

func parseParallelism(value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("expected a number of test cases greater than 0, got %q", value)
	}

	return n, nil
}
