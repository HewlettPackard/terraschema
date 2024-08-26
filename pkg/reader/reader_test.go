// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package reader

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestGetVarMap_Required(t *testing.T) {
	t.Parallel()
	tfPath := "../../test/modules"
	testCases := []string{
		"empty",
		"simple",
		"simple-types",
		"complex-types",
		"custom-validation",
	}
	for i := range testCases {
		name := testCases[i]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			varMap, err := GetVarMap(filepath.Join(tfPath, name))
			if err != nil && !errors.Is(err, ErrFilesNotFound) {
				t.Errorf("Error reading tf files: %v", err)
			}

			for k, v := range varMap {
				if v.Required && v.Variable.Default != nil {
					t.Errorf("Variable %q is required but has a default", k)
				}
				if !v.Required && v.Variable.Default == nil {
					t.Errorf("Variable %q is not required but has no default", k)
				}
			}
		})
	}
}
