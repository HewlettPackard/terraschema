// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package jsonschema

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/HewlettPackard/terraschema/pkg/reader"
)

func isExpressionVarName(ex hcl.Expression, name string) bool {
	newEx, ok := ex.(*hclsyntax.ScopeTraversalExpr)
	if !ok {
		return false
	}
	if newEx == nil {
		return false
	}

	t := newEx.Traversal
	if len(t) != 2 {
		return false
	}

	root, ok := t[0].(hcl.TraverseRoot)
	if !ok {
		return false
	}
	attr, ok := t[1].(hcl.TraverseAttr)
	if !ok {
		return false
	}

	return root.Name == "var" && attr.Name == name
}

func isExpressionLengthVarName(ex hcl.Expression, name string) bool {
	args, ok := argumentsOfCall(ex, "length", 1)
	if !ok {
		return false
	}

	return isExpressionVarName(args[0], name)
}

func argumentsOfCall(ex hcl.Expression, functionName string, args int) ([]hcl.Expression, bool) {
	call, d := hcl.ExprCall(ex)
	if d.HasErrors() {
		return nil, false
	}
	if call.Name != functionName {
		return nil, false
	}
	if len(call.Arguments) != args {
		return nil, false
	}

	return call.Arguments, true
}

func walkComparison(ex hcl.Expression, name string, node *map[string]any, nodeType string) error {
	var err error
	switch ex := ex.(type) {
	case *hclsyntax.BinaryOpExpr:
		switch ex.Op {
		case hclsyntax.OpLogicalAnd: // &&
			err = walkComparison(ex.LHS, name, node, nodeType)
			if err != nil {
				return err
			}

			return walkComparison(ex.RHS, name, node, nodeType)
		case hclsyntax.OpGreaterThan,
			hclsyntax.OpGreaterThanOrEqual,
			hclsyntax.OpLessThanOrEqual,
			hclsyntax.OpLessThan,
			hclsyntax.OpEqual:
			return parseComparisonExpression(ex, name, node, nodeType)
		default:
			return fmt.Errorf("operator is not one of && <=, >=, <, >, ==")
		}
	case *hclsyntax.ParenthesesExpr:
		return walkComparison(ex.Expression, name, node, nodeType)
	default:
		return fmt.Errorf("could not evaluate expression")
	}
}

func parseComparisonExpression(ex *hclsyntax.BinaryOpExpr, name string, node *map[string]any, nodeType string) error {
	if isExpressionVarName(ex.RHS, name) || isExpressionLengthVarName(ex.RHS, name) {
		// swap the LHS and RHS
		ex.LHS, ex.RHS = ex.RHS, ex.LHS
		ex.Op = flipSign(ex.Op)
		if ex.Op == nil {
			return fmt.Errorf("could not flip sign")
		}
	}
	// parse the right hand side as a constant numeric value. Can't reference other variables since that would require
	// making a more complex JSON schema (eg with "if" and "then" properties).
	val, d := ex.RHS.Value(&hcl.EvalContext{})
	if d.HasErrors() {
		return fmt.Errorf("could not evaluate expression as a constant value: %w", d)
	}
	var num float64
	err := gocty.FromCtyValue(val, &num)
	if err != nil {
		return fmt.Errorf("could not convert value to number: %w", err)
	}

	// valid comparisons:
	// var <> number when type(var) == number
	// len(var) <> number when type(var) == array,object,string
	// len(var) == number when type(var) == array,object,string

	condition1 := isExpressionVarName(ex.LHS, name) &&
		nodeType == "number" &&
		ex.Op != hclsyntax.OpEqual // don't compare a number to a number with ==. Use const if needed later.
	condition2 := isExpressionLengthVarName(ex.LHS, name) &&
		(nodeType == "array" || nodeType == "object" || nodeType == "string")
	if condition1 || condition2 {
		return performOp(ex.Op, node, num, nodeType)
	} else {
		return fmt.Errorf("operation not supported for type %q op %v", nodeType, ex.Op)
	}
}

func performOp(op *hclsyntax.Operation, node *map[string]any, num float64, nodeType string) error {
	// bundle the operation and type name together
	type operationWithTypeName struct {
		operation *hclsyntax.Operation
		typeName  string
	}
	// get the field name and whether the field is exclusive, which is a term i use to
	// describe whole number quantities paired with "must be greater than this" etc., so
	// in that case I have to add 1 to the value before updating "minItems" or similar.
	type fieldInfo struct {
		minField  string
		maxField  string
		exclusive bool
	}
	// making a lookup table here so I can match an op and a type to its corresponding field
	// in a JSON schema.
	fieldMap := map[operationWithTypeName]fieldInfo{
		{hclsyntax.OpGreaterThan, "number"}:        {"exclusiveMinimum", "", false},
		{hclsyntax.OpGreaterThanOrEqual, "number"}: {"minimum", "", false},
		{hclsyntax.OpLessThan, "number"}:           {"", "exclusiveMaximum", false},
		{hclsyntax.OpLessThanOrEqual, "number"}:    {"", "maximum", false},
		{hclsyntax.OpEqual, "number"}:              {"minimum", "maximum", false},

		{hclsyntax.OpGreaterThan, "array"}:        {"minItems", "", true},
		{hclsyntax.OpGreaterThanOrEqual, "array"}: {"minItems", "", false},
		{hclsyntax.OpLessThan, "array"}:           {"", "maxItems", true},
		{hclsyntax.OpLessThanOrEqual, "array"}:    {"", "maxItems", false},
		{hclsyntax.OpEqual, "array"}:              {"minItems", "maxItems", false},

		{hclsyntax.OpGreaterThan, "object"}:        {"minProperties", "", true},
		{hclsyntax.OpGreaterThanOrEqual, "object"}: {"minProperties", "", false},
		{hclsyntax.OpLessThan, "object"}:           {"", "maxProperties", true},
		{hclsyntax.OpLessThanOrEqual, "object"}:    {"", "maxProperties", false},
		{hclsyntax.OpEqual, "object"}:              {"minProperties", "maxProperties", false},

		{hclsyntax.OpGreaterThan, "string"}:        {"minLength", "", true},
		{hclsyntax.OpGreaterThanOrEqual, "string"}: {"minLength", "", false},
		{hclsyntax.OpLessThan, "string"}:           {"", "maxLength", true},
		{hclsyntax.OpLessThanOrEqual, "string"}:    {"", "maxLength", false},
		{hclsyntax.OpEqual, "string"}:              {"minLength", "maxLength", false},
	}
	info, ok := fieldMap[operationWithTypeName{op, nodeType}]
	if !ok {
		return fmt.Errorf("operation not supported for type %q op %v", nodeType, op)
	}
	if info.minField != "" {
		if info.exclusive {
			num += 1
		}
		if canUpdateField(node, num, info.minField, false) {
			(*node)[info.minField] = num
		}
	}
	if info.maxField != "" {
		if info.exclusive {
			num -= 1
		}
		if canUpdateField(node, num, info.maxField, true) {
			(*node)[info.maxField] = num
		}
	}

	return nil
}

// each of the minimum and maximum fields must pass this validation step before they are updated
// the field must not exist
// or
//   - if the field exists but can't be converted to a float, do not update it.
//   - if mustDecrease is true, the new value must be less than the current value
//   - if mustDecrease is false, the new value must be greater than the current value
func canUpdateField(node *map[string]any, num float64, fieldName string, mustDecrease bool) bool {
	if node == nil {
		// would result in nil pointer dereference
		return false
	}
	field, ok := (*node)[fieldName]
	if !ok {
		// can update field if it doesn't exist
		return true
	}
	current, ok := field.(float64)
	if !ok {
		// field exists but isn't a float64
		return false
	}
	if mustDecrease {
		return num < current
	}

	return num > current
}

func flipSign(sign *hclsyntax.Operation) *hclsyntax.Operation {
	flip := map[*hclsyntax.Operation]*hclsyntax.Operation{
		hclsyntax.OpGreaterThan:        hclsyntax.OpLessThan,
		hclsyntax.OpGreaterThanOrEqual: hclsyntax.OpLessThanOrEqual,
		hclsyntax.OpLessThan:           hclsyntax.OpGreaterThan,
		hclsyntax.OpLessThanOrEqual:    hclsyntax.OpGreaterThanOrEqual,
		hclsyntax.OpEqual:              hclsyntax.OpEqual,
	}
	// nil if sign is not in the map.
	newSign := flip[sign]

	return newSign
}

func walkIsOneOf(ex hcl.Expression, name string, enum *[]any) error {
	switch ex := ex.(type) {
	case *hclsyntax.BinaryOpExpr:
		switch ex.Op {
		case hclsyntax.OpLogicalOr: // ||
			err := walkIsOneOf(ex.LHS, name, enum)
			if err != nil {
				return err
			}

			return walkIsOneOf(ex.RHS, name, enum)
		case hclsyntax.OpEqual: // ==
			return parseEqualityExpression(ex, name, enum)
		default:
			return fmt.Errorf("operator is not || or ==")
		}
	case *hclsyntax.ParenthesesExpr:
		return walkIsOneOf(ex.Expression, name, enum)
	default:
		return fmt.Errorf("could not evaluate expression")
	}
}

func parseEqualityExpression(ex *hclsyntax.BinaryOpExpr, name string, enum *[]any) error {
	if isExpressionVarName(ex.RHS, name) {
		// swap the LHS and RHS
		ex.LHS, ex.RHS = ex.RHS, ex.LHS
	}

	if isExpressionVarName(ex.LHS, name) {
		object, err := reader.ExpressionToJSONObject(ex.RHS)
		if err != nil {
			return fmt.Errorf("value could not be converted to JSON: %w", err)
		}

		*enum = append(*enum, object)

		return nil
	}

	return fmt.Errorf("variable name not found in expression")
}
