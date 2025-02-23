package json

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/HewlettPackard/terraschema/pkg/model"
	"github.com/HewlettPackard/terraschema/pkg/reader"
)

type ExportVariablesOptions struct {
	AllowEmpty      bool
	DebugOut        bool
	SuppressLogging bool
	// this option is used to escape HTML characters in the output JSON. Since these schema files
	// aren't intended to be used directly in a web context, this is set to false by default.
	EscapeJSON      bool
	Indent          string
	IgnoreVariables []string
}

type MarshallableVariableBlock struct {
	model.TranslatedVariable
	EscapeHTML bool
	Indent     string
}

var _ json.Marshaler = MarshallableVariableBlock{}

type JSONVariableBlock struct {
	Default     *any                  `json:"default,omitempty"`
	Description *string               `json:"description,omitempty"`
	Nullable    *bool                 `json:"nullable,omitempty"`
	Sensitive   *bool                 `json:"sensitive,omitempty"`
	Validations []JSONValidationBlock `json:"validation,omitempty"`
	Type        *any                  `json:"type,omitempty"`
}

type JSONValidationBlock struct {
	Condition    string `json:"condition"`
	ErrorMessage string `json:"error_message"`
}

func ExportVariables(path string, options ExportVariablesOptions) (map[string]MarshallableVariableBlock, error) {
	jsonMap := make(map[string]MarshallableVariableBlock)
	varMap, err := reader.GetVarMap(path, options.DebugOut)
	if err != nil {
		if options.AllowEmpty && (errors.Is(err, reader.ErrFilesNotFound) || errors.Is(err, reader.ErrNoVariablesFound)) {
			if !options.SuppressLogging {
				fmt.Printf("Warning: directory %q: %v, creating empty variables file\n", path, err)
			}

			return jsonMap, nil
		} else {
			return jsonMap, fmt.Errorf("error reading tf files at %q: %w", path, err)
		}
	}

	for k, v := range varMap {
		if slices.Contains(options.IgnoreVariables, k) {
			continue
		}
		jsonMap[k] = MarshallableVariableBlock{v, options.EscapeJSON, options.Indent}
	}

	return jsonMap, nil
}

func (j MarshallableVariableBlock) MarshalJSON() ([]byte, error) {
	translatedBlock := JSONVariableBlock{
		Description: j.Variable.Description,
		Nullable:    j.Variable.Nullable,
		Sensitive:   j.Variable.Sensitive,
	}

	translatedType, err := reader.GetTypeConstraint(j.Variable.Type)
	if err != nil {
		return nil, fmt.Errorf("error marshalling type constraint: %w", err)
	}
	translatedBlock.Type = &translatedType

	translatedDefault, err := reader.ExpressionToJSONObject(j.Variable.Default)
	if err != nil {
		return nil, fmt.Errorf("error marshalling default expression: %w", err)
	}
	translatedBlock.Default = &translatedDefault

	if len(j.Variable.Validations) != 0 {
		if len(j.ConditionsAsString) != len(j.Variable.Validations) {
			return nil, errors.New("mismatch between the number of validation blocks and conditions")
		}
		translatedBlock.Validations = make([]JSONValidationBlock, len(j.Variable.Validations))
		for i := range translatedBlock.Validations {
			translatedBlock.Validations[i] = JSONValidationBlock{
				Condition:    j.ConditionsAsString[i],
				ErrorMessage: j.Variable.Validations[i].ErrorMessage,
			}
		}
	}

	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(j.EscapeHTML)
	encoder.SetIndent("", j.Indent)
	err = encoder.Encode(translatedBlock)
	if err != nil {
		return nil, fmt.Errorf("error marshalling variable block: %w", err)
	}

	return buffer.Bytes(), nil
}
