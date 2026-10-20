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

// Package legacy05 is what xprin knows about the testsuite files in the format of up to 0.5, where a
// testcase that uses the outputs of others (`.Tests`) doesn't list them in `needs`: the dependencies are
// implicit, found by reading the templates.
package legacy05

import (
	"regexp"
	"slices"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/templates"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
)

var (
	// An action: what is between the placeholders of {{ and }}.
	action = regexp.MustCompile("(?s)" + regexp.QuoteMeta(placeholder.Open) + "(.*?)" + regexp.QuoteMeta(placeholder.Close))
	// Any use of .Tests.
	testsUse = regexp.MustCompile(`\.Tests\b`)
)

// hasImplicitNeeds tells if a testcase uses `.Tests` in the actions of its templates, as the runner
// renders them (the testcase without its hooks, and each hook on its own). It is also true for:
//   - a mention that is not a use (a comment or a string in an action), so that a use is never missed
//   - a testcase that can't be marshalled, which the runner fails anyway
func hasImplicitNeeds(testCase api.TestCase) bool {
	body, err := templates.MarshalWithoutHooks(testCase)
	if err != nil {
		// Can't be rendered, so the runner fails the testcase: assume that it uses them.
		return true
	}

	sources := []string{string(body)}
	for _, hook := range slices.Concat(testCase.Hooks.PreTest, testCase.Hooks.PostTest) {
		sources = append(sources, hook.Run)
	}

	for _, source := range sources {
		for _, act := range action.FindAllStringSubmatch(source, -1) {
			if testsUse.MatchString(act[1]) {
				return true
			}
		}
	}

	return false
}
