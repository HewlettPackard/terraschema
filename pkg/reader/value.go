// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package reader

import (
	"encoding/json"

	"github.com/hashicorp/hcl/v2"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// ExpressionToJSONObject converts an HCL expression to an `any` type so that can be marshaled to JSON later.
func ExpressionToJSONObject(in hcl.Expression) (any, error) {
	if in == nil {
		return nil, nil //nolint:nilnil
	}

	v, d := in.Value(&hcl.EvalContext{})
	if d.HasErrors() {
		return nil, d
	}

	// convert the value to a simple JSON value, so that it can
	// be reliably marshaled to JSON. Then, unmarshal it to an
	// `any` type so that it can be passed around the code without
	// the additional type information that was unmarshaled by the
	// hcl package.
	simpleObject := ctyjson.SimpleJSONValue{Value: v}
	simpleAsJSON, err := simpleObject.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var out any
	err = json.Unmarshal(simpleAsJSON, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}
