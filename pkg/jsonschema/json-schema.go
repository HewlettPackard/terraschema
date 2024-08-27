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
	DebugOut                  bool
	SuppressLogging           bool
	NullableAll               bool
}

func CreateSchema(path string, options CreateSchemaOptions) (map[string]any, error) {
	schemaOut := make(map[string]any)

	varMap, err := reader.GetVarMap(path, options.DebugOut)
	if err != nil {
		if options.AllowEmpty && (errors.Is(err, reader.ErrFilesNotFound) || errors.Is(err, reader.ErrNoVariablesFound)) {
			if !options.SuppressLogging {
				fmt.Printf("Warning: directory %q: %v, creating empty schema file\n", path, err)
			}

			return schemaOut, nil
		} else {
			return schemaOut, fmt.Errorf("error reading tf files at %q: %w", path, err)
		}
	}

	schemaOut["$schema"] = "http://json-schema.org/draft-07/schema#"

	schemaOut["additionalProperties"] = options.AllowAdditionalProperties

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

	// The default value for nullable is the value of NullableAll. For the purpose of keeping the JSON Schema relatively
	// clean, this is normally set to false. Setting the default value to true is consistent with Terraform behavior.
	nullableTranslatedValue := options.NullableAll
	if v.Variable.Nullable != nil {
		nullableTranslatedValue = *v.Variable.Nullable
	}

	node, err := getNodeFromType(name, tc, nullableTranslatedValue, options)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", name, err)
	}

	if v.Variable.Default != nil {
		def, err := reader.ExpressionToJSONObject(v.Variable.Default)
		if err != nil {
			return nil, fmt.Errorf("error converting default value to JSON object: %w", err)
		}
		node["default"] = def
	}

	if v.Variable.Validation != nil && v.ConditionAsString != nil {
		err = parseConditionToNode(v.Variable.Validation.Condition, *v.ConditionAsString, name, &node)
		// if an error occurs, log it and continue.
		if err != nil && !options.SuppressLogging {
			fmt.Printf("Warning: couldn't apply validation for %q with condition %q: %v\n",
				name,
				*v.ConditionAsString,
				err,
			)
			// if the debug flag is set, print all the errors returned by each of the rules as they try to apply to the condition.
			var validationError ValidationApplyError
			if ok := errors.As(err, &validationError); ok && options.DebugOut {
				fmt.Printf("Debug: condition located at %q\n", v.Variable.Validation.Condition.Range().String())
				fmt.Println("Debug: the following errors occurred:")
				for k, v := range validationError.ErrorMap {
					fmt.Printf("\t%s: %v\n", k, v)
				}
			}
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
