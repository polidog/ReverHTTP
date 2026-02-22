package lsp

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

const serverName = "rever-lsp"
const serverVersion = "0.1.0"

func NewServer() *server.Server {
	store := NewDocumentStore()
	handler := &protocol.Handler{}

	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
		capabilities := handler.CreateServerCapabilities()
		capabilities.TextDocumentSync = protocol.TextDocumentSyncKindFull
		capabilities.CompletionProvider = &protocol.CompletionOptions{}

		version := serverVersion
		return protocol.InitializeResult{
			Capabilities: capabilities,
			ServerInfo: &protocol.InitializeResultServerInfo{
				Name:    serverName,
				Version: &version,
			},
		}, nil
	}

	handler.Initialized = func(context *glsp.Context, params *protocol.InitializedParams) error {
		return nil
	}

	handler.Shutdown = func(context *glsp.Context) error {
		return nil
	}

	handler.TextDocumentDidOpen = func(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
		uri := params.TextDocument.URI
		store.Open(uri, params.TextDocument.Text)
		publishDiagnostics(context, uri, store.Get(uri))
		return nil
	}

	handler.TextDocumentDidChange = func(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
		uri := params.TextDocument.URI
		for _, change := range params.ContentChanges {
			if c, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
				store.Update(uri, c.Text)
			}
		}
		publishDiagnostics(context, uri, store.Get(uri))
		return nil
	}

	handler.TextDocumentDidClose = func(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
		uri := params.TextDocument.URI
		store.Close(uri)
		context.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: []protocol.Diagnostic{},
		})
		return nil
	}

	handler.TextDocumentCompletion = func(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
		uri := params.TextDocument.URI
		text := store.Get(uri)
		return Complete(text, params.Position), nil
	}

	return server.NewServer(handler, serverName, false)
}
