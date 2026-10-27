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
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

// The keywords of the required categories, set by the jsonschema_extras tags of internal/api.
const (
	keyMerged = "x-xprin-required-merged" // true: the property has to be in common or in the test case
	keyOneOf  = "x-xprin-required-one-of" // The sibling of the property: exactly one of the two has to be set
)

// noteRe matches the note that the xprinReflector adds at the end of a description.
var noteRe = regexp.MustCompile(` \((Required|Optional|One of)[^()]*\)$`)

// fieldInfo is what the tags of a field say.
type fieldInfo struct {
	owner    reflect.Type        // The struct type that has the field
	name     string              // The name in the schema, from the json tag
	required bool                // The reflector makes it required: the json tag has no omitempty or omitzero
	extras   map[string][]string // The jsonschema_extras tag, where a key that is repeated has all its values
}

// newFieldInfo reads the tags of a field of the struct type owner.
func newFieldInfo(owner reflect.Type, field reflect.StructField) fieldInfo {
	name, opts, _ := strings.Cut(field.Tag.Get("json"), ",")
	options := strings.Split(opts, ",")

	info := fieldInfo{
		owner:    owner,
		name:     name,
		required: !slices.Contains(options, "omitempty") && !slices.Contains(options, "omitzero"),
		extras:   map[string][]string{},
	}

	for pair := range strings.SplitSeq(field.Tag.Get("jsonschema_extras"), ",") {
		if k, v, ok := strings.Cut(pair, "="); ok {
			info.extras[k] = append(info.extras[k], v)
		}
	}

	return info
}

// sibling reads the tags of the field of the same struct that has the name in the schema.
func (info fieldInfo) sibling(name string) (fieldInfo, bool) {
	for field := range info.owner.Fields() {
		if other := newFieldInfo(info.owner, field); other.name == name {
			return other, true
		}
	}

	return fieldInfo{}, false
}

// note is the note that ends the description of the field, from its tags, or an error if they don't go together. The comment is
// the Go comment of the field.
func (info fieldInfo) note(comment string) (string, error) {
	if comment == "" {
		return "", errors.New("it has no comment")
	}

	if noteRe.MatchString(comment) {
		return "", errors.New("its comment ends with a note, schema-gen adds it")
	}

	for key := range info.extras {
		if strings.HasPrefix(key, "x-xprin-") && key != keyMerged && key != keyOneOf {
			return "", fmt.Errorf("unknown keyword %s", key)
		}
	}

	merged := false

	if values, ok := info.extras[keyMerged]; ok {
		if len(values) != 1 || (values[0] != "true" && values[0] != "false") {
			return "", fmt.Errorf("%s has to be true or false, not %s", keyMerged, strings.Join(values, ","))
		}

		merged = values[0] == "true"
	}

	oneOf := info.extras[keyOneOf]

	// The siblings are properties of the same type, and they name this one back
	for _, sib := range oneOf {
		other, ok := info.sibling(sib)
		if !ok {
			return "", fmt.Errorf("its sibling %q is not a property of the same type", sib)
		}

		if !slices.Contains(other.extras[keyOneOf], info.name) {
			return "", fmt.Errorf("its sibling %q does not name %q back", sib, info.name)
		}
	}

	return requirementNote(info.name, info.required, merged, oneOf)
}

// requirementNote is the note that ends the description of a property, or an error if what it says about it doesn't go together.
func requirementNote(name string, required, merged bool, oneOf []string) (string, error) {
	switch {
	case required && (merged || len(oneOf) > 0):
		return "", errors.New("it is required where it is, so it can't also be merged or one-of: it needs omitempty")
	case required:
		return "Required", nil
	case len(oneOf) > 0 && merged:
		return fmt.Sprintf("One of %s must be set, per testcase or in common", joinOr(append([]string{name}, oneOf...))), nil
	case len(oneOf) > 0:
		return fmt.Sprintf("One of %s must be set", joinOr(append([]string{name}, oneOf...))), nil
	case merged:
		return "Required per testcase or in common", nil
	default:
		return "Optional", nil
	}
}

// joinOr writes the names as "a, b or c".
func joinOr(names []string) string {
	if len(names) == 1 {
		return names[0]
	}

	return strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
}
