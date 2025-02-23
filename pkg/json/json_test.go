package json

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestCreateSchema(t *testing.T) {
	t.Parallel()
	tfPath := "../../test/modules"
	schemaPath := "../../test/expected"
	testCases := []string{
		"empty",
		"simple",
		"simple-types",
		"complex-types",
		"custom-validation",
		"ignore-variables",
	}
	for i := range testCases {
		name := testCases[i]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			expected, err := os.ReadFile(filepath.Join(schemaPath, name, "variables.json"))
			require.NoError(t, err)

			result, err := ExportVariables(filepath.Join(tfPath, name), ExportVariablesOptions{
				AllowEmpty:      true,
				DebugOut:        true,
				SuppressLogging: false,
				EscapeJSON:      false,
				Indent:          "\t",
				IgnoreVariables: []string{"ignored", "also_ignored"},
			})
			require.NoError(t, err)

			// marshal and unmarshal to ensure that the map is in the correct format
			buf, err := json.Marshal(result)
			require.NoError(t, err)

			var gotMap map[string]any
			err = json.Unmarshal(buf, &gotMap)
			require.NoError(t, err)

			var expectedMap map[string]any
			err = json.Unmarshal(expected, &expectedMap)
			require.NoError(t, err)

			if d := cmp.Diff(expectedMap, gotMap); d != "" {
				t.Errorf("Schema has incorrect value (-want,+got):\n%s", d)
			}
		})
	}
}
