package main

import "github.com/polidog/reverhttp/internal/lsp"

func main() {
	srv := lsp.NewServer()
	srv.RunStdio()
}
