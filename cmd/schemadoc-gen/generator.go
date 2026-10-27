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

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	indexName   = "INDEX"
	legendName  = "LEGEND"
	startMarker = "<!-- DOCS_START_%s -->"
	endMarker   = "<!-- DOCS_END_%s -->"
)

// generator fills a document with the tables and the index of the types of a schema, and the legend of its emojis.
type generator struct {
	sections sectionList
	defs     map[string]*node
	doc      string
}

// newGenerator reads the schema, and checks that its types match the sections.
func newGenerator(rawSchema []byte, doc string) (*generator, error) {
	g := &generator{sections: newSections(), doc: doc}

	if err := g.parseSchema(rawSchema); err != nil {
		return nil, err
	}

	if err := g.checkSections(); err != nil {
		return nil, err
	}

	return g, nil
}

// parseSchema reads the types of the schema.
func (g *generator) parseSchema(rawSchema []byte) error {
	var s schema
	if err := json.Unmarshal(rawSchema, &s); err != nil {
		return fmt.Errorf("parse schema: %w", err)
	}

	g.defs = s.Defs

	return nil
}

// checkSections fails if a type of the schema has no section, so a new type can't go undocumented, or a section has no type.
func (g *generator) checkSections() error {
	covered := map[string]bool{}

	for _, sec := range g.sections {
		if _, ok := g.defs[sec.def]; !ok {
			return fmt.Errorf("section %s: no type %q in the schema", sec.name, sec.def)
		}

		covered[sec.def] = true
	}

	var missing []string

	for name := range g.defs {
		if !covered[name] {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("types without a section, add them to newSections: %s", strings.Join(missing, ", "))
	}

	return nil
}

// renderDoc returns the document with the content between each pair of markers replaced by the table of its section, the index and the legend.
func (g *generator) renderDoc() (string, error) {
	doc := g.doc

	for _, sec := range g.sections {
		var err error

		doc, err = fillMarkers(doc, sec.name, g.renderSection(sec))
		if err != nil {
			return "", err
		}
	}

	root := g.sections[0] // The first section is the one of the root type

	doc, err := fillMarkers(doc, indexName, g.renderIndex(root))
	if err != nil {
		return "", err
	}

	return fillMarkers(doc, legendName, renderLegend())
}

// fillMarkers replaces what is between the markers of the section.
func fillMarkers(doc, name, content string) (string, error) {
	start, end := fmt.Sprintf(startMarker, name), fmt.Sprintf(endMarker, name)

	i := strings.Index(doc, start)
	j := strings.Index(doc, end)

	if i < 0 || j < i {
		return "", fmt.Errorf("the document has no %s ... %s markers (in this order)", start, end)
	}

	return doc[:i+len(start)] + "\n\n" + content + "\n" + doc[j:], nil
}
