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
// implicit, found by reading the templates. It is what the converter of those files uses.
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
	// index .Tests "<id>", or with backticks: the way to use an ID that is not a field name (a hyphen).
	indexedID = regexp.MustCompile("\\bindex\\s+\\.Tests\\s+(?:\"([^\"]*)\"|`([^`]*)`)")
	// .Tests.<id>.
	dottedID = regexp.MustCompile(`\.Tests\.([A-Za-z0-9_]+)`)
	// Any use of .Tests.
	testsUse = regexp.MustCompile(`\.Tests\b`)
)

// FindImplicitNeeds finds what a testcase needs without listing it, reading its templates as the runner renders
// them (the testcase without its hooks as one template, and each hook command on its own). It returns:
//   - ids: the testcases whose outputs it uses through `.Tests`
//   - needsAll: true when it needs all of them, because it uses `.Tests` in a way that can't be traced to an ID
//     (`range .Tests`, `index .Tests $id`), or because it can't be marshalled (the runner fails it anyway)
//
// Pass the testcase with the common configuration merged in.
func FindImplicitNeeds(testCase api.TestCase) (ids []string, needsAll bool) {
	body, err := templates.MarshalWithoutHooks(testCase)
	if err != nil {
		// Can't be rendered, so the runner fails the testcase: what it needs is unknown.
		return nil, true
	}

	sources := []string{string(body)}
	for _, hook := range slices.Concat(testCase.Hooks.PreTest, testCase.Hooks.PostTest) {
		sources = append(sources, hook.Run)
	}

	for _, source := range sources {
		found, all := findImplicitNeedsInString(source)
		ids = append(ids, found...)
		needsAll = needsAll || all
	}

	return ids, needsAll
}

// findImplicitNeedsInString is FindImplicitNeeds for one template, as the loader stores it (with placeholders):
//   - it reads the actions only, not the text around them
//   - it takes out the uses of `.Tests` that name an ID: a `.Tests` left is not understood, so needsAll is true
//   - a mention that is not a use (a comment or a string of an action) counts too, so that a use is never missed
func findImplicitNeedsInString(s string) (ids []string, needsAll bool) {
	for _, act := range action.FindAllStringSubmatch(s, -1) {
		body := act[1]

		for _, match := range indexedID.FindAllStringSubmatch(body, -1) {
			ids = append(ids, match[1]+match[2])
		}

		body = indexedID.ReplaceAllString(body, " ")

		for _, match := range dottedID.FindAllStringSubmatch(body, -1) {
			ids = append(ids, match[1])
		}

		body = dottedID.ReplaceAllString(body, " ")

		if testsUse.MatchString(body) {
			needsAll = true
		}
	}

	return ids, needsAll
}
