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

package main

// section is a part of the document: a type of the schema, with its table between its markers, under the heading that has this anchor.
type section struct {
	name   string // Suffix of the markers: DOCS_START_<name>
	title  string // What the links to this section say
	anchor string // Anchor of its heading in the document, and the prefix of the ids of its rows
	def    string // Name of the type in $defs
}

// sectionList is the sections of the document, in order.
type sectionList []section

// newSections returns the sections of the document. Every type in $defs has one. The first one is the root type: renderDoc builds the index from it.
func newSections() sectionList {
	return sectionList{
		{"ROOT", "root", "root", "TestSuiteSpec"}, // Its table is under ## Reference, so "root" is only the prefix of its rows
		{"COMMON", "common", "common", "Common"},
		{"TESTS", "tests[*]", "tests", "TestCase"},
		{"INPUTS", "inputs", "inputs", "Inputs"},
		{"PATCHES", "patches", "patches", "Patches"},
		{"HOOKS", "hooks", "hooks", "Hooks"},
		{"HOOK", "hook", "hookspre-test-and-hookspost-test", "Hook"},
		{"ASSERTIONS", "assertions", "assertions", "Assertions"},
		{"ASSERTIONS_XPRIN", "assertions.xprin[*]", "assertionsxprin", "AssertionXprin"},
		{"ASSERTIONS_GOLDEN", "golden-file assertion", "assertionsdiff-and-assertionsdyff", "AssertionGoldenFile"},
	}
}

// find returns the section of a type.
func (l sectionList) find(def string) (section, bool) {
	for _, sec := range l {
		if sec.def == def {
			return sec, true
		}
	}

	return section{}, false
}
