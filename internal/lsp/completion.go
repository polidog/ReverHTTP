package lsp

import (
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func Complete(text string, pos protocol.Position) []protocol.CompletionItem {
	ctx := detectContext(text, pos)

	var items []protocol.CompletionItem
	kind := protocol.CompletionItemKindKeyword

	switch ctx {
	case contextTopLevel:
		for _, kw := range topLevelKeywords {
			items = append(items, protocol.CompletionItem{Label: kw, Kind: &kind})
		}
	case contextPipeline:
		for _, kw := range pipelineSteps {
			items = append(items, protocol.CompletionItem{Label: kw, Kind: &kind})
		}
	case contextDefaults:
		for _, kw := range directiveKeywords {
			items = append(items, protocol.CompletionItem{Label: kw, Kind: &kind})
		}
	case contextValidate:
		for _, kw := range validateKeywords {
			items = append(items, protocol.CompletionItem{Label: kw, Kind: &kind})
		}
	}

	return items
}

type completionContext int

const (
	contextTopLevel completionContext = iota
	contextPipeline
	contextDefaults
	contextValidate
)

var topLevelKeywords = []string{
	"import", "type", "defaults",
	"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS",
}

var pipelineSteps = []string{
	"input", "validate", "transform", "guard", "match", "respond",
}

var directiveKeywords = []string{
	"cache", "cors", "auth",
}

var validateKeywords = []string{
	"int", "string", "bool", "float", "datetime",
	"min", "max", "format",
}

func detectContext(text string, pos protocol.Position) completionContext {
	lines := strings.Split(text, "\n")
	lineIdx := int(pos.Line)
	if lineIdx >= len(lines) {
		return contextTopLevel
	}

	// Search backwards from the cursor line to determine context.
	for i := lineIdx; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		if strings.HasPrefix(line, "|>") {
			// Check if we're inside validate(...)
			rest := strings.TrimPrefix(line, "|>")
			rest = strings.TrimSpace(rest)
			if strings.HasPrefix(rest, "validate") {
				return contextValidate
			}
			return contextPipeline
		}

		if strings.HasPrefix(line, "defaults") {
			return contextDefaults
		}

		// If we hit a route definition, we're in pipeline context.
		for _, method := range []string{"GET ", "POST ", "PUT ", "DELETE ", "PATCH ", "HEAD ", "OPTIONS "} {
			if strings.HasPrefix(line, method) {
				return contextPipeline
			}
		}
	}

	return contextTopLevel
}
