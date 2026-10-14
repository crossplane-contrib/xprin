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

package graph

import (
	"fmt"
	"io"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
)

// WriteTree writes g to w as a tree, like tree(1), headed by title. Testcases that need nothing
// are at the top level; the others are listed under every testcase they need, expanded the
// first time and marked (*) afterwards.
func WriteTree(w io.Writer, title string, spec *api.TestSuiteSpec, g *Graph) error {
	children := make([][]int, len(spec.Tests))

	var roots []int

	for i := range spec.Tests {
		needs := g.deps[i]
		if len(needs) == 0 {
			roots = append(roots, i)
		}

		for _, j := range needs {
			children[j] = append(children[j], i)
		}
	}

	var (
		b        strings.Builder
		repeated bool
		seen     = make([]bool, len(spec.Tests))
		write    func(nodes []int, prefix string)
	)

	fmt.Fprintln(&b, title)

	write = func(nodes []int, prefix string) {
		for k, i := range nodes {
			connector, indent := "├── ", "│   "
			if k == len(nodes)-1 {
				connector, indent = "└── ", "    "
			}

			label := spec.Tests[i].Name
			if id := spec.Tests[i].ID; id != "" {
				label += " [" + id + "]"
			}

			if seen[i] {
				repeated = true

				fmt.Fprintf(&b, "%s%s%s (*)\n", prefix, connector, label)

				continue
			}

			seen[i] = true

			fmt.Fprintf(&b, "%s%s%s\n", prefix, connector, label)
			write(children[i], prefix+indent)
		}
	}

	write(roots, "")

	if repeated {
		b.WriteString("\n(*) needs more than one test case, listed above\n")
	}

	_, err := io.WriteString(w, b.String())

	return err
}
