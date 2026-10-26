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

	"github.com/invopop/jsonschema"
)

// xprinReflector is the reflector of the library with what xprin adds: it puts a note at the end of the description of every
// property, as it looks up its comment, from the tags of the field.
type xprinReflector struct {
	*jsonschema.Reflector

	errs []error // The hook can only return the comment, so its errors are collected here
}

func newXprinReflector() *xprinReflector {
	r := &xprinReflector{Reflector: new(jsonschema.Reflector)}
	r.LookupComment = r.lookupComment

	return r
}

// lookupComment is the hook of the reflector, called for the comment of a type (with no field name) and of a field.
func (r *xprinReflector) lookupComment(t reflect.Type, fieldName string) string {
	if fieldName == "" {
		return "" // The comment of a type: the reflector looks it up itself
	}

	field, _ := t.FieldByName(fieldName)
	info := newFieldInfo(t, field)
	comment := r.CommentMap[t.PkgPath()+"."+t.Name()+"."+fieldName]

	note, err := info.note(comment)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s.%s: %w", t.Name(), info.name, err))

		return comment
	}

	return comment + " (" + note + ")"
}

// err returns the errors of the lookups, to call after the reflection.
func (r *xprinReflector) err() error {
	return errors.Join(r.errs...)
}
