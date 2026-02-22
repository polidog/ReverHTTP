package lsp

import (
	"regexp"
	"strconv"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/polidog/reverhttp/internal/lexer"
	"github.com/polidog/reverhttp/internal/parser"
)

// errorPattern matches parser error format: "file:line:column: message"
var errorPattern = regexp.MustCompile(`^[^:]+:(\d+):(\d+): (.+)$`)

func publishDiagnostics(ctx *glsp.Context, uri, text string) {
	diags := diagnose(text)
	ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	})
}

func diagnose(text string) []protocol.Diagnostic {
	l := lexer.New(text, "buffer")
	p := parser.New(l)
	p.ParseFile()

	errs := p.Errors()
	diags := make([]protocol.Diagnostic, 0, len(errs))
	source := serverName
	severity := protocol.DiagnosticSeverityError

	for _, e := range errs {
		m := errorPattern.FindStringSubmatch(e)
		if m == nil {
			diags = append(diags, protocol.Diagnostic{
				Range:    protocol.Range{},
				Severity: &severity,
				Source:   &source,
				Message:  e,
			})
			continue
		}

		line, _ := strconv.Atoi(m[1])
		col, _ := strconv.Atoi(m[2])
		// Parser positions are 1-based; LSP is 0-based.
		pos := protocol.Position{
			Line:      uint32(line - 1),
			Character: uint32(col - 1),
		}

		diags = append(diags, protocol.Diagnostic{
			Range:    protocol.Range{Start: pos, End: pos},
			Severity: &severity,
			Source:   &source,
			Message:  m[3],
		})
	}

	return diags
}
