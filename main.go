package main

import (
	"context"
	"fmt"
	"os"
	"github.com/urfave/cli/v3"
	"goanna/haskell/parser"
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
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
