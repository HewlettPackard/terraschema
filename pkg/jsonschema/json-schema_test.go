// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package jsonschema

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/santhosh-tekuri/jsonschema/v5"
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
			expected, err := os.ReadFile(filepath.Join(schemaPath, name, "schema.json"))
			require.NoError(t, err)

			result, err := CreateSchema(filepath.Join(tfPath, name), CreateSchemaOptions{
				RequireAll:                false,
				AllowAdditionalProperties: true,
				AllowEmpty:                true,
				NullableAll:               false,
				IgnoreVariables:           []string{"ignored", "also_ignored"},
			})
			require.NoError(t, err)

			var expectedMap map[string]any
			err = json.Unmarshal(expected, &expectedMap)
			require.NoError(t, err)

			if d := cmp.Diff(expectedMap, result); d != "" {
				t.Errorf("Schema has incorrect value (-want,+got):\n%s", d)
			}
		})
	}
}

func TestCreateSchemaWithRootProperties(t *testing.T) {
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
			expected, err := os.ReadFile(filepath.Join(schemaPath, name, "schema-with-title.json"))
			require.NoError(t, err)

			result, err := CreateSchema(filepath.Join(tfPath, name), CreateSchemaOptions{
				RequireAll:                false,
				AllowAdditionalProperties: true,
				AllowEmpty:                true,
				NullableAll:               false,
				IgnoreVariables:           []string{"ignored", "also_ignored"},
				RootProperties: map[string]string{
					"$id":   "http://example.com/schema",
					"title": "Example Schema",
				},
			})
			require.NoError(t, err)

			var expectedMap map[string]any
			err = json.Unmarshal(expected, &expectedMap)
			require.NoError(t, err)

			if d := cmp.Diff(expectedMap, result); d != "" {
				t.Errorf("Schema has incorrect value (-want,+got):\n%s", d)
			}
		})
	}
}

type errorLocation struct {
	name            string
	nestedLocations []errorLocation
}

func getErrorLocationsFromValidationErr(t *testing.T, valErr *jsonschema.ValidationError) []errorLocation {
	t.Helper()
	if len(valErr.Causes) == 0 {
		return nil
	}

	keywordLocations := []errorLocation{}

	for _, cause := range valErr.Causes {
		newLocation := errorLocation{
			name:            cause.KeywordLocation,
			nestedLocations: getErrorLocationsFromValidationErr(t, cause),
		}
		keywordLocations = append(keywordLocations, newLocation)
	}

	slices.SortFunc(keywordLocations, func(a, b errorLocation) int {
		if a.name < b.name {
			return -1
		}
		if a.name == b.name {
			return 0
		}

		return 1
	})

	return keywordLocations
}

//nolint:funlen,maintidx
func TestSampleInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		schemaPath       string
		filePath         string
		keywordLocations []errorLocation
	}{
		// Minimum input
		{
			name:             "empty minimum input",
			filePath:         "../../test/expected/empty/sample-input/test-input-min.json",
			schemaPath:       "../../test/expected/empty/schema.json",
			keywordLocations: nil,
		},
		{
			name:             "simple minimum input",
			filePath:         "../../test/expected/simple/sample-input/test-input-min.json",
			schemaPath:       "../../test/expected/simple/schema.json",
			keywordLocations: nil,
		},
		{
			name:             "simple-types minimum input",
			filePath:         "../../test/expected/simple-types/sample-input/test-input-min.json",
			schemaPath:       "../../test/expected/simple-types/schema.json",
			keywordLocations: nil,
		},
		{
			name:             "complex-types minimum input",
			filePath:         "../../test/expected/complex-types/sample-input/test-input-min.json",
			schemaPath:       "../../test/expected/complex-types/schema.json",
			keywordLocations: nil,
		},
		{
			name:             "custom-validation minimum input",
			filePath:         "../../test/expected/custom-validation/sample-input/test-input-min.json",
			schemaPath:       "../../test/expected/custom-validation/schema.json",
			keywordLocations: nil,
		},
		// Maximum input plus an unknown variable, additionalProperties is true
		{
			name:             "simple full input",
			filePath:         "../../test/expected/simple/sample-input/test-input-all.json",
			schemaPath:       "../../test/expected/simple/schema.json",
			keywordLocations: nil,
		},
		{
			name:             "simple-types full input",
			filePath:         "../../test/expected/simple-types/sample-input/test-input-all.json",
			schemaPath:       "../../test/expected/simple-types/schema.json",
			keywordLocations: nil,
		},
		{
			name:             "complex-types full input",
			filePath:         "../../test/expected/complex-types/sample-input/test-input-all.json",
			schemaPath:       "../../test/expected/complex-types/schema.json",
			keywordLocations: nil,
		},
		{
			name:             "custom-validation full input",
			filePath:         "../../test/expected/custom-validation/sample-input/test-input-all.json",
			schemaPath:       "../../test/expected/custom-validation/schema.json",
			keywordLocations: nil,
		},
		// Maximum input plus an unknown variable, additionalProperties is false
		{
			name:       "simple full input additionalProperties false",
			filePath:   "../../test/expected/simple/sample-input/test-input-all.json",
			schemaPath: "../../test/expected/simple/schema-disallow-additional.json",
			keywordLocations: []errorLocation{
				{name: "/additionalProperties"},
			},
		},
		{
			name:       "simple-types full input additionalProperties false",
			filePath:   "../../test/expected/simple-types/sample-input/test-input-all.json",
			schemaPath: "../../test/expected/simple-types/schema-disallow-additional.json",
			keywordLocations: []errorLocation{
				{name: "/additionalProperties"},
			},
		},
		{
			name:       "complex-types full input additionalProperties false",
			filePath:   "../../test/expected/complex-types/sample-input/test-input-all.json",
			schemaPath: "../../test/expected/complex-types/schema-disallow-additional.json",
			keywordLocations: []errorLocation{
				{name: "/additionalProperties"},
			},
		},
		{
			name:       "custom-validation full input additionalProperties false",
			filePath:   "../../test/expected/custom-validation/sample-input/test-input-all.json",
			schemaPath: "../../test/expected/custom-validation/schema-disallow-additional.json",
			keywordLocations: []errorLocation{
				{name: "/additionalProperties"},
				{name: "/properties/an_object_maximum_minimum_items/additionalProperties"},
			},
		},
		// bad input on all fields
		{
			name:       "simple bad input",
			filePath:   "../../test/expected/simple/sample-input/test-input-bad.json",
			schemaPath: "../../test/expected/simple/schema.json",
			keywordLocations: []errorLocation{
				{name: "/properties/name/type"},
				{name: "/required"},
			},
		},
		{
			name:       "simple-types bad input",
			filePath:   "../../test/expected/simple-types/sample-input/test-input-bad.json",
			schemaPath: "../../test/expected/simple-types/schema.json",
			keywordLocations: []errorLocation{
				{name: "/properties/a_bool/type"},
				{name: "/properties/a_list/items/type"},
				{
					name: "/properties/a_map_of_strings",
					nestedLocations: []errorLocation{
						{name: "/properties/a_map_of_strings/additionalProperties/type"},
						{name: "/properties/a_map_of_strings/additionalProperties/type"},
					},
				},
				{
					name: "/properties/a_nullable_string/oneOf",
					nestedLocations: []errorLocation{
						{name: "/properties/a_nullable_string/oneOf/0/type"},
						{name: "/properties/a_nullable_string/oneOf/1/type"},
					},
				},
				{name: "/properties/a_set/uniqueItems"},
				{name: "/properties/a_string/type"},
				{name: "/properties/a_tuple/minItems"},
				{
					name: "/properties/an_object",
					nestedLocations: []errorLocation{
						{name: "/properties/an_object/properties/c/type"},
						{name: "/properties/an_object/required"},
					},
				},
				{name: "/required"},
			},
		},
		{
			name:       "complex-types bad input",
			filePath:   "../../test/expected/complex-types/sample-input/test-input-bad.json",
			schemaPath: "../../test/expected/complex-types/schema.json",
			keywordLocations: []errorLocation{
				{
					name: "/properties/a_very_complicated_object",
					nestedLocations: []errorLocation{
						{name: "/properties/a_very_complicated_object/properties/a/type"},
						{name: "/properties/a_very_complicated_object/properties/b/minItems"},
						{name: "/properties/a_very_complicated_object/properties/c/additionalProperties/type"},
						{
							name: "/properties/a_very_complicated_object/properties/d",
							nestedLocations: []errorLocation{
								{
									name: "/properties/a_very_complicated_object/properties/d/properties/a",
									nestedLocations: []errorLocation{
										{name: "/properties/a_very_complicated_object/properties/d/properties/a/items/items/type"},
										{name: "/properties/a_very_complicated_object/properties/d/properties/a/items/type"},
									},
								},
								{name: "/properties/a_very_complicated_object/properties/d/properties/b/type"},
							},
						},
						{name: "/properties/a_very_complicated_object/properties/e/items/1/type"},
						{
							name: "/properties/a_very_complicated_object/properties/f",
							nestedLocations: []errorLocation{
								{name: "/properties/a_very_complicated_object/properties/f/items/items/type"},
								{name: "/properties/a_very_complicated_object/properties/f/uniqueItems"},
							},
						},
					},
				},
				{
					name: "/properties/an_object_with_optional",
					nestedLocations: []errorLocation{
						{name: "/properties/an_object_with_optional/properties/a/type"},
						{name: "/properties/an_object_with_optional/properties/b/type"},
						{name: "/properties/an_object_with_optional/properties/d/type"},
						{name: "/properties/an_object_with_optional/required"},
					},
				},
			},
		},
		{
			name:       "custom-validation bad input",
			filePath:   "../../test/expected/custom-validation/sample-input/test-input-bad.json",
			schemaPath: "../../test/expected/custom-validation/schema.json",
			keywordLocations: []errorLocation{
				{name: "/properties/a_list_maximum_minimum_length/minItems"},
				{name: "/properties/a_map_maximum_minimum_entries/minProperties"},
				{name: "/properties/a_number_enum_kind_1/type"},
				{name: "/properties/a_number_enum_kind_2/enum"},
				{name: "/properties/a_number_exclusive_maximum_minimum/exclusiveMaximum"},
				{name: "/properties/a_number_maximum_minimum/maximum"},
				{
					name: "/properties/a_set_maximum_minimum_items",
					nestedLocations: []errorLocation{
						{name: "/properties/a_set_maximum_minimum_items/maxItems"},
						{name: "/properties/a_set_maximum_minimum_items/uniqueItems"},
					},
				},
				{name: "/properties/a_string_enum_escaped_characters_kind_1/enum"},
				{name: "/properties/a_string_enum_escaped_characters_kind_2/enum"},
				{name: "/properties/a_string_enum_kind_1/enum"},
				{name: "/properties/a_string_enum_kind_2/type"},
				{name: "/properties/a_string_maximum_minimum_length/maxLength"},
				{name: "/properties/a_string_multiple_validation_conditions/minLength"},
				{name: "/properties/a_string_pattern_1/pattern"},
				{name: "/properties/a_string_pattern_2/pattern"},
				{
					name: "/properties/an_object_maximum_minimum_items",
					nestedLocations: []errorLocation{
						{name: "/properties/an_object_maximum_minimum_items/maxProperties"},
						{name: "/properties/an_object_maximum_minimum_items/properties/name/type"},
					},
				},
			},
		},
		// null input on all fields, nullableAll is false
		{
			name:       "simple null input nullableAll false",
			filePath:   "../../test/expected/simple/sample-input/test-input-null.json",
			schemaPath: "../../test/expected/simple/schema.json",
			keywordLocations: []errorLocation{
				{name: "/properties/age/type"},
				{name: "/properties/name/type"},
			},
		},
		{
			name:       "simple-types null input nullableAll false",
			filePath:   "../../test/expected/simple-types/sample-input/test-input-null.json",
			schemaPath: "../../test/expected/simple-types/schema.json",
			keywordLocations: []errorLocation{
				{name: "/properties/a_bool/type"},
				{name: "/properties/a_list/type"},
				{name: "/properties/a_list_of_any/type"},
				{name: "/properties/a_map_of_any/type"},
				{name: "/properties/a_map_of_strings/type"},
				{name: "/properties/a_number/type"},
				{name: "/properties/a_set/type"},
				{name: "/properties/a_set_of_any/type"},
				{name: "/properties/a_string/type"},
				{name: "/properties/a_tuple/type"},
				{name: "/properties/a_variable_in_another_file/type"},
				{name: "/properties/an_any_as_boolean/anyOf", nestedLocations: []errorLocation{
					{name: "/properties/an_any_as_boolean/anyOf/0/type"},
					{name: "/properties/an_any_as_boolean/anyOf/1/type"},
					{name: "/properties/an_any_as_boolean/anyOf/2/type"},
					{name: "/properties/an_any_as_boolean/anyOf/3/type"},
					{name: "/properties/an_any_as_boolean/anyOf/4/type"},
				}},
				{name: "/properties/an_any_as_list/anyOf", nestedLocations: []errorLocation{
					{name: "/properties/an_any_as_list/anyOf/0/type"},
					{name: "/properties/an_any_as_list/anyOf/1/type"},
					{name: "/properties/an_any_as_list/anyOf/2/type"},
					{name: "/properties/an_any_as_list/anyOf/3/type"},
					{name: "/properties/an_any_as_list/anyOf/4/type"},
				}},
				{name: "/properties/an_any_as_map/anyOf", nestedLocations: []errorLocation{
					{name: "/properties/an_any_as_map/anyOf/0/type"},
					{name: "/properties/an_any_as_map/anyOf/1/type"},
					{name: "/properties/an_any_as_map/anyOf/2/type"},
					{name: "/properties/an_any_as_map/anyOf/3/type"},
					{name: "/properties/an_any_as_map/anyOf/4/type"},
				}},
				{name: "/properties/an_any_as_number/anyOf", nestedLocations: []errorLocation{
					{name: "/properties/an_any_as_number/anyOf/0/type"},
					{name: "/properties/an_any_as_number/anyOf/1/type"},
					{name: "/properties/an_any_as_number/anyOf/2/type"},
					{name: "/properties/an_any_as_number/anyOf/3/type"},
					{name: "/properties/an_any_as_number/anyOf/4/type"},
				}},
				{name: "/properties/an_any_as_string/anyOf", nestedLocations: []errorLocation{
					{name: "/properties/an_any_as_string/anyOf/0/type"},
					{name: "/properties/an_any_as_string/anyOf/1/type"},
					{name: "/properties/an_any_as_string/anyOf/2/type"},
					{name: "/properties/an_any_as_string/anyOf/3/type"},
					{name: "/properties/an_any_as_string/anyOf/4/type"},
				}},
				{name: "/properties/an_object/type"},
				{name: "/properties/an_unspecified_as_boolean/anyOf", nestedLocations: []errorLocation{
					{name: "/properties/an_unspecified_as_boolean/anyOf/0/type"},
					{name: "/properties/an_unspecified_as_boolean/anyOf/1/type"},
					{name: "/properties/an_unspecified_as_boolean/anyOf/2/type"},
					{name: "/properties/an_unspecified_as_boolean/anyOf/3/type"},
					{name: "/properties/an_unspecified_as_boolean/anyOf/4/type"},
				}},
				{name: "/properties/an_unspecified_as_list/anyOf", nestedLocations: []errorLocation{
					{name: "/properties/an_unspecified_as_list/anyOf/0/type"},
					{name: "/properties/an_unspecified_as_list/anyOf/1/type"},
					{name: "/properties/an_unspecified_as_list/anyOf/2/type"},
					{name: "/properties/an_unspecified_as_list/anyOf/3/type"},
					{name: "/properties/an_unspecified_as_list/anyOf/4/type"},
				}},
				{name: "/properties/an_unspecified_as_map/anyOf", nestedLocations: []errorLocation{
					{name: "/properties/an_unspecified_as_map/anyOf/0/type"},
					{name: "/properties/an_unspecified_as_map/anyOf/1/type"},
					{name: "/properties/an_unspecified_as_map/anyOf/2/type"},
					{name: "/properties/an_unspecified_as_map/anyOf/3/type"},
					{name: "/properties/an_unspecified_as_map/anyOf/4/type"},
				}},
				{name: "/properties/an_unspecified_as_number/anyOf", nestedLocations: []errorLocation{
					{name: "/properties/an_unspecified_as_number/anyOf/0/type"},
					{name: "/properties/an_unspecified_as_number/anyOf/1/type"},
					{name: "/properties/an_unspecified_as_number/anyOf/2/type"},
					{name: "/properties/an_unspecified_as_number/anyOf/3/type"},
					{name: "/properties/an_unspecified_as_number/anyOf/4/type"},
				}},
				{name: "/properties/an_unspecified_as_string/anyOf", nestedLocations: []errorLocation{
					{name: "/properties/an_unspecified_as_string/anyOf/0/type"},
					{name: "/properties/an_unspecified_as_string/anyOf/1/type"},
					{name: "/properties/an_unspecified_as_string/anyOf/2/type"},
					{name: "/properties/an_unspecified_as_string/anyOf/3/type"},
					{name: "/properties/an_unspecified_as_string/anyOf/4/type"},
				}},
			},
		},
		{
			name:       "complex-types null input nullableAll false",
			filePath:   "../../test/expected/complex-types/sample-input/test-input-null.json",
			schemaPath: "../../test/expected/complex-types/schema.json",
			keywordLocations: []errorLocation{
				{name: "/properties/a_very_complicated_object/type"},
				{name: "/properties/an_object_with_optional/type"},
			},
		},
		{
			name:       "custom-validation null input nullableAll false",
			filePath:   "../../test/expected/custom-validation/sample-input/test-input-null.json",
			schemaPath: "../../test/expected/custom-validation/schema.json",
			keywordLocations: []errorLocation{
				{name: "/properties/a_complex_condition_with_complex_error_message/type"},
				{name: "/properties/a_list_maximum_minimum_length/type"},
				{name: "/properties/a_map_maximum_minimum_entries/type"},
				{name: "/properties/a_number_enum_kind_1/type"},
				{name: "/properties/a_number_enum_kind_2/type"},
				{name: "/properties/a_number_exclusive_maximum_minimum/type"},
				{name: "/properties/a_number_maximum_minimum/type"},
				{name: "/properties/a_set_maximum_minimum_items/type"},
				{name: "/properties/a_string_enum_escaped_characters_kind_1/type"},
				{name: "/properties/a_string_enum_escaped_characters_kind_2/type"},
				{name: "/properties/a_string_enum_kind_1/type"},
				{name: "/properties/a_string_enum_kind_2/type"},
				{name: "/properties/a_string_length_over_defined/type"},
				{name: "/properties/a_string_maximum_minimum_length/type"},
				{name: "/properties/a_string_multiple_validation_conditions/type"},
				{name: "/properties/a_string_pattern_1/type"},
				{name: "/properties/a_string_pattern_2/type"},
				{name: "/properties/a_string_set_length/type"},
				{name: "/properties/an_object_maximum_minimum_items/type"},
			},
		},
		// null input on all fields, nullableAll is true
		{
			name:             "simple null input nullableAll true",
			filePath:         "../../test/expected/simple/sample-input/test-input-null.json",
			schemaPath:       "../../test/expected/simple/schema-nullable-all.json",
			keywordLocations: nil,
		},
		{
			name:             "simple-types null input nullableAll true",
			filePath:         "../../test/expected/simple-types/sample-input/test-input-null.json",
			schemaPath:       "../../test/expected/simple-types/schema-nullable-all.json",
			keywordLocations: nil,
		},
		{
			name:             "complex-types null input nullableAll true",
			filePath:         "../../test/expected/complex-types/sample-input/test-input-null.json",
			schemaPath:       "../../test/expected/complex-types/schema-nullable-all.json",
			keywordLocations: nil,
		},
		{
			// of note: custom validation still applies to nullable fields, and sometimes 'null' doesn't satisfy the
			// condition, meaning these fields effectively can't be null.
			// This seems to primarily be the case for "enum" fields. Other fields tend to ignore this error, in JSON Schema.
			name:       "custom-validation null input nullableAll true",
			filePath:   "../../test/expected/custom-validation/sample-input/test-input-null.json",
			schemaPath: "../../test/expected/custom-validation/schema-nullable-all.json",
			keywordLocations: []errorLocation{
				{name: "/properties/a_number_enum_kind_1/enum"},
				{name: "/properties/a_number_enum_kind_2/enum"},
				{name: "/properties/a_string_enum_kind_1/enum"},
				{name: "/properties/a_string_enum_kind_2/enum"},
			},
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path, err := filepath.Abs(tc.schemaPath)
			require.NoError(t, err)
			c := jsonschema.NewCompiler()
			f, err := os.Open(path)
			require.NoError(t, err)
			err = c.AddResource("file://"+path, f)
			require.NoError(t, err)
			s, err := c.Compile("file://" + path)
			require.NoError(t, err)

			input, err := os.ReadFile(tc.filePath)
			require.NoError(t, err)
			var m any
			err = json.Unmarshal(input, &m)
			require.NoError(t, err)

			var keywordLocations []errorLocation
			err = s.Validate(m)
			if err != nil {
				valErr := &jsonschema.ValidationError{}
				ok := errors.As(err, &valErr)
				require.True(t, ok)
				keywordLocations = getErrorLocationsFromValidationErr(t, valErr)
			}
			require.Equal(t, tc.keywordLocations, keywordLocations)
		})
	}
}
