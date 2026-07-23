// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package reader

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/stretchr/testify/require"

	"github.com/HewlettPackard/terraschema/pkg/model"
)

// objectCommentsFromSource parses HCL source containing a single variable block
// and returns the comment metadata extracted from its type expression.
func objectCommentsFromSource(t *testing.T, src string) map[string]model.AttributeMetadata {
	t.Helper()
	parser := hclparse.NewParser()
	file, d := parser.ParseHCL([]byte(src), "test.tf")
	require.False(t, d.HasErrors(), d)

	blocks, _, d := file.Body.PartialContent(fileSchema)
	require.False(t, d.HasErrors(), d)
	require.Len(t, blocks.Blocks, 1)

	comments := indexFileComments(file, "test.tf")
	_, translated, err := getTranslatedVariableFromBlock(blocks.Blocks[0], file, comments)
	require.NoError(t, err)

	return translated.ObjectComments
}

func TestExtractObjectAttributeComments(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		src      string
		expected map[string]model.AttributeMetadata
	}{
		"leading and trailing comments": {
			src: `
variable "v" {
	type = object({
		# This is A
		a = string
		b = number # This is B
		c = bool
	})
}`,
			expected: map[string]model.AttributeMetadata{
				"a": {Description: "This is A"},
				"b": {Description: "This is B"},
			},
		},
		"blank line breaks adjacency": {
			src: `
# file header comment
variable "v" {
	type = object({
		# detached comment

		a = string
	})
}`,
			expected: nil,
		},
		"double slash comments": {
			src: `
variable "v" {
	type = object({
		// This is A
		a = string
		b = number // This is B
	})
}`,
			expected: map[string]model.AttributeMetadata{
				"a": {Description: "This is A"},
				"b": {Description: "This is B"},
			},
		},
		"example annotation": {
			src: `
variable "v" {
	type = object({
		# This is A
		# @example: line one
		# line two
		a = string
		# @example: only an example
		b = number
	})
}`,
			expected: map[string]model.AttributeMetadata{
				"a": {Description: "This is A", Examples: []string{"line one\nline two"}},
				"b": {Examples: []string{"only an example"}},
			},
		},
		"nested objects in wrapper types": {
			src: `
variable "v" {
	type = object({
		a = optional(object({
			# nested in optional
			b = string
		}))
		c = list(object({
			d = number # nested in list
		}))
	})
}`,
			expected: map[string]model.AttributeMetadata{
				"a.b": {Description: "nested in optional"},
				"c.d": {Description: "nested in list"},
			},
		},
		"comment above one-line object does not attach": {
			src: `
variable "v" {
	# comment above the type
	type = object({ a = string })
}`,
			expected: nil,
		},
		"block comments are ignored": {
			src: `
variable "v" {
	type = object({
		/* block comment */
		a = string
	})
}`,
			expected: nil,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := objectCommentsFromSource(t, tc.src)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestParseLeadingBlock(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		lines               []string
		expectedDescription string
		expectedExamples    []string
	}{
		"no lines": {
			lines: nil,
		},
		"description only": {
			lines:               []string{"line one", "line two"},
			expectedDescription: "line one\nline two",
		},
		"description and example": {
			lines:               []string{"a description", "@example: first", "second"},
			expectedDescription: "a description",
			expectedExamples:    []string{"first\nsecond"},
		},
		"example only": {
			lines:            []string{"@example: just this"},
			expectedExamples: []string{"just this"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			description, examples := parseLeadingBlock(tc.lines)
			require.Equal(t, tc.expectedDescription, description)
			require.Equal(t, tc.expectedExamples, examples)
		})
	}
}
