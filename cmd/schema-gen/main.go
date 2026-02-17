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

// schema-gen generates a JSON schema from the Go types in internal/api using invopop/jsonschema. The output path is required.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/invopop/jsonschema"
)

func main() {
	outPath := flag.String("out", "", "Output path for the generated schema (required)")
	flag.Parse()

	if *outPath == "" {
		fmt.Fprintf(os.Stderr, "schema-gen: -out is required\n")
		os.Exit(1)
	}

	if err := run(*outPath); err != nil {
		fmt.Fprintf(os.Stderr, "schema-gen: %v\n", err)
		os.Exit(1)
	}
}

func run(outPath string) error {
	r := new(jsonschema.Reflector)
	if err := r.AddGoComments("github.com/crossplane-contrib/xprin", "internal/api"); err != nil {
		return fmt.Errorf("reading comments: %w", err)
	}

	schema := r.Reflect(&api.TestSuiteSpec{})
	if schema == nil {
		return errors.New("reflect returned nil")
	}

	schema.Title = "xprin"
	schema.Description = "xprin test suite files (xprin.yaml or *_xprin.yaml)"

	data, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	var full map[string]any
	if err := json.Unmarshal(data, &full); err != nil {
		return fmt.Errorf("unmarshal schema: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o750); err != nil {
		return fmt.Errorf("mkdir out: %w", err)
	}

	enc, err := json.MarshalIndent(full, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal indent: %w", err)
	}

	if err := os.WriteFile(outPath, append(enc, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	return nil
}
