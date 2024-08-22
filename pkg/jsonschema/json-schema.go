// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package jsonschema

import (
	"errors"
	"fmt"
	"slices"

	"github.com/HewlettPackard/terraschema/pkg/model"
	"github.com/HewlettPackard/terraschema/pkg/reader"
)

func CreateSchema(path string, strict bool) (map[string]any, error) {
	schemaOut := make(map[string]any)

	varMap, err := reader.GetVarMap(path)
	// GetVarMaps returns an error if no .tf files are found in the directory. We
	// ignore this error for now.
	if err != nil && !errors.Is(err, reader.ErrFilesNotFound) {
		return schemaOut, fmt.Errorf("error reading tf files at %s: %w", path, err)
	}

	if len(varMap) == 0 {
		return schemaOut, nil
	}

	schemaOut["$schema"] = "http://json-schema.org/draft-07/schema#"

	if strict {
		schemaOut["additionalProperties"] = false
	} else {
		schemaOut["additionalProperties"] = true
	}

	properties := make(map[string]any)
	requiredArray := []any{}
	for name, variable := range varMap {
		if variable.Required {
			requiredArray = append(requiredArray, name)
		}
		node, err := createNode(name, variable, strict)
		if err != nil {
			return schemaOut, fmt.Errorf("error creating node for %s: %w", name, err)
		}

		properties[name] = node
	}

	schemaOut["properties"] = properties

	slices.SortFunc(requiredArray, sortInterfaceAlphabetical) // get required in alphabetical order
	schemaOut["required"] = requiredArray

	return schemaOut, nil
}

func createNode(name string, v model.TranslatedVariable, strict bool) (map[string]any, error) {
	tc, err := reader.GetTypeConstraint(v.Variable.Type)
	if err != nil {
		return nil, fmt.Errorf("getting type constraint for %s: %w", name, err)
	}

	nullableIsTrue := v.Variable.Nullable != nil && *v.Variable.Nullable
	node, err := getNodeFromType(name, tc, nullableIsTrue, strict)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
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
