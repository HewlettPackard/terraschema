// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package reader

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/HewlettPackard/terraschema/pkg/model"
)

const exampleMarker = "@example:"

type trailingComment struct {
	text      string
	startByte int
}

// fileComments indexes the single-line comments of one .tf file by line number.
// Comments are not part of the HCL AST, so they are recovered by re-lexing the
// raw source bytes.
type fileComments struct {
	// standalone holds comments that are the only content on their line.
	standalone map[int]string
	// trailing holds comments preceded by code on the same line.
	trailing map[int]trailingComment
}

func indexFileComments(file *hcl.File, fileName string) *fileComments {
	fc := &fileComments{
		standalone: make(map[int]string),
		trailing:   make(map[int]trailingComment),
	}

	// the file has already been parsed successfully, so lex diagnostics are not expected.
	tokens, d := hclsyntax.LexConfig(file.Bytes, fileName, hcl.InitialPos)
	if d.HasErrors() {
		return fc
	}

	codeLines := make(map[int]bool)
	for _, token := range tokens {
		switch token.Type {
		case hclsyntax.TokenComment, hclsyntax.TokenNewline, hclsyntax.TokenEOF:
			continue
		default:
			for line := token.Range.Start.Line; line <= token.Range.End.Line; line++ {
				codeLines[line] = true
			}
		}
	}

	for _, token := range tokens {
		if token.Type != hclsyntax.TokenComment {
			continue
		}
		raw := string(token.Bytes)
		// only single-line comments attach to attributes; /* */ blocks are ignored.
		if !strings.HasPrefix(raw, "#") && !strings.HasPrefix(raw, "//") {
			continue
		}
		line := token.Range.Start.Line
		text := stripCommentMarker(raw)
		if codeLines[line] {
			fc.trailing[line] = trailingComment{text: text, startByte: token.Range.Start.Byte}
		} else {
			fc.standalone[line] = text
		}
	}

	return fc
}

func stripCommentMarker(raw string) string {
	text := strings.TrimRight(raw, "\r\n")
	text = strings.TrimRight(text, " \t")
	if strings.HasPrefix(text, "//") {
		text = text[2:]
	} else {
		text = strings.TrimPrefix(text, "#")
	}

	return strings.TrimPrefix(text, " ")
}

// extractObjectAttributeComments finds comments attached to attributes inside
// object(...) type constraints within expr, keyed by dotted attribute path
// (e.g. "e", "e.a"). It returns nil if nothing attaches.
func extractObjectAttributeComments(expr hcl.Expression, fc *fileComments) map[string]model.AttributeMetadata {
	syntaxExpr, ok := expr.(hclsyntax.Expression)
	if !ok || fc == nil {
		return nil
	}

	out := make(map[string]model.AttributeMetadata)
	collectObjectComments(syntaxExpr, "", fc, out)
	if len(out) == 0 {
		return nil
	}

	return out
}

func collectObjectComments(expr hclsyntax.Expression, prefix string, fc *fileComments, out map[string]model.AttributeMetadata) {
	switch e := expr.(type) {
	case *hclsyntax.FunctionCallExpr:
		if e.Name == "object" && len(e.Args) == 1 {
			if objExpr, ok := e.Args[0].(*hclsyntax.ObjectConsExpr); ok {
				collectFromObjectCons(objExpr, prefix, fc, out)

				return
			}
		}
		// wrapper types (optional, list, set, map, tuple) add no path segment,
		// mirroring the JSON schema recursion.
		for _, arg := range e.Args {
			collectObjectComments(arg, prefix, fc, out)
		}
	case *hclsyntax.TupleConsExpr:
		for _, item := range e.Exprs {
			collectObjectComments(item, prefix, fc, out)
		}
	}
}

func collectFromObjectCons(
	obj *hclsyntax.ObjectConsExpr,
	prefix string,
	fc *fileComments,
	out map[string]model.AttributeMetadata,
) {
	// a trailing comment on a line where several attributes end belongs to the last one.
	lastEndOnLine := make(map[int]int)
	for _, item := range obj.Items {
		endRange := item.ValueExpr.Range()
		if endRange.End.Byte > lastEndOnLine[endRange.End.Line] {
			lastEndOnLine[endRange.End.Line] = endRange.End.Byte
		}
	}

	for _, item := range obj.Items {
		traversal, d := hcl.AbsTraversalForExpr(item.KeyExpr)
		if d.HasErrors() || len(traversal) == 0 {
			continue
		}
		name := traversal.RootName()

		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		description, examples := parseLeadingBlock(leadingCommentLines(item, obj, fc))
		if description == "" {
			if text, ok := trailingCommentText(item, fc, lastEndOnLine); ok {
				description = text
			}
		}
		if description != "" || len(examples) > 0 {
			out[path] = model.AttributeMetadata{Description: description, Examples: examples}
		}

		collectObjectComments(item.ValueExpr, path, fc, out)
	}
}

// leadingCommentLines collects the contiguous block of standalone comment lines
// directly above the attribute, in source order. The walk stops at the opening
// line of the enclosing object so that comments above the whole type expression
// (e.g. file headers) never attach to the first attribute.
func leadingCommentLines(item hclsyntax.ObjectConsItem, obj *hclsyntax.ObjectConsExpr, fc *fileComments) []string {
	var lines []string
	keyLine := item.KeyExpr.Range().Start.Line
	for line := keyLine - 1; line > obj.SrcRange.Start.Line; line-- {
		text, ok := fc.standalone[line]
		if !ok {
			break
		}
		lines = append([]string{text}, lines...)
	}

	return lines
}

func trailingCommentText(item hclsyntax.ObjectConsItem, fc *fileComments, lastEndOnLine map[int]int) (string, bool) {
	endRange := item.ValueExpr.Range()
	if endRange.End.Byte != lastEndOnLine[endRange.End.Line] {
		return "", false
	}
	tc, ok := fc.trailing[endRange.End.Line]
	if !ok || tc.startByte < endRange.End.Byte {
		return "", false
	}

	return tc.text, true
}

// parseLeadingBlock splits a leading comment block into a description and an
// optional single example: lines before the first "@example:" line form the
// description, and everything from that marker onward forms one example string.
func parseLeadingBlock(lines []string) (string, []string) {
	exampleIndex := -1
	for i, line := range lines {
		if strings.HasPrefix(line, exampleMarker) {
			exampleIndex = i

			break
		}
	}
	if exampleIndex == -1 {
		return strings.Join(lines, "\n"), nil
	}

	description := strings.Join(lines[:exampleIndex], "\n")
	first := strings.TrimPrefix(strings.TrimPrefix(lines[exampleIndex], exampleMarker), " ")
	exampleLines := append([]string{first}, lines[exampleIndex+1:]...)

	return description, []string{strings.Join(exampleLines, "\n")}
}
