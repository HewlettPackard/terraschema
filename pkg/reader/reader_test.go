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
				t.Errorf("error reading tf files: %v", err)
			}

			for k, v := range varMap {
				if v.Required && v.DefaultAsString != nil {
					t.Errorf("Variable %s is required but has a default", k)
				}
				if !v.Required && v.DefaultAsString == nil {
					t.Errorf("Variable %s is not required but has no default", k)
				}
			}
		})
	}
}
