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
			expected, err := os.ReadFile(filepath.Join(expectedPath, name, "type-constraints.json"))
			require.NoError(t, err)

			varMap, err := reader.GetVarMap(filepath.Join(tfPath, name))
			if err != nil && !errors.Is(err, reader.ErrFilesNotFound) {
				t.Errorf("error reading tf files: %v", err)
			}

			var expectedMap map[string]interface{}
			err = json.Unmarshal(expected, &expectedMap)
			require.NoError(t, err)

			require.Equal(t, len(varMap), len(expectedMap))

			for key, val := range varMap {
				expectedVal, ok := expectedMap[key]
				if !ok {
					t.Errorf("Variable %s not found in expected map", key)
				}

				constraint, err := getTypeConstraint(val.Variable.Type)
				require.NoError(t, err)

				if d := cmp.Diff(expectedVal, constraint); d != "" {
					t.Errorf("Variable %s has incorrect type constraint (-want,+got):\n%s", key, d)
				}
			}
		})
	}
}
