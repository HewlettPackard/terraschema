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

func TestExpressionToJSONObject_Default(t *testing.T) {
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
			expected, err := os.ReadFile(filepath.Join(expectedPath, name, "defaults.json"))
			require.NoError(t, err)
			var expectedMap map[string]any
			err = json.Unmarshal(expected, &expectedMap)
			require.NoError(t, err)

			defaults := make(map[string]any)

			varMap, err := GetVarMap(filepath.Join(tfPath, name), true)
			if err != nil && !errors.Is(err, ErrFilesNotFound) {
				t.Errorf("error reading tf files: %v", err)
			}

			for key, val := range varMap {
				if val.Variable.Default == nil {
					continue
				}

				defaults[key], err = ExpressionToJSONObject(val.Variable.Default)
				require.NoError(t, err)
			}

			if len(defaults) != len(expectedMap) {
				t.Errorf("Expected %d variables with defaults, got %d", len(expectedMap), len(varMap))
			}

			for key, val := range defaults {
				expectedVal, ok := expectedMap[key]
				if !ok {
					t.Errorf("Variable %q not found in expected map", key)
				}

				if d := cmp.Diff(expectedVal, val); d != "" {
					t.Errorf("Variable %q has incorrect default (-want,+got):\n%s", key, d)
				}
			}
		})
	}
}
