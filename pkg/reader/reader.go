// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package reader

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"

	"github.com/HewlettPackard/terraschema/pkg/model"
)

var fileSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "variable",
			LabelNames: []string{"name"},
		},
	},
}

var (
	ErrFilesNotFound    = fmt.Errorf("no .tf files found")
	ErrNoVariablesFound = fmt.Errorf("tf files don't contain any variables")
)

// GetVarMap reads all .tf files in a directory and returns a map of variable names to their translated values.
// For the purpose of this application, all that matters is the model.VariableBlock contained in this, which
// contains a direct unmarshal of the block itself using the hcl package. The rest of the information is for
// debugging purposes, and to simplify the process of deciding if a variable is 'required' later. Note: in 'strict'
// mode, all variables are required, regardless of whether they have a default value or not.
func GetVarMap(path string, debugOut bool) (map[string]model.TranslatedVariable, error) {
	// read all tf files in directory
	files, err := filepath.Glob(filepath.Join(path, "*.tf"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, ErrFilesNotFound
	}

	if debugOut {
		fmt.Printf("Debug: found the following files in %q:\n", path)
	}

	parser := hclparse.NewParser()

	varMap := make(map[string]model.TranslatedVariable)
	for _, fileName := range files {
		if debugOut {
			fmt.Printf("\t%q, with variable(s):\n", fileName)
		}

		file, d := parser.ParseHCLFile(fileName)
		if d.HasErrors() {
			return nil, d
		}

		blocks, _, d := file.Body.PartialContent(fileSchema)
		if d.HasErrors() {
			return nil, d
		}
		for _, block := range blocks.Blocks {
			name, translated, err := getTranslatedVariableFromBlock(block, file)
			if err != nil {
				return nil, fmt.Errorf("error getting parsing %q: %w", name, err)
			}
			varMap[name] = translated

			if debugOut {
				fmt.Printf("\t\t%s\n", name)
			}
		}
	}

	if len(varMap) == 0 {
		return nil, ErrNoVariablesFound
	}

	return varMap, nil
}

func getTranslatedVariableFromBlock(block *hcl.Block, file *hcl.File) (string, model.TranslatedVariable, error) {
	name := block.Labels[0]
	variable := model.VariableBlock{}
	d := gohcl.DecodeBody(block.Body, nil, &variable)
	if d.HasErrors() {
		return name, model.TranslatedVariable{}, d
	}

	variable.Default = filterMissingExpression(variable.Default)
	variable.Type = filterMissingExpression(variable.Type)

	out := model.TranslatedVariable{Variable: variable, Required: true}

	// Get type, default, and condition as strings and add them to the translated variable struct.
	// This is to make the code easier to debug, since hcl.Expressions are difficult to read out of context.

	// check if 'default' exists
	if variable.Default != nil {
		defaultAsString := printToString(variable.Default, file)
		out.DefaultAsString = &defaultAsString
		out.Required = false
	}

	// check if 'type' exists
	if variable.Type != nil {
		typeAsString := printToString(variable.Type, file)
		out.TypeAsString = &typeAsString
	}

	// plaintext print all condition expressions into the ConditionsAsString field.
	out.ConditionsAsString = make([]string, len(variable.Validations))
	for i, validation := range variable.Validations {
		out.ConditionsAsString[i] = printToString(validation.Condition, file)
	}

	return name, out, nil
}

func filterMissingExpression(in hcl.Expression) hcl.Expression {
	// if the start and the end range are the same, this means the field is not
	// real, so it can be removed.
	if in.Range().Start.Byte == in.Range().End.Byte {
		return nil
	}

	return in
}

func printToString(in hcl.Expression, f *hcl.File) string {
	out := string(in.Range().SliceBytes(f.Bytes))

	return out
}
