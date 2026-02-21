package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/polidog/reverhttp/internal/gen"
	"github.com/polidog/reverhttp/internal/ir"
	"github.com/polidog/reverhttp/internal/lexer"
	"github.com/polidog/reverhttp/internal/parser"
)

func main() {
	output := flag.String("o", "", "output file (default: stdout)")
	indent := flag.Bool("indent", true, "indent JSON output")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: reverc [options] <file.rever> ...\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Parse and merge all files
	root := &ir.Root{
		Version: "0.1",
	}

	hasErrors := false
	for _, file := range args {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		l := lexer.New(string(data), file)
		p := parser.New(l)
		ast := p.ParseFile()

		if errs := p.Errors(); len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintln(os.Stderr, e)
			}
			hasErrors = true
			continue
		}

		fileIR := gen.Generate(ast)
		mergeIR(root, fileIR)
	}

	if hasErrors {
		os.Exit(1)
	}

	var jsonData []byte
	var err error
	if *indent {
		jsonData, err = json.MarshalIndent(root, "", "  ")
	} else {
		jsonData, err = json.Marshal(root)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	jsonData = append(jsonData, '\n')

	if *output != "" {
		if err := os.WriteFile(*output, jsonData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
			os.Exit(1)
		}
	} else {
		os.Stdout.Write(jsonData)
	}
}

func mergeIR(dst, src *ir.Root) {
	// Merge imports
	if len(src.Imports) > 0 {
		if dst.Imports == nil {
			dst.Imports = make(map[string]*ir.Import)
		}
		for k, v := range src.Imports {
			dst.Imports[k] = v
		}
	}

	// Merge types
	if len(src.Types) > 0 {
		if dst.Types == nil {
			dst.Types = make(map[string]ir.TypeFields)
		}
		for k, v := range src.Types {
			dst.Types[k] = v
		}
	}

	// Merge defaults (last one wins)
	if src.Defaults != nil {
		dst.Defaults = src.Defaults
	}

	// Append routes
	dst.Routes = append(dst.Routes, src.Routes...)
}
