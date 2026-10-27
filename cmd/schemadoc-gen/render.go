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
	"fmt"
	"slices"
	"strings"
)

// The emojis of the requirement column, explained by renderLegend.
const (
	emojiRequired = "✅"
	emojiMerged   = "🧩"
	emojiOneOf    = "🔀"
	emojiOptional = "🔹"
)

// renderLegend renders the table that explains the emojis of the requirement column.
func renderLegend() string {
	categories := []struct{ emoji, name, meaning string }{
		{emojiRequired, "required", "The property has to be here."},
		{emojiMerged, "merged", "The property has to be in `common` or in the test case, in the merged result: it can't be in neither. If it is in both, the one of the test case prevails."},
		{emojiOneOf, "one-of", "Exactly one of the property and the one it is exclusive with, named in its description, has to be set."},
		{emojiOptional, "optional", "The property can be missing."},
	}

	var b strings.Builder

	b.WriteString("| | Name | Meaning |\n")
	b.WriteString("| :-: | :--- | :------ |\n")

	for _, c := range categories {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", c.emoji, c.name, c.meaning)
	}

	return b.String()
}

// renderIndex lists the path of every property, from the root of the file, each linking to its row (or to its section, when its type has one).
func (g *generator) renderIndex(sec section) string {
	var b strings.Builder

	g.writeIndexEntries(&b, sec, "")

	return b.String()
}

// writeIndexEntries writes the properties of the section sec: shown is their path in the file.
func (g *generator) writeIndexEntries(b *strings.Builder, sec section, shown string) {
	def := g.defs[sec.def]

	for _, key := range def.Properties.keys {
		p := def.Properties.byKey[key]

		// A list has [*] after its name, in the path of the list and of its elements
		path := shown + key
		ref := p.Ref

		if p.Items != nil {
			path += "[*]"
			ref = p.Items.Ref
		}

		// A property whose type (or the items of it) has a section links to the section, and its properties follow
		inner, hasSection := g.sections.find(defName(ref))

		link := sec.anchor + "." + key
		if hasSection {
			link = inner.anchor
		}

		fmt.Fprintf(b, "- [`%s`](#%s)\n", path, link)

		if hasSection {
			g.writeIndexEntries(b, inner, path+".")
		}
	}
}

// renderSection renders the table of a section: a row for each property of its type.
func (g *generator) renderSection(sec section) string {
	def := g.defs[sec.def]

	var b strings.Builder

	b.WriteString("| Property | Type | Requirement | Description |\n")
	b.WriteString("| :------- | :--- | :---------: | :---------- |\n")

	for _, key := range def.Properties.keys {
		b.WriteString(g.newRow(sec, def, key).String())
	}

	return b.String()
}

// row is a line of the table of a section: the cells of a property.
type row struct {
	id          string // The anchor of the row: the one of its section, then the name
	name        string
	dataType    string
	requirement string
	description string
}

// newRow makes the row of the property of the key, which is one of def.
func (g *generator) newRow(sec section, def *node, key string) row {
	p := def.Properties.byKey[key]

	return row{
		id:          sec.anchor + "." + key,
		name:        key,
		dataType:    g.renderType(p),
		requirement: def.renderRequirement(key),
		description: p.renderDescription(),
	}
}

// String returns the row as a line of the markdown table.
func (r row) String() string {
	return fmt.Sprintf("| <a id=\"%s\"></a>`%s` | %s | %s | %s |\n", r.id, r.name, r.dataType, r.requirement, r.description)
}

// renderRequirement returns the emojis of the requirement column of the property of the key: one for each keyword, or else if it is required or optional.
func (n *node) renderRequirement(key string) string {
	p := n.Properties.byKey[key]

	var emojis string

	if p.Merged {
		emojis += emojiMerged
	}

	if len(p.OneOf) > 0 {
		emojis += emojiOneOf
	}

	switch {
	case emojis != "":
		return emojis
	case slices.Contains(n.Required, key):
		return emojiRequired
	default:
		return emojiOptional
	}
}

// renderType renders the type of a property, linking to the section of its type.
func (g *generator) renderType(p *node) string {
	switch {
	case p.Ref != "":
		return g.renderTypeRef(p.Ref)
	case p.Type == "array":
		t := "list of " + g.renderType(p.Items)
		if p.MinItems != nil {
			t += fmt.Sprintf(" (at least %d)", *p.MinItems)
		}

		return t
	case p.Type == "object":
		if p.AdditionalProperties != "" {
			return "map of `" + string(p.AdditionalProperties) + "`"
		}

		return "`object`"
	case p.Type == "":
		return "any"
	default:
		return "`" + p.Type + "`"
	}
}

// renderTypeRef renders a link to the section of the type that the ref points to.
func (g *generator) renderTypeRef(ref string) string {
	// Every type of the schema has a section, checkSections makes sure of it, so the lookup can't fail
	sec, _ := g.sections.find(defName(ref))

	return fmt.Sprintf("[`%s`](#%s)", sec.title, sec.anchor)
}

// renderDescription returns the description of a property, without the note that the requirement column says, and with its siblings and allowed values.
func (n *node) renderDescription() string {
	// schema-gen ends the description of every property with its note, in parentheses, so there is always one to cut: it fails before this tool if not
	d := n.Description[:strings.LastIndex(n.Description, " (")]
	d = strings.ReplaceAll(d, "|", "\\|")

	if len(n.OneOf) > 0 {
		d += ". Exclusive with: `" + strings.Join(n.OneOf, "`, `") + "`"
	}

	if len(n.Enum) > 0 {
		vals := make([]string, 0, len(n.Enum))
		for _, v := range n.Enum {
			vals = append(vals, fmt.Sprintf("`%v`", v))
		}

		d += ". Allowed values: " + strings.Join(vals, ", ")
	}

	return d
}
