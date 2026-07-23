// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package reader

import (
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/HewlettPackard/terraschema/pkg/model"
)

const (
	exampleMarker    = "@example:"
	deprecatedMarker = "@deprecated"
)

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
// (e.g. "e", "e.a"; tuple elements add their index, e.g. "t.0.name"). It
// returns nil if nothing attaches.
func extractObjectAttributeComments(expr hcl.Expression, fc *fileComments) map[string]model.AttributeMetadata {
	syntaxExpr, ok := expr.(hclsyntax.Expression)
	if !ok || fc == nil {
		return nil
	}

	w := &commentWalker{
		fc:         fc,
		lineMaxEnd: make(map[int]int),
		out:        make(map[string]model.AttributeMetadata),
	}
	w.recordLineEnds(syntaxExpr)
	w.collect(syntaxExpr, "")
	if len(w.out) == 0 {
		return nil
	}

	return w.out
}

// commentWalker carries the state of one walk over a type expression.
type commentWalker struct {
	fc *fileComments
	// lineMaxEnd holds, per line, the greatest end byte of any expression that
	// ends on that line. A trailing comment attaches to an attribute only if the
	// attribute's value is the last expression ending before it, so a comment
	// after the closing brackets of a nested construct never attaches to the
	// attributes inside it.
	lineMaxEnd map[int]int
	out        map[string]model.AttributeMetadata
}

func (w *commentWalker) recordLineEnds(expr hclsyntax.Expression) {
	endPos := expr.Range().End
	if endPos.Byte > w.lineMaxEnd[endPos.Line] {
		w.lineMaxEnd[endPos.Line] = endPos.Byte
	}
	switch e := expr.(type) {
	case *hclsyntax.FunctionCallExpr:
		for _, arg := range e.Args {
			w.recordLineEnds(arg)
		}
	case *hclsyntax.TupleConsExpr:
		for _, element := range e.Exprs {
			w.recordLineEnds(element)
		}
	case *hclsyntax.ObjectConsExpr:
		for _, item := range e.Items {
			w.recordLineEnds(item.ValueExpr)
		}
	}
}

func (w *commentWalker) collect(expr hclsyntax.Expression, prefix string) {
	switch e := expr.(type) {
	case *hclsyntax.FunctionCallExpr:
		if e.Name == "object" && len(e.Args) == 1 {
			if objExpr, ok := e.Args[0].(*hclsyntax.ObjectConsExpr); ok {
				w.collectObject(objExpr, prefix)

				return
			}
		}
		// wrapper types (optional, list, set, map) add no path segment,
		// mirroring the JSON schema recursion.
		for _, arg := range e.Args {
			w.collect(arg, prefix)
		}
	case *hclsyntax.TupleConsExpr:
		// tuple elements are addressed by index, as in the JSON schema recursion,
		// so same-named attributes in different elements do not collide.
		for i, element := range e.Exprs {
			w.collect(element, joinPath(prefix, strconv.Itoa(i)))
		}
	}
}

func (w *commentWalker) collectObject(obj *hclsyntax.ObjectConsExpr, prefix string) {
	for _, item := range obj.Items {
		traversal, d := hcl.AbsTraversalForExpr(item.KeyExpr)
		if d.HasErrors() || len(traversal) == 0 {
			continue
		}
		path := joinPath(prefix, traversal.RootName())

		meta := parseLeadingBlock(w.leadingCommentLines(item, obj))
		if meta.Description == "" {
			if text, ok := w.trailingCommentText(item); ok {
				meta.Description = text
			}
		}
		if meta.Description != "" || len(meta.Examples) > 0 || meta.Deprecated {
			w.out[path] = meta
		}

		w.collect(item.ValueExpr, path)
	}
}

func joinPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}

	return prefix + "." + segment
}

// leadingCommentLines collects the contiguous block of standalone comment lines
// directly above the attribute, in source order. The walk stops at the opening
// line of the enclosing object so that comments above the whole type expression
// (e.g. file headers) never attach to the first attribute.
func (w *commentWalker) leadingCommentLines(item hclsyntax.ObjectConsItem, obj *hclsyntax.ObjectConsExpr) []string {
	var lines []string
	keyLine := item.KeyExpr.Range().Start.Line
	for line := keyLine - 1; line > obj.SrcRange.Start.Line; line-- {
		text, ok := w.fc.standalone[line]
		if !ok {
			break
		}
		lines = append(lines, text)
	}
	slices.Reverse(lines)

	return lines
}

func (w *commentWalker) trailingCommentText(item hclsyntax.ObjectConsItem) (string, bool) {
	endPos := item.ValueExpr.Range().End
	if endPos.Byte != w.lineMaxEnd[endPos.Line] {
		return "", false
	}
	tc, ok := w.fc.trailing[endPos.Line]
	if !ok || tc.startByte < endPos.Byte {
		return "", false
	}

	return tc.text, true
}

// parseLeadingBlock interprets a leading comment block as a description with
// optional annotations. Each "@example:" line starts a new example, which
// collects the following lines until the next annotation. An "@deprecated"
// line marks the attribute deprecated; any text after it (and any later plain
// lines) continues the description.
func parseLeadingBlock(lines []string) model.AttributeMetadata {
	var meta model.AttributeMetadata
	var description []string
	var examples [][]string
	// index of the example currently being collected, or -1 for the description.
	target := -1
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, exampleMarker):
			examples = append(examples, annotationText(line, exampleMarker))
			target = len(examples) - 1
		case isDeprecatedLine(line):
			meta.Deprecated = true
			description = append(description, annotationText(line, deprecatedMarker)...)
			target = -1
		case target == -1:
			description = append(description, line)
		default:
			examples[target] = append(examples[target], line)
		}
	}

	meta.Description = strings.Join(description, "\n")
	for _, example := range examples {
		meta.Examples = append(meta.Examples, strings.Join(example, "\n"))
	}

	return meta
}

// isDeprecatedLine reports whether the line is an @deprecated annotation. The
// marker must be the whole word: longer words such as "@deprecated_in_v3" are
// plain description text.
func isDeprecatedLine(line string) bool {
	rest, found := strings.CutPrefix(line, deprecatedMarker)
	if !found {
		return false
	}

	return rest == "" || strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, "\t")
}

// annotationText returns the text of an annotation line after its marker as a
// slice of zero or one lines, so that markers on a line of their own do not
// contribute an empty line to the collected block.
func annotationText(line, marker string) []string {
	text := strings.TrimPrefix(line, marker)
	text = strings.TrimPrefix(text, ":")
	text = strings.TrimPrefix(text, " ")
	if text == "" {
		return nil
	}

	return []string{text}
}
