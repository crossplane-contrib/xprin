/*
Copyright 2025 The Crossplane Authors.

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

package patchxr

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	schema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	structuraldefaulting "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/defaulting"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"

	apiextensionsv1 "github.com/crossplane/crossplane/apis/apiextensions/v1"
)

// hasPatchingFlags determines if any patching flags are provided.
func (c *Cmd) hasPatchingFlags() bool {
	return c.XRD != "" || c.AddConnectionSecret
}

// LoadXRD loads an XRD from a YAML file as an unstructured object.
func LoadXRD(fs afero.Fs, filePath string) (*apiextensionsv1.CompositeResourceDefinition, error) {
	y, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read XRD file")
	}

	xrd := &apiextensionsv1.CompositeResourceDefinition{}

	return xrd, errors.Wrap(yaml.Unmarshal(y, xrd), "cannot unmarshal XRD YAML")
}

// DefaultValuesFromXRD sets default values on the XR based on the XRD schema.
// Based on the `crossplane render --xrd` flag of Crossplane CLI v2
// https://github.com/crossplane/crossplane/blob/v2.0.2/cmd/crank/render/xrd.go#L13-L43
func DefaultValuesFromXRD(xr map[string]any, apiVersion string, xrd apiextensionsv1.CompositeResourceDefinition) error {
	var version *apiextensionsv1.CompositeResourceDefinitionVersion

	for _, vr := range xrd.Spec.Versions {
		checkAPIVersion := xrd.Spec.Group + "/" + vr.Name
		if checkAPIVersion == apiVersion {
			version = &vr
			break
		}
	}

	if version == nil {
		return errors.Errorf("the specified API version '%s' does not exist in the XRD", apiVersion)
	}

	if version.Schema == nil || len(version.Schema.OpenAPIV3Schema.Raw) == 0 {
		// No schema to apply defaults from
		return nil
	}

	// Parse the raw extension to get the JSONSchemaProps
	var schemaProps extv1.JSONSchemaProps
	if err := json.Unmarshal(version.Schema.OpenAPIV3Schema.Raw, &schemaProps); err != nil {
		return errors.Wrap(err, "failed to unmarshal OpenAPIV3Schema")
	}

	// Convert to internal types for structural schema
	var internalSchema apiextensions.JSONSchemaProps
	if err := extv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(&schemaProps, &internalSchema, nil); err != nil {
		return errors.Wrap(err, "failed to convert schema")
	}

	// Create structural schema for defaulting
	crdWithDefaults, err := schema.NewStructural(&internalSchema)
	if err != nil {
		return errors.Wrap(err, "failed to create structural schema")
	}

	// Apply defaults using Kubernetes structural defaulting (same as Crossplane)
	structuraldefaulting.Default(xr, crdWithDefaults)

	return nil
}

// AddConnectionSecret adds writeConnectionSecretToRef to the XR spec based on the provided connection secret parameters.
func AddConnectionSecret(xr *unstructured.Unstructured, connectionSecretName, connectionSecretNamespace string) error {
	xrPaved, err := fieldpath.PaveObject(xr)
	if err != nil {
		return errors.Wrap(err, "failed to pave XR object")
	}

	uid := uuid.New().String()
	if err := xrPaved.SetValue("metadata.uid", uid); err != nil {
		return errors.Wrap(err, "failed to set metadata.uid")
	}

	var (
		finalConnectionSecretName      string
		finalConnectionSecretNamespace string
	)

	// Use provided values or defaults

	if connectionSecretName != "" {
		finalConnectionSecretName = connectionSecretName
	} else {
		finalConnectionSecretName = uid
	}

	if connectionSecretNamespace != "" {
		finalConnectionSecretNamespace = connectionSecretNamespace
	} else {
		finalConnectionSecretNamespace = "default"
	}

	secretRef := map[string]interface{}{
		"name":      finalConnectionSecretName,
		"namespace": finalConnectionSecretNamespace,
	}
	if err := xrPaved.SetValue("spec.writeConnectionSecretToRef", secretRef); err != nil {
		return errors.Wrap(err, "failed to set writeConnectionSecretToRef")
	}

	// Update the XR object with the paved changes
	xr.Object = xrPaved.UnstructuredContent()

	return nil
}
