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
	"reflect"
	"testing"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// The types below have tags that don't go together. reflectWithComments gives their fields the comments that the reflector reads from the source.

type mergedNotOmitempty struct {
	X string `json:"x" jsonschema_extras:"x-xprin-required-merged=true"`
}

// unknownKeyword has a keyword that does not exist (not a misspelling of one: the linter would correct that).
type unknownKeyword struct {
	X string `json:"x,omitempty" jsonschema_extras:"x-xprin-optional=true"`
}

type notABoolean struct {
	X string `json:"x,omitempty" jsonschema_extras:"x-xprin-required-merged=yes"`
}

type plain struct {
	X string `json:"x,omitempty"`
}

type oneOfWithoutMerged struct {
	X string `json:"x,omitempty" jsonschema_extras:"x-xprin-required-one-of=y"`
	Y string `json:"y,omitempty" jsonschema_extras:"x-xprin-required-one-of=x"`
}

type alternativeMissing struct {
	X string `json:"x,omitempty" jsonschema_extras:"x-xprin-required-merged=true,x-xprin-required-one-of=zzz"`
}

type alternativeDoesNotNameBack struct {
	X string `json:"x,omitempty" jsonschema_extras:"x-xprin-required-merged=true,x-xprin-required-one-of=y"`
	Y string `json:"y,omitempty" jsonschema_extras:"x-xprin-required-merged=true"`
}

// reflectWithComments reflects v as the tool does, with the comments of its fields, and returns the schema and the errors.
func reflectWithComments(v any, comments map[string]string) (*jsonschema.Schema, error) {
	typ := reflect.TypeOf(v)

	r := newXprinReflector()
	r.CommentMap = map[string]string{}

	for field, comment := range comments {
		r.CommentMap[typ.PkgPath()+"."+typ.Name()+"."+field] = comment
	}

	schema := r.Reflect(v)

	return schema, r.err()
}

func TestXprinReflector_errors(t *testing.T) {
	x := map[string]string{"X": "X"}

	failures := map[string]struct {
		v        any
		comments map[string]string
		wantErr  string
	}{
		"a merged property without omitempty": {mergedNotOmitempty{}, x, "needs omitempty"},
		"an unknown keyword":                  {unknownKeyword{}, x, "unknown keyword x-xprin-optional"},
		"a merged that is not a boolean":      {notABoolean{}, x, keyMerged + " has to be true or false"},
		"a one-of without merged":             {oneOfWithoutMerged{}, map[string]string{"X": "X", "Y": "Y"}, keyOneOf + " needs " + keyMerged},
		"an alternative that doesn't exist":   {alternativeMissing{}, x, `its alternative "zzz" is not a property`},
		"an alternative that doesn't name it": {alternativeDoesNotNameBack{}, map[string]string{"X": "X", "Y": "Y"}, `does not name "x" back`},
		"a property without a comment":        {plain{}, nil, "it has no comment"},
		"a note written in the comment":       {plain{}, map[string]string{"X": "X (Optional)"}, "schema-gen adds it"},
	}

	for name, tt := range failures {
		t.Run(name, func(t *testing.T) {
			_, err := reflectWithComments(tt.v, tt.comments)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
