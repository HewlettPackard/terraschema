// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	tsjson "github.com/HewlettPackard/terraschema/pkg/json"
	"github.com/HewlettPackard/terraschema/pkg/jsonschema"
)

var (
	disallowAdditionalProperties bool
	overwrite                    bool
	allowEmpty                   bool
	requireAll                   bool
	outputStdOut                 bool
	nullableAll                  bool
	inputPath                    string
	outputPath                   string
	debugOut                     bool
	exportVariables              bool
	escapeJSON                   bool
	ignoreVariables              []string
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
	PreRunE:      preRunCommand,
	RunE:         runCommand,
	SilenceUsage: true,
}

// Execute command with the following flags:
//   - disallow-additional-properties: disallow additional properties in schema (default is false)
//   - overwrite: overwrite an existing file (default is false for safety reasons)
//   - stdout: suppress errors and output schema to stdout (generally not recommended)
//   - output: file, default is ./schema.json. Allow creation of directories.
//   - input: folder, default is .
//   - allow-empty: if no variables are found, print empty schema and exit with 0
//   - require-all: require all variables to be present in the schema, even if a default value is specified
//   - debug: output logs to track variables retrieved from each file, and get more verbose logs from custom validation rules
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().BoolVar(&disallowAdditionalProperties, "disallow-additional-properties", false,
		"set additionalProperties to false in the JSON Schema and in nested objects",
	)

	rootCmd.Flags().BoolVar(&allowEmpty, "allow-empty", false,
		"allow an empty JSON Schema if no variables are found",
	)

	rootCmd.Flags().BoolVar(&requireAll, "require-all", false,
		"set all variables to be 'required' in the JSON Schema, even if a default\n"+
			"value is specified",
	)

	rootCmd.Flags().StringVarP(&inputPath, "input", "i", ".",
		"input folder containing a Terraform module",
	)

	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "schema.json",
		"output path for the JSON Schema file",
	)

	rootCmd.Flags().BoolVar(&overwrite, "overwrite", false,
		"allow overwriting an existing file",
	)

	rootCmd.Flags().BoolVar(&outputStdOut, "stdout", false,
		"output JSON Schema content to stdout instead of a file and disable any other logging\n"+
			"unless an error occurs. Overrides 'debug' and 'output.",
	)

	rootCmd.Flags().BoolVar(&debugOut, "debug", false,
		"output debug logs, may useful for troubleshooting issues relating to translating\n"+
			"validation rules. Does not work with --stdout",
	)

	rootCmd.Flags().BoolVar(&nullableAll, "nullable-all", false,
		"make all variables nullable unless nullable set to false explicitly, to make behavior consistent with Terraform",
	)

	rootCmd.Flags().BoolVar(&exportVariables, "export-variables", false,
		"export variables to a JSON file or stdout instead of creating a schema",
	)

	rootCmd.Flags().BoolVar(&escapeJSON, "escape-json", false,
		"escape JSON special characters in the output, so that the Schema can be used in a\n"+
			"web context",
	)
	rootCmd.Flags().StringSliceVar(&ignoreVariables, "ignore-variable", []string{},
		"ignore variable by name when generating schema or exporting variables, repeating this argument allows you to ignore multiple variables",
	)

	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		_ = rootCmd.Usage()

		return err
	})
}

func preRunCommand(cmd *cobra.Command, args []string) error {
	err := inputFileChecks()
	if err != nil {
		return err
	}
	if !outputStdOut {
		return outputFileChecks()
	}

	return nil
}

func inputFileChecks() error {
	_, err := filepath.Abs(inputPath) // absolute path
	if err != nil {
		return fmt.Errorf("could not get absolute path for %q: %w", inputPath, err)
	}

	folder, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("could not access directory %q: %w", inputPath, err)
	}

	if !folder.IsDir() {
		return fmt.Errorf("input path %q is not a directory", inputPath)
	}

	return nil
}

func outputFileChecks() error {
	_, err := filepath.Abs(outputPath) // absolute path
	if err != nil {
		return fmt.Errorf("could not get absolute path for %q: %w", outputPath, err)
	}

	outputFile, err := os.Stat(outputPath)
	if err == nil {
		if overwrite {
			if outputFile.IsDir() {
				return fmt.Errorf(
					"output path %q is an existing directory, please specify a file path",
					outputPath,
				)
			}
		} else {
			return fmt.Errorf("output path %q already exists, use --overwrite to overwrite", outputPath)
		}
	}

	if !strings.HasSuffix(outputPath, ".json") {
		fmt.Printf("Warning: output path %q does not have a .json extension, continuing\n", outputPath)
	}

	return nil
}

func runCommand(cmd *cobra.Command, args []string) error {
	var outputMap any
	var err error

	jsonIndent := "\t"

	if exportVariables {
		outputMap, err = tsjson.ExportVariables(inputPath, tsjson.ExportVariablesOptions{
			AllowEmpty:      allowEmpty,
			SuppressLogging: outputStdOut,
			DebugOut:        debugOut && !outputStdOut,
			EscapeJSON:      escapeJSON,
			Indent:          jsonIndent,
			IgnoreVariables: ignoreVariables,
		})
		if err != nil {
			return fmt.Errorf("error exporting variables: %w", err)
		}
	} else {
		outputMap, err = jsonschema.CreateSchema(inputPath, jsonschema.CreateSchemaOptions{
			RequireAll:                requireAll,
			AllowAdditionalProperties: !disallowAdditionalProperties,
			AllowEmpty:                allowEmpty,
			DebugOut:                  debugOut && !outputStdOut,
			SuppressLogging:           outputStdOut,
			NullableAll:               nullableAll,
			IgnoreVariables:           ignoreVariables,
		})
		if err != nil {
			return fmt.Errorf("error creating schema: %w", err)
		}
	}

	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(escapeJSON)
	encoder.SetIndent("", jsonIndent)

	err = encoder.Encode(outputMap)
	if err != nil {
		return fmt.Errorf("error marshalling schema: %w", err)
	}
	jsonOutput := buffer.Bytes()

	if outputStdOut {
		fmt.Println(string(jsonOutput))

		return nil
	}

	// create folder path for output file if it doesn't exist
	err = os.MkdirAll(filepath.Dir(outputPath), 0o755)
	if err != nil {
		return fmt.Errorf("error creating folder for %q: %w", outputPath, err)
	}

	// Create a file with 644 file permissions. If this causes issues, we can use 600 instead later.
	//nolint:gosec
	err = os.WriteFile(outputPath, jsonOutput, 0o644)
	if err != nil {
		return fmt.Errorf("error writing schema to %q: %w", outputPath, err)
	}
	fmt.Printf("Info: schema written to %q\n", outputPath)

	return nil
}
