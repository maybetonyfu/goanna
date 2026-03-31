package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"github.com/urfave/cli/v3"
	"goanna/haskell/meta"
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

// parseAndRename walks dir for *.hs files, parses them with a shared node ID
// counter (ensuring globally unique IDs), and renames all identifiers.
func parseAndRename(dir string) ([]*parser.Module, error) {
	var modules []*parser.Module
	counter := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".hs") {
			code, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			moduleName := parser.GuessModuleName(path, dir)
			m := parser.ParseWithCounter(code, moduleName, &counter)
			if m != nil {
				modules = append(modules, m)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	rename.RenameAll(modules)
	return modules, nil
}

func renameCommand(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("usage: rename <dir>")
	}
	modules, err := parseAndRename(cmd.Args().Get(0))
	if err != nil {
		return err
	}
	for _, m := range modules {
		fmt.Printf("\n======== %s ========\n\n", m.Name)
		parser.PrintASTWithCanonicals(m)
	}
	return nil
}

func declsCommand(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("usage: decls <dir>")
	}
	modules, err := parseAndRename(cmd.Args().Get(0))
	if err != nil {
		return err
	}

	graph := meta.GetDeclGraph(modules)

	keys := make([]string, 0, len(graph))
	for k := range graph {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		deps := graph[k]
		fmt.Printf("%s -> [%s]\n", k, strings.Join(deps, ", "))
	}
  return nil
}

func typeVarClassesCommand(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("usage: typevarclasses <dir>")
	}
	modules, err := parseAndRename(cmd.Args().Get(0))
	if err != nil {
		return err
	}

	superclasses := meta.GetClassSuperclasses(modules)
	tvClasses := meta.GetTypeVarClasses(modules, superclasses)

	keys := make([]string, 0, len(tvClasses))
	for k := range tvClasses {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		vals := tvClasses[k]
		sort.Strings(vals)
		fmt.Printf("%s -> [%s]\n", k, strings.Join(vals, ", "))
	}
	return nil
}

func typeVarsCommand(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("usage: typevars <dir>")
	}
	modules, err := parseAndRename(cmd.Args().Get(0))
	if err != nil {
		return err
	}

	declTypeVars := meta.GetDeclTypeVars(modules)

	keys := make([]string, 0, len(declTypeVars))
	for k := range declTypeVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("%s -> [%s]\n", k, strings.Join(declTypeVars[k], ", "))
	}
	return nil
}

func classesCommand(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("usage: classes <dir>")
	}
	modules, err := parseAndRename(cmd.Args().Get(0))
	if err != nil {
		return err
	}

	superclasses := meta.GetClassSuperclasses(modules)

	keys := make([]string, 0, len(superclasses))
	for k := range superclasses {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("%s -> [%s]\n", k, strings.Join(superclasses[k], ", "))
	}
	return nil
}

func paramsCommand(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("usage: params <dir>")
	}
	modules, err := parseAndRename(cmd.Args().Get(0))
	if err != nil {
		return err
	}

	params := meta.GetDeclParams(modules)
	graph := meta.GetDeclGraph(modules)
	inherited := meta.InheritParams(params, graph)

	keys := make([]string, 0, len(inherited))
	for k := range inherited {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("%s -> [%s]\n", k, strings.Join(inherited[k], ", "))
	}
	return nil
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
				Usage:     "Parse all *.hs files in a directory, rename identifiers, and print all module ASTs with canonicals",
				ArgsUsage: "<dir>",
				Action:    renameCommand,
			},
			{
				Name:      "decls",
				Usage:     "Parse all *.hs files in a directory, rename identifiers, and print the declaration dependency graph",
				ArgsUsage: "<dir>",
				Action:    declsCommand,
			},
			{
				Name:      "params",
				Usage:     "Parse all *.hs files in a directory, rename identifiers, and print parameters for each declaration",
				ArgsUsage: "<dir>",
				Action:    paramsCommand,
			},
			{
				Name:      "classes",
				Usage:     "Parse all *.hs files in a directory and print each typeclass with all its superclasses (transitive)",
				ArgsUsage: "<dir>",
				Action:    classesCommand,
			},
			{
				Name:      "typevars",
				Usage:     "Parse all *.hs files in a directory and print the type variables for each declaration",
				ArgsUsage: "<dir>",
				Action:    typeVarsCommand,
			},
			{
				Name:      "typevarclasses",
				Usage:     "Parse all *.hs files in a directory and print the typeclass constraints for each type variable (with superclasses)",
				ArgsUsage: "<dir>",
				Action:    typeVarClassesCommand,
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
