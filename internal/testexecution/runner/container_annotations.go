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
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
	"github.com/crossplane-contrib/xprin/internal/testexecution/containers"
	"github.com/crossplane-contrib/xprin/internal/utils"
)

// warnUserSetContainerAnnotations tells, for the functions of the testsuite file, which Functions set annotations of
// their container (see containers.UserSetAnnotations), when container reuse is on. It looks at the functions of every
// test case before any of them runs, and tells about each file once, in one notice for the testsuite file.
// It goes away with the support of the annotations, in xprin 0.7.0.
func (r *Runner) warnUserSetContainerAnnotations() {
	if r.Containers == nil {
		return
	}

	seen := make(map[string]struct{})

	var found []containers.UserSetAnnotations

	for _, testCase := range r.testSuite.Spec.Tests {
		if r.testSuite.Spec.HasCommon() {
			testCase.MergeCommon(r.testSuite.Spec.Common)
		}

		functions, ok := r.functionsBeforeRunning(testCase)
		if !ok {
			continue
		}

		if _, ok := seen[functions]; ok {
			continue
		}

		seen[functions] = struct{}{}

		found = append(found, containers.FindUserSetAnnotations(r.fs, functions)...)
	}

	if len(found) == 0 {
		return
	}

	utils.WarningPrintf("%s\n", containers.Notice(r.testSuite.File))

	for _, f := range found {
		utils.WarningPrintf("%s\n", f.Line())
	}
}

// functionsBeforeRunning returns the path of the functions of the test case (a file or a directory), as it is when the
// test case runs. It is not ok when it can't be known before: the path is empty, or a template that can't be rendered
// without running the test cases it needs (`.Tests`).
func (r *Runner) functionsBeforeRunning(testCase api.TestCase) (string, bool) {
	functions := testCase.Inputs.Functions
	if functions == "" {
		return "", false
	}

	if strings.Contains(functions, placeholder.Open) {
		rendered, err := r.renderTemplate(placeholder.Restore(functions), newTemplateContext(r.Repositories, testCase.Inputs, nil, nil), "functions")
		if err != nil {
			return "", false
		}

		functions = rendered
	}

	functions, err := r.expandPathRelativeToTestSuiteFile(r.testSuite.File, functions)
	if err != nil {
		return "", false
	}

	return functions, true
}
