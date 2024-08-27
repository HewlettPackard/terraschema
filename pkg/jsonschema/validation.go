// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package jsonschema

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/HewlettPackard/terraschema/pkg/reader"
)

type conditionMutator func(hcl.Expression, string, string) (map[string]any, error)

var ErrConditionNotApplied = fmt.Errorf("no translation rules are supported for this condition")

type ValidationApplyError struct {
	error
	ErrorMap map[string]error
}

func parseConditionToNode(ex hcl.Expression, _ string, name string, m *map[string]any) error {
	if m == nil {
		return fmt.Errorf("node is nil")
	}
	t, ok := (*m)["type"].(string)
	if !ok {
		return fmt.Errorf("cannot apply validation, type is not defined for %v", *m)
	}
	functions := map[string]conditionMutator{
		"contains([...],var.input_parameter)":          contains,
		"var == \"a\" || var == \"b\"":                 isOneOf,
		"a <>= (variable or variable length) (&& ...)": comparison,
		"can(regex(\"...\",var.input_parameter))":      canRegex,
	}

	errorMap := make(map[string]error)
	for fnName, fn := range functions {
		updatedNode, err := fn(ex, name, t)
		if err == nil {
			// apply updated node to m:
			for k, v := range updatedNode {
				(*m)[k] = v
			}

			return nil
		}
		errorMap[fnName] = err
	}

	return ValidationApplyError{ErrConditionNotApplied, errorMap}
}

func isOneOf(ex hcl.Expression, name string, _ string) (map[string]any, error) {
	enum := []any{}
	err := walkIsOneOf(ex, name, &enum)
	if err != nil {
		return nil, err
	}

	if len(enum) == 0 {
		return nil, fmt.Errorf("no options found")
	}

	return map[string]any{"enum": enum}, nil
}

func contains(ex hcl.Expression, name string, _ string) (map[string]any, error) {
	args, ok := argumentsOfCall(ex, "contains", 2)
	if !ok {
		return nil, fmt.Errorf("condition is not a 'contains()' function")
	}

	l, d := hcl.ExprList(args[0])
	if d.HasErrors() {
		return nil, fmt.Errorf("first argument is not a list")
	}

	if !isExpressionVarName(args[1], name) {
		return nil, fmt.Errorf("second argument is not a direct reference to the input variable")
	}

	newEnum := []any{}
	for _, val := range l {
		simple, err := reader.ExpressionToJSONObject(val)
		if err != nil {
			return nil, fmt.Errorf("value in list could not be converted to JSON")
		}
		newEnum = append(newEnum, simple)
	}

	return map[string]any{"enum": newEnum}, nil
}

func comparison(ex hcl.Expression, name string, t string) (map[string]any, error) {
	allowedTypes := map[string]bool{
		"object": true,
		"array":  true,
		"number": true,
		"string": true,
	}
	if !allowedTypes[t] {
		return nil, fmt.Errorf("rule can only be applied to object, array, number or string types, not %q", t)
	}

	node := map[string]any{"type": t}
	err := walkComparison(ex, name, &node, t)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func canRegex(ex hcl.Expression, name string, t string) (map[string]any, error) {
	if t != "string" {
		return nil, fmt.Errorf("rule can only be applied to string types, not %q", t)
	}

	canArgs, ok := argumentsOfCall(ex, "can", 1)
	if !ok {
		return nil, fmt.Errorf("condition is not a 'can()' function")
	}

	regexArgs, ok := argumentsOfCall(canArgs[0], "regex", 2)
	if !ok {
		return nil, fmt.Errorf("condition is not a 'can(regex())' function")
	}
	if !isExpressionVarName(regexArgs[1], name) {
		return nil, fmt.Errorf("second argument is not a direct reference to the input variable")
	}

	patternJSON, err := reader.ExpressionToJSONObject(regexArgs[0])
	if err != nil {
		return nil, fmt.Errorf("pattern could not be converted to JSON: %w", err)
	}

	return map[string]any{"pattern": patternJSON}, nil
}
