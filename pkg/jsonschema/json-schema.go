// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package jsonschema

import (
	"errors"
	"fmt"
	"slices"

	"github.com/HewlettPackard/terraschema/pkg/model"
	"github.com/HewlettPackard/terraschema/pkg/reader"
)

type CreateSchemaOptions struct {
	RequireAll                bool
	AllowAdditionalProperties bool
	AllowEmpty                bool
}

func CreateSchema(path string, options CreateSchemaOptions) (map[string]any, error) {
	schemaOut := make(map[string]any)

	varMap, err := reader.GetVarMap(path)
	if err != nil {
		if errors.Is(err, reader.ErrFilesNotFound) {
			if options.AllowEmpty {
				fmt.Printf("No tf files were found in %q, creating empty schema\n", path)

				return schemaOut, nil
			}
		} else {
			return schemaOut, fmt.Errorf("error reading tf files at %q: %w", path, err)
		}
	}

	if len(varMap) == 0 {
		if options.AllowEmpty {
			return schemaOut, nil
		} else {
			return schemaOut, errors.New("no variables found in tf files")
		}
	}

	schemaOut["$schema"] = "http://json-schema.org/draft-07/schema#"

	if !options.AllowAdditionalProperties {
		schemaOut["additionalProperties"] = false
	} else {
		schemaOut["additionalProperties"] = true
	}

	properties := make(map[string]any)
	requiredArray := []any{}
	for name, variable := range varMap {
		if variable.Required && !options.RequireAll {
			requiredArray = append(requiredArray, name)
		}
		if options.RequireAll {
			requiredArray = append(requiredArray, name)
		}
		node, err := createNode(name, variable, options)
		if err != nil {
			return schemaOut, fmt.Errorf("error creating node for %q: %w", name, err)
		}

		properties[name] = node
	}

	schemaOut["properties"] = properties

	slices.SortFunc(requiredArray, sortInterfaceAlphabetical) // get required in alphabetical order
	schemaOut["required"] = requiredArray

	return schemaOut, nil
}

func createNode(name string, v model.TranslatedVariable, options CreateSchemaOptions) (map[string]any, error) {
	tc, err := reader.GetTypeConstraint(v.Variable.Type)
	if err != nil {
		return nil, fmt.Errorf("getting type constraint for %q: %w", name, err)
	}

	nullableIsTrue := v.Variable.Nullable != nil && *v.Variable.Nullable
	node, err := getNodeFromType(name, tc, nullableIsTrue, options)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", name, err)
	}

	if v.Variable.Default != nil {
		def, err := expressionToJSONObject(v.Variable.Default)
		if err != nil {
			return nil, fmt.Errorf("error converting default value to JSON object: %w", err)
		}
		node["default"] = def
	}

	if v.Variable.Validation != nil && v.ConditionAsString != nil {
		err = parseConditionToNode(v.Variable.Validation.Condition, *v.ConditionAsString, name, &node)
		// if an error occurs, log it and continue.
		if err != nil {
			fmt.Printf("couldn't apply validation for %q with condition %q. Error: %v\n", name, *v.ConditionAsString, err)
		}
	}

	if v.Variable.Description != nil {
		node["description"] = *v.Variable.Description
	}

	return node, nil
}

func sortInterfaceAlphabetical(a, b any) int {
	aString, ok := a.(string)
	if !ok {
		return 0
	}
	bString, ok := b.(string)
	if !ok {
		return 0
	}
	if aString < bString {
		return -1
	}
	if aString > bString {
		return 1
	}

	return 0
}
