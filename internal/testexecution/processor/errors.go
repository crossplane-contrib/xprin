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

package processor

import (
	"fmt"
	"os"
)

// reportError handles error reporting: print detailed error and FAIL status, returns the error for tracking.
func reportError(target, failureReason string, err error) error {
	errorMsg := fmt.Sprintf("%s in %s: %v", failureReason, target, err)
	fmt.Fprintf(os.Stderr, "# %s\n%s\n", target, errorMsg)
	fmt.Fprintf(os.Stderr, "FAIL\t%s\t[%s]\n", target, failureReason)

	return fmt.Errorf("%s", errorMsg)
}

// reportTestSuiteError handles error reporting for test suite files with detailed error message.
func reportTestSuiteError(testSuiteFile string, err error, failureReason string) error {
	errMsg := fmt.Sprintf("# %s\n%v", testSuiteFile, err)
	fmt.Fprintf(os.Stderr, "%s\n", errMsg)
	fmt.Fprintf(os.Stderr, "FAIL\t%s\t[%s]\n", testSuiteFile, failureReason)

	return fmt.Errorf("%s", errMsg)
}
