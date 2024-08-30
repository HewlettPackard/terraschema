// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package reader

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestGetTypeConstraint(t *testing.T) {
	t.Parallel()
	tfPath := "../../test/modules"
	expectedPath := "../../test/expected/"
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
			expected, err := os.ReadFile(filepath.Join(expectedPath, name, "variables.json"))
			require.NoError(t, err)

			varMap, err := GetVarMap(filepath.Join(tfPath, name), true)
			if err != nil && !errors.Is(err, ErrFilesNotFound) {
				t.Errorf("Error reading tf files: %v", err)
			}

			var expectedMap map[string]any
			err = json.Unmarshal(expected, &expectedMap)
			require.NoError(t, err)

			require.Equal(t, len(varMap), len(expectedMap))

			for key, val := range varMap {
				expectedVal, ok := expectedMap[key].(map[string]any)["type"]
				if !ok {
					t.Errorf("Variable %q not found in expected map", key)
				}

				constraint, err := GetTypeConstraint(val.Variable.Type)
				require.NoError(t, err)

				if d := cmp.Diff(expectedVal, constraint); d != "" {
					t.Errorf("Variable %q has incorrect type constraint (-want,+got):\n%s", key, d)
				}
			}
		})
	}
}
