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

package claimtoxr

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apiserver/pkg/storage/names"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/fieldpath"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource/unstructured/composite"

	apiextensionsv1 "github.com/crossplane/crossplane/apis/v2/apiextensions/v1"
)

const (
	// Error messages.
	errNilInput        = "input is nil"
	errEmptyClaimYAML  = "invalid Claim YAML: parsed object is empty"
	errNoAPIVersion    = "Claim has no apiVersion"
	errParseAPIVersion = "failed to parse Claim APIVersion"
	errNoKind          = "Claim has no kind section"
	errNoSpecSection   = "Claim has no spec section"

	// Label keys.
	labelClaimName      = "crossplane.io/claim-name"
	labelClaimNamespace = "crossplane.io/claim-namespace"
	labelComposite      = "crossplane.io/composite"
)

// Options configures ConvertClaimToXR.
type Options struct {
	// Name is the XR name. Empty falls back to:
	//   - claim.Name when Direct is true
	//   - claim.Name with a random suffix when Direct is false
	// A non-empty Name overrides both fallbacks.
	Name string

	// Kind is the XR kind. Empty falls back to XRD.Spec.Names.Kind when XRD is set, or
	// "X" + claim.Kind otherwise.
	Kind string

	// XRD is the CompositeResourceDefinition that owns the Claim. When Kind is empty, its
	// Spec.Names.Kind is used instead of guessing "X" + claim.Kind, which isn't guaranteed to
	// match.
	XRD *apiextensionsv1.CompositeResourceDefinition

	// Direct controls XR linkage to the claim:
	//   - true:  no spec.claimRef; no claim-name/claim-namespace labels
	//   - false: spec.claimRef is set; claim-name/claim-namespace labels added
	Direct bool

	// GenerateUID, when true, sets metadata.uid to a fresh random UUID.
	GenerateUID bool
}

// ResolveKind returns the XR kind ConvertClaimToXR would use for a Claim of the given kind,
// following the precedence: explicit kind, then xrd's real kind, then "X" + claimKind. Exported so
// callers (e.g. golden-file name matching) can learn the target kind before conversion runs, not
// just as a side effect of it.
func ResolveKind(claimKind, kind string, xrd *apiextensionsv1.CompositeResourceDefinition) string {
	if kind != "" {
		return kind
	}

	if xrd != nil {
		return xrd.Spec.Names.Kind
	}

	return "X" + claimKind
}

// ConvertClaimToXR converts a Crossplane Claim to a Composite Resource (XR).
func ConvertClaimToXR(claim *unstructured.Unstructured, opts Options) (*composite.Unstructured, error) {
	if claim == nil {
		return nil, errors.New(errNilInput)
	}

	if claim.Object == nil {
		return nil, errors.New(errEmptyClaimYAML)
	}

	// Get Claim's properties
	claimName := claim.GetName()

	claimKind := claim.GetKind()
	if claimKind == "" {
		return nil, errors.New(errNoKind)
	}

	apiVersion := claim.GetAPIVersion()
	if apiVersion == "" {
		return nil, errors.New(errNoAPIVersion)
	}

	if _, err := schema.ParseGroupVersion(apiVersion); err != nil {
		return nil, errors.Wrap(err, errParseAPIVersion)
	}

	annotations := claim.GetAnnotations()

	labels := claim.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	claimSpec, ok := claim.Object["spec"].(map[string]any)
	if !ok || claimSpec == nil {
		return nil, errors.New(errNoSpecSection)
	}

	// Create a new XR and pave it for manipulation
	xr := composite.New()

	xrPaved, err := fieldpath.PaveObject(xr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pave object")
	}

	if err := xrPaved.SetString("apiVersion", apiVersion); err != nil {
		return nil, errors.Wrap(err, "failed to set apiVersion")
	}

	kind := ResolveKind(claimKind, opts.Kind, opts.XRD)

	if err := xrPaved.SetString("kind", kind); err != nil {
		return nil, errors.Wrap(err, "failed to set kind")
	}

	if len(annotations) > 0 {
		if err := xrPaved.SetValue("metadata.annotations", annotations); err != nil {
			return nil, errors.Wrap(err, "failed to set annotations")
		}
	}

	if err := xrPaved.SetValue("spec", claimSpec); err != nil {
		return nil, errors.Wrap(err, "failed to set spec")
	}

	xrName := claimName

	if !opts.Direct {
		xrName = names.SimpleNameGenerator.GenerateName(claimName + "-")
		labels[labelClaimName] = claim.GetName()

		labels[labelClaimNamespace] = claim.GetNamespace()
		if err := xrPaved.SetValue("spec.claimRef", map[string]any{
			"apiVersion": apiVersion,
			"kind":       claimKind,
			"name":       claimName,
			"namespace":  claim.GetNamespace(),
		}); err != nil {
			return nil, errors.Wrap(err, "failed to set claimRef")
		}
	}

	// Explicit Name overrides both Direct's claim-name default and the generated suffix.
	if opts.Name != "" {
		xrName = opts.Name
	}

	if err := xrPaved.SetString("metadata.name", xrName); err != nil {
		return nil, errors.Wrap(err, "failed to set name")
	}

	if len(labels) > 0 {
		delete(labels, labelComposite)

		if err := xrPaved.SetValue("metadata.labels", labels); err != nil {
			return nil, errors.Wrap(err, "failed to set labels")
		}
	}

	if opts.GenerateUID {
		xr.SetUID(uuid.NewUUID())
	}

	return xr, nil
}
