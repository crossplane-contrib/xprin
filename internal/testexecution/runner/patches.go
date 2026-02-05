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

package runner

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/crossplane-contrib/xprin/cmd/xprin-helpers/claimtoxr"
	"github.com/crossplane-contrib/xprin/cmd/xprin-helpers/patchxr"
	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	apiextensionsv1 "github.com/crossplane/crossplane/apis/apiextensions/v1"
)

// copyInput copies a file or directory to the inputs directory organized by type and returns the destination path.
func (r *Runner) copyInput(src, inputType string) (string, error) {
	// Create subdirectory for the input type
	typeDir := filepath.Join(r.inputsDir, inputType)
	if err := r.fs.MkdirAll(typeDir, 0o750); err != nil {
		return "", fmt.Errorf("failed to create %s directory: %w", inputType, err)
	}

	// Copy to typeDir with original filename
	dest := filepath.Join(typeDir, filepath.Base(src))
	if err := r.copy(src, dest); err != nil {
		return "", fmt.Errorf("failed to copy %s: %w", inputType, err)
	}

	if r.Debug {
		utils.DebugPrintf("Copied %s to: %s\n", inputType, dest)
	}

	return dest, nil
}

// copyToPath copies a file or directory to the given destination path, creating parent directories as needed.
// Use this when the destination path must be chosen by the caller (e.g. to avoid overwriting same-named files).
func (r *Runner) copyToPath(src, dest string) (string, error) {
	if err := r.fs.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return "", fmt.Errorf("failed to create directory for %s: %w", dest, err)
	}

	if err := r.copy(src, dest); err != nil {
		return "", fmt.Errorf("failed to copy to %s: %w", dest, err)
	}

	if r.Debug {
		utils.DebugPrintf("Copied to: %s\n", dest)
	}

	return dest, nil
}

// uniqueBaseNamesForPaths returns a unique base filename for each path so that paths with the
// same base name (e.g. aws/xrd.yaml and gcp/xrd.yaml) get distinct names (xrd.yaml, xrd_1.yaml).
// The returned slice has the same length and order as paths. Returns nil if paths is nil.
func uniqueBaseNamesForPaths(paths []string) []string {
	if paths == nil {
		return nil
	}

	names := make([]string, len(paths))
	nameCount := make(map[string]int)

	for i, p := range paths {
		base := filepath.Base(p)
		ext := filepath.Ext(base)
		nameWithoutExt := strings.TrimSuffix(base, ext)
		n := nameCount[nameWithoutExt]

		nameCount[nameWithoutExt]++
		if n > 0 {
			base = fmt.Sprintf("%s_%d%s", nameWithoutExt, n, ext)
		}

		names[i] = base
	}

	return names
}

// convertClaimToXR converts a Claim to XR using the convert-claim-to-xr library.
func (r *Runner) convertClaimToXR(claimPath, outputPath string) (string, error) {
	claimData, err := afero.ReadFile(r.fs, claimPath)
	if err != nil {
		return "", fmt.Errorf("failed to read claim file: %w", err)
	}

	claim := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(claimData, claim); err != nil {
		return "", fmt.Errorf("failed to parse claim YAML: %w", err)
	}

	if r.Debug {
		utils.DebugPrintf("Converting Claim to XR\n")
	}

	xr, err := claimtoxr.ConvertClaimToXR(claim, "", false)
	if err != nil {
		return "", fmt.Errorf("failed to convert claim to XR: %w", err)
	}

	output, err := yaml.Marshal(xr)
	if err != nil {
		return "", fmt.Errorf("failed to marshal XR to YAML: %w", err)
	}

	output = append([]byte("---\n"), output...)

	xrPath := filepath.Join(outputPath, "xr.yaml")
	if err := afero.WriteFile(r.fs, xrPath, output, 0o600); err != nil {
		return "", fmt.Errorf("failed to write XR to temporary file: %w", err)
	}

	if r.Debug {
		utils.DebugPrintf("Wrote converted XR to temporary file: %s\n", xrPath)
	}

	return xrPath, nil
}

// patchXR applies XRD defaults and connection secret patches to an XR using the patch-xr library.
func (r *Runner) patchXR(xrPath, outputPath string, patches api.Patches) (string, error) {
	// Check connection secret configuration first
	if err := patches.CheckConnectionSecret(); err != nil {
		return "", err
	}

	xrData, err := afero.ReadFile(r.fs, xrPath)
	if err != nil {
		return "", fmt.Errorf("failed to read XR file: %w", err)
	}

	xr := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(xrData, xr); err != nil {
		return "", fmt.Errorf("failed to parse XR YAML: %w", err)
	}

	// Apply XRD defaults if XRD file is provided
	if patches.XRD != "" {
		if r.Debug {
			utils.DebugPrintf("Patching XR: Applying XRD defaults\n")
		}

		// Read XRD file
		xrdData, err := afero.ReadFile(r.fs, patches.XRD)
		if err != nil {
			return "", fmt.Errorf("failed to read XRD file: %w", err)
		}

		xrd := &apiextensionsv1.CompositeResourceDefinition{}
		if err := yaml.Unmarshal(xrdData, xrd); err != nil {
			return "", fmt.Errorf("failed to parse XRD YAML: %w", err)
		}

		// Apply defaults using the library function
		if err := patchxr.DefaultValuesFromXRD(xr.UnstructuredContent(), xr.GetAPIVersion(), *xrd); err != nil {
			return "", fmt.Errorf("failed to apply XRD defaults: %w", err)
		}
	}

	// Add connection secret if requested
	if patches.HasConnectionSecret() {
		if r.Debug {
			utils.DebugPrintf("Patching XR: Adding connection secret\n")
		}

		if err := patchxr.AddConnectionSecret(xr, patches.ConnectionSecretName, patches.ConnectionSecretNamespace); err != nil {
			return "", fmt.Errorf("failed to add connection secret: %w", err)
		}
	}

	output, err := yaml.Marshal(xr)
	if err != nil {
		return "", fmt.Errorf("failed to marshal XR to YAML: %w", err)
	}

	output = append([]byte("---\n"), output...)

	patchedXRPath := filepath.Join(outputPath, "patched-xr.yaml")
	if err := afero.WriteFile(r.fs, patchedXRPath, output, 0o600); err != nil {
		return "", fmt.Errorf("failed to write patched XR to temporary file: %w", err)
	}

	return patchedXRPath, nil
}
