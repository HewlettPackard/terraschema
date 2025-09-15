// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package model

import (
	"github.com/hashicorp/hcl/v2"
)

// VariableBlock represents a Terraform variable block. It contains all fields that can be present in a variable block.
// The variable name is stored separately.
type VariableBlock struct {
	Default     hcl.Expression `hcl:"default,optional"`
	Description *string        `hcl:"description,optional"`
	Nullable    *bool          `hcl:"nullable,optional"`
	// Sensitive is ignored.
	Sensitive *bool `hcl:"sensitive,optional"`
	// Validations blocks can be used to add extra rules to the JSON schema, as long as their conditions
	// are written in a certain format.
	Validations []ValidationBlock `hcl:"validation,block"`
	Type        hcl.Expression    `hcl:"type,optional"`

	// ignore other attributes (triggers partial decoding)
	Other hcl.Body `hcl:",remain"`
}

type ValidationBlock struct {
	Condition hcl.Expression `hcl:"condition,attr"`

	// ignore other attributes (triggers partial decoding)
	Other hcl.Body `hcl:",remain"`
}

// TranslatedVariable contains the Variable struct, as well as some extra information that can be used for debugging.
// This is done here because it can be difficult to extract this information from pure hcl.Expressions as present in
// the Variable struct without further context. Required is used internally, and the others are for debugging.
type TranslatedVariable struct {
	// if the variable has validation blocks, this stores their conditions as a string. This is useful for debugging.
	ConditionsAsString []string
	// DefaultAsString is the default value of the variable, as a string. This is useful for debugging complex default values.
	DefaultAsString *string
	// TypeAsString is the type of the variable, as a string. This is useful for debugging complex types, such as objects.
	// A nil value for type implies no type has been specified, so the terraform code accepts "any" type.
	TypeAsString *string
	// Required is true if and only if the variable has no default value.
	Required bool
	// The variable block used to generate the other fields in this struct.
	Variable VariableBlock
}
