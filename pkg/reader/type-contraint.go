// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package reader

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
)

// GetTypeConstraint converts the expression into a type constraint and marshals it to JSON.
// Returns the JSON output as a []byte.
// More info on exactly how this works is here:
// https://pkg.go.dev/github.com/zclconf/go-cty@v1.14.4/cty#Type.MarshalJSON
func GetTypeConstraint(in hcl.Expression) (any, error) {
	if in == nil {
		return "any", nil
	}

	t, _, d := typeexpr.TypeConstraintWithDefaults(in) // do something with default value?
	if d.HasErrors() {
		return nil, fmt.Errorf("could not parse type constraint from expression: %w", d)
	}
	typeJSON, err := t.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("could not marshal constraint to JSON: %w", err)
	}

	var typeInterface any
	err = json.Unmarshal(typeJSON, &typeInterface)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal type from JSON: %w", err)
	}

	return typeInterface, nil
}
