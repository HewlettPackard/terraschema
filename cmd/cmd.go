// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/HewlettPackard/terraschema/pkg/jsonschema"
)

// wanted behaviour:
// - disallow-additional-properties: disallow additional properties in schema (default is false)
// - overwrite: overwrite an existing file (default is false for safety reasons)
// - stdout: suppress errors and output schema to stdout (generally not recommended)
// - output: file, default is ./schema.json. Allow creation of directories.
// - input: folder, default is .
// - allow-empty: if no variables are found, print empty schema and exit with 0

var (
	disallowAdditionalProperties bool
	overwrite                    bool
	allowEmpty                   bool
	requireAll                   bool
	outputStdOut                 bool
	output                       string
	input                        string

	errReturned error
)

// rootCmd is the base command for terraschema
var rootCmd = &cobra.Command{
	Use:     "terraschema",
	Example: "terraschema -i /path/to/module -o /path/to/schema.json",
	Short:   "Generate JSON schema from HCL Variable Blocks in a Terraform/OpenTofu module",
	Long: "TerraSchema is a CLI tool which scans Terraform configuration ('.tf') " +
		"files, parses a list of variables along with their type and validation rules, and converts " +
		"them to a schema which complies with JSON Schema Draft-07.\nThe default behaviour is to scan " +
		"the current directory and output a schema file called 'schema.json' in the same location. " +
		"\nFor more information see https://github.com/HewlettPackard/terraschema.",
	Run: runCommand,
	PostRun: func(cmd *cobra.Command, args []string) {
		if errReturned != nil {
			fmt.Printf("error: %v\n", errReturned)
			os.Exit(1)
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// TODO: implement
	rootCmd.Flags().BoolVar(&overwrite, "overwrite", false, "allow overwriting an existing file")
	// TODO: implement
	rootCmd.Flags().BoolVar(&outputStdOut, "stdout", false,
		"output JSON Schema content to stdout instead of a file and disable error output",
	)

	rootCmd.Flags().BoolVar(&disallowAdditionalProperties, "disallow-additional-properties", false,
		"set additionalProperties to false in the JSON Schema and in nested objects",
	)

	rootCmd.Flags().BoolVar(&allowEmpty, "allow-empty", false,
		"allow an empty JSON Schema if no variables are found",
	)

	rootCmd.Flags().BoolVar(&requireAll, "require-all", false,
		"set all variables to be 'required' in the JSON Schema, even if a default value is specified",
	)

	rootCmd.Flags().StringVarP(&input, "input", "i", ".",
		"input folder containing a Terraform module",
	)

	// TODO: implement
	rootCmd.Flags().StringVarP(&output, "output", "o", "schema.json",
		"output path for the JSON Schema file",
	)
}

func runCommand(cmd *cobra.Command, args []string) {
	path, err := filepath.Abs(input) // absolute path
	if err != nil {
		errReturned = fmt.Errorf("could not get absolute path for %q: %w", input, err)

		return
	}

	folder, err := os.Stat(path)
	if err != nil {
		errReturned = fmt.Errorf("could not access directory %q: %w", path, err)

		return
	}

	if !folder.IsDir() {
		errReturned = fmt.Errorf("input %q is not a directory", path)

		return
	}

	output, err := jsonschema.CreateSchema(path, jsonschema.CreateSchemaOptions{
		RequireAll:                requireAll,
		AllowAdditionalProperties: !disallowAdditionalProperties,
		AllowEmpty:                allowEmpty,
	})
	if err != nil {
		errReturned = fmt.Errorf("error creating schema: %w", err)

		return
	}

	jsonOutput, err := json.MarshalIndent(output, "", "    ")
	if err != nil {
		errReturned = fmt.Errorf("error marshalling schema: %w", err)

		return
	}

	fmt.Println(string(jsonOutput))
}
