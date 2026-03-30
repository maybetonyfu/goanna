package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	treesitter "github.com/tree-sitter/go-tree-sitter"
	treesitterhaskell "github.com/tree-sitter/tree-sitter-haskell/bindings/go"
)

var fixity = map[string]int{
	".":   9,
	"$":   0,
	"==":  4,
	"/=":  4,
	"<":   4,
	">":   4,
	"<=":  4,
	">=":  4,
	"&&":  3,
	"||":  2,
	"++":  5,
	"<$>": 4,
	"<*>": 4,
	">>":  1,
	">>=": 1,
	"+":   6,
	"-":   6,
	"*":   7,
	"/":   7,
}

var associativity = map[string]string{
	".":   "r",
	"$":   "r",
	"==":  "l",
	"/=":  "l",
	"<":   "l",
	">":   "l",
	"<=":  "l",
	">=":  "l",
	"&&":  "r",
	"||":  "r",
	"++":  "r",
	"<$>": "l",
	"<*>": "l",
	">>":  "l",
	">>=": "l",
	"+":   "l",
	"-":   "l",
	"*":   "l",
	"/":   "l",
}

type parseEnv struct {
	counter       *int
	source        []byte
	cursor        *treesitter.TreeCursor
	fixity        map[string]int
	associativity map[string]string
}

func (pe parseEnv) id() int {
	id := *pe.counter
	*pe.counter += 1
	return id
}

func (pe parseEnv) loc(node *treesitter.Node) Loc {
	nodeRange := node.Range()
	return Loc{
		fromLine: int(nodeRange.StartPoint.Row),
		fromCol:  int(nodeRange.StartPoint.Column),
		toLine:   int(nodeRange.EndPoint.Row),
		toCol:    int(nodeRange.EndPoint.Column),
	}
}

func (pe parseEnv) node(node *treesitter.Node) Node {
	n := Node{
		id:  pe.id(),
		loc: pe.loc(node),
	}
	return n
}

func (pe parseEnv) text(node *treesitter.Node) string {
	return node.Utf8Text(pe.source)
}

func (pe parseEnv) children(node *treesitter.Node, path string) []treesitter.Node {
	paths := strings.Split(path, ":")
	var currentNode = node
	for i, path := range paths {
		switch {
		case currentNode == nil:
			break

		case i != len(paths)-1:
			currentNode = currentNode.ChildByFieldName(path)
		case path == "*":
			children := make([]treesitter.Node, 0)
			for _, node := range currentNode.NamedChildren(pe.cursor) {
				if node.Kind() == "comment" {
					continue
				}
				children = append(children, node)
			}
			return children

		default:
			return currentNode.ChildrenByFieldName(path, pe.cursor)
		}
	}
	return nil
}

func (pe parseEnv) child(node *treesitter.Node, path string) *treesitter.Node {
	paths := strings.Split(path, ":")
	var currentNode = node
	for _, path := range paths {
		switch {
		case currentNode == nil:
			break
		default:
			currentNode = currentNode.ChildByFieldName(path)
		}
	}
	return currentNode
}

func (pe parseEnv) fix(sym string) int {
	return pe.fixity[sym]
}

func (pe parseEnv) assoc(sym string) string {
	return pe.associativity[sym]
}

// Parse parses Haskell code and returns the AST as a Module
// ParseWithCounter parses Haskell source using the given shared node ID counter,
// so that multiple modules can have globally unique node IDs.
func ParseWithCounter(code []byte, altname string, counter *int) *Module {
	return parseWithCounter(code, altname, counter)
}

func Parse(code []byte, altname string) *Module {
	return parse(code, altname)
}

func parse(code []byte, altname string) *Module {
	initialCounter := 0
	return parseWithCounter(code, altname, &initialCounter)
}

func parseWithCounter(code []byte, altname string, counter *int) *Module {
	parser := treesitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(treesitter.NewLanguage(treesitterhaskell.Language()))

	tree := parser.Parse(code, nil)
	root := tree.RootNode()
	cursor := root.Walk()
	pe := parseEnv{
		counter:       counter,
		source:        code,
		cursor:        cursor,
		fixity:        fixity,
		associativity: associativity,
	}
	children := root.Children(cursor)

	moduleName := altname
	for _, child := range children {
		if child.Kind() == "header" {
			node := pe.child(&child, "module")
			moduleName = pe.text(node)
		}
	}

	// Parse imports
	importsNode := pe.child(root, "imports")
	imports := []Import{}
	if importsNode != nil {
		iNodes := pe.children(importsNode, "import")
		imports = make([]Import, len(iNodes))
		for i, in := range iNodes {
			imports[i] = pe.parseImport(&in)
		}
	}

	dNodes := pe.children(root, "declarations:*")
	var decls []Decl
	for _, d := range dNodes {
		decl := pe.parseDecl(&d)
		if decl != nil { // Filter out nil declarations (comments, etc.)
			decls = append(decls, decl)
		}
	}

	return &Module{
		Name:    moduleName,
		Decls:   decls,
		Imports: imports,
		Node:    pe.node(root),
	}
}

// GuessModuleName converts a file path to a Haskell module name relative to baseDir.
// Example: baseDir="src", filePath="src/data/list.hs" -> "Data.List"
func GuessModuleName(filePath string, baseDir string) string {
	// Make the path relative to baseDir if possible
	rel, err := filepath.Rel(baseDir, filePath)
	if err != nil {
		rel = filePath
	}

	// Remove .hs extension
	path := strings.TrimSuffix(rel, ".hs")

	// Remove leading ./ if present
	path = strings.TrimPrefix(path, "./")

	// Split by / and capitalize each part
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + part[1:]
		}
	}

	return strings.Join(parts, ".")
}

func ParseFile(filePath string) error {
	code, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	baseDir := filepath.Dir(filePath)
	moduleName := GuessModuleName(filePath, baseDir)
	if moduleName == "" {
		moduleName = "Main"
	}

	module := parse(code, moduleName)
	if module == nil {
		return fmt.Errorf("error parsing file")
	}

	fmt.Println(module.Pretty())
	return nil
}

// PrintSexp reads a Haskell file and prints the tree-sitter S-expression for debugging
func PrintSexp(filePath string) error {
	// Read the file
	code, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Create a tree-sitter parser
	parser := treesitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(treesitter.NewLanguage(treesitterhaskell.Language()))

	// Parse the code
	tree := parser.Parse(code, nil)
	defer tree.Close()

	// Get the root node and print its S-expression
	root := tree.RootNode()
	fmt.Println(root.ToSexp())

	return nil
}

var _ = fmt.Append
