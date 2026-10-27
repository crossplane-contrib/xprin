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

// schemadoc-gen fills the property tables, the index and the legend of a document from the JSON schema written by schema-gen.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	schemaPath := flag.String("schema", "", "JSON schema written by schema-gen (required)")
	docPath := flag.String("doc", "", "Document to fill (required)")

	flag.Parse()

	if *schemaPath == "" || *docPath == "" {
		fmt.Fprintf(os.Stderr, "schemadoc-gen: -schema and -doc are required\n")
		os.Exit(1)
	}

	if err := run(*schemaPath, *docPath); err != nil {
		fmt.Fprintf(os.Stderr, "schemadoc-gen: %v\n", err)
		os.Exit(1)
	}
}

func run(schemaPath, docPath string) error {
	rawSchema, err := os.ReadFile(schemaPath) //nolint:gosec // A path given on the command line to a development tool.
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	rawDoc, err := os.ReadFile(docPath) //nolint:gosec // A path given on the command line to a development tool.
	if err != nil {
		return fmt.Errorf("read doc: %w", err)
	}

	doc := string(rawDoc)

	g, err := newGenerator(rawSchema, doc)
	if err != nil {
		return err
	}

	out, err := g.renderDoc()
	if err != nil {
		return err
	}

	if out == doc {
		return nil
	}

	if err := os.WriteFile(docPath, []byte(out), 0o644); err != nil { //nolint:gosec // A document, not a secret.
		return fmt.Errorf("write doc: %w", err)
	}

	return nil
}
