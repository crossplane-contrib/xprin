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
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// node is the part of a JSON schema node that the tables need.
type node struct {
	Ref                  string     `json:"$ref"`
	Type                 string     `json:"type"`
	Description          string     `json:"description"`
	Enum                 []any      `json:"enum"`
	MinItems             *int       `json:"minItems"`
	Items                *node      `json:"items"`
	AdditionalProperties valueType  `json:"additionalProperties"`
	Properties           properties `json:"properties"`
	Required             []string   `json:"required"`
	Merged               bool       `json:"x-xprin-required-merged"` //nolint:tagliatelle // A keyword of the JSON schema, set by the jsonschema_extras tags of internal/api.
	OneOf                stringList `json:"x-xprin-required-one-of"` //nolint:tagliatelle // Same: the siblings of a property.
}

type schema struct {
	Defs map[string]*node `json:"$defs"`
}

// properties are the properties of a node, in the order of the schema, which is the one of the Go types.
type properties struct {
	keys  []string         // In the order of the schema
	byKey map[string]*node // The property of each key
}

func (p *properties) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))

	if _, err := dec.Token(); err != nil { // The opening brace
		return err
	}

	p.byKey = map[string]*node{}

	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return err
		}

		var child node
		if err := dec.Decode(&child); err != nil {
			return fmt.Errorf("%v: %w", key, err)
		}

		k, _ := key.(string)

		p.keys = append(p.keys, k)
		p.byKey[k] = &child
	}

	return nil
}

// stringList is a string or a list of strings.
type stringList []string

func (l *stringList) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*l = []string{one}

		return nil
	}

	return json.Unmarshal(data, (*[]string)(l))
}

// valueType is the type of the values of a map. The same keyword can be a boolean, which has no type.
type valueType string

func (v *valueType) UnmarshalJSON(data []byte) error { //nolint:unparam // Needed by json.Unmarshaler, and a boolean is not an error.
	var values struct {
		Type string `json:"type"`
	}

	if json.Unmarshal(data, &values) == nil {
		*v = valueType(values.Type)
	}

	return nil
}

func defName(ref string) string {
	return strings.TrimPrefix(ref, "#/$defs/")
}
