package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"github.com/urfave/cli/v3"
	"goanna/haskell/parser"
	"goanna/haskell/rename"
)

func parseCommand(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("missing file argument")
	}
	filePath := cmd.Args().Get(0)
	return parser.ParseFile(filePath)
}

func sexpCommand(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("missing file argument")
	}
	filePath := cmd.Args().Get(0)
	return parser.PrintSexp(filePath)
}

func printASTCommand(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("missing file argument")
	}
	filePath := cmd.Args().Get(0)
	return parser.PrintASTFromFile(filePath)
}

func renameCommand(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 2 {
		return fmt.Errorf("usage: rename <dir> <module-name>")
	}
	dir := cmd.Args().Get(0)
	moduleName := cmd.Args().Get(1)

	// Recursively find all *.hs files
	var modules []*parser.Module
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".hs") {
			code, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			m := parser.Parse(code, path)
			if m != nil {
				modules = append(modules, m)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	rename.RenameAll(modules)

	// Print the AST of the requested module with canonicals
	for _, m := range modules {
		fmt.Printf("Module: %s\n", m.Name)
		if m.Name == moduleName {
			parser.PrintASTWithCanonicals(m)
			return nil
		}
	}
	return fmt.Errorf("module %q not found in %s", moduleName, dir)
}

func main() {
	cmd := &cli.Command{
		Name:  "goanna",
		Usage: "Haskell analysis and parsing tool",
		Commands: []*cli.Command{
			{
				Name:      "parse",
				Usage:     "Parse a Haskell file and print its AST",
				ArgsUsage: "<file.hs>",
				Action:    parseCommand,
			},
			{
				Name:      "sexp",
				Usage:     "Print the tree-sitter S-expression for debugging",
				ArgsUsage: "<file.hs>",
				Action:    sexpCommand,
			},
			{
				Name:      "ast",
				Usage:     "Print the AST in indented tree format with type, ID, and location",
				ArgsUsage: "<file.hs>",
				Action:    printASTCommand,
			},
			{
				Name:      "rename",
				Usage:     "Parse all *.hs files in a directory, rename identifiers, and print the AST with canonicals for the given module",
				ArgsUsage: "<dir> <module-name>",
				Action:    renameCommand,
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
