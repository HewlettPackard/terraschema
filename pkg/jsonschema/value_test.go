package jsonschema

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/AislingHPE/TerraSchema/pkg/reader"
)

func TestExpressionToJSONObject(t *testing.T) {
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

			varMap, err := reader.GetVarMap(filepath.Join(tfPath, name))
			if err != nil && !errors.Is(err, reader.ErrFilesNotFound) {
				t.Errorf("error reading tf files: %v", err)
			}

			for key, val := range varMap {
				if val.DefaultAsString == nil {
					continue
				}

				defaultValue, err := expressionToJSONObject(val.Variable.Default)
				require.NoError(t, err)

				defaultJSON, err := json.Marshal(defaultValue)
				require.NoError(t, err)

				var defaultUnmarshaled any
				err = json.Unmarshal(defaultJSON, &defaultUnmarshaled)
				require.NoError(t, err)

				defaults[key] = defaultUnmarshaled
			}

			if len(defaults) != len(expectedMap) {
				t.Errorf("Expected %d variables with defaults, got %d", len(expectedMap), len(varMap))
			}

			for key, val := range defaults {
				expectedVal, ok := expectedMap[key]
				if !ok {
					t.Errorf("Variable %s not found in expected map", key)
				}

				if d := cmp.Diff(expectedVal, val); d != "" {
					t.Errorf("Variable %s has incorrect default (-want,+got):\n%s", key, d)
				}
			}
		})
	}
}
