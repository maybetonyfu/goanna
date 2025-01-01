package parser

import (
	"fmt"
	treesitter "github.com/tree-sitter/go-tree-sitter"
	treesitterhaskell "github.com/tree-sitter/tree-sitter-haskell/bindings/go"
	"strings"
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
	counter       int
	source        []byte
	cursor        *treesitter.TreeCursor
	fixity        map[string]int
	associativity map[string]string
}

func (pe parseEnv) id() int {
	id := pe.counter
	pe.counter += 1
	return id
}

func (pe parseEnv) loc(node *treesitter.Node) Loc {
	nodeRange := node.Range()
	return Loc{
		fromLine: int(nodeRange.StartPoint.Row),
		fromCol: int(nodeRange.StartPoint.Column),
		toLine: int(nodeRange.EndPoint.Row),
		toCol: int(nodeRange.EndPoint.Column),
	}
}

func (pe parseEnv) node(node *treesitter.Node) Node {
	n := Node{
		id: pe.id(),
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

func parse(code []byte, altname string) *Module {
	parser := treesitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(treesitter.NewLanguage(treesitterhaskell.Language()))

	tree := parser.Parse(code, nil)
	root := tree.RootNode()
	cursor := root.Walk()

	pe := parseEnv{
		counter: 0,
		source:  code,
		cursor:  cursor,
		fixity: fixity,
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
	dNodes := pe.children(root, "declarations:*")
	decls := make([]Decl, len(dNodes))
	for i, d := range dNodes {
		decls[i] = pe.parseDecl(&d)
	}

	return &Module{
		name:    moduleName,
		decls:   decls,
		imports: []string{},
		Node:    pe.node(root),
	}
}

func Main() {
	code := []byte("module Main where")

	parser := treesitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(treesitter.NewLanguage(treesitterhaskell.Language()))

	tree := parser.Parse(code, nil)
	defer tree.Close()

	root := tree.RootNode()
	fmt.Println(root.ToSexp())
	cursor := root.Walk()
	pe := parseEnv{0, code, cursor, nil, nil}
	decls := pe.children(root, "declarations:*")
	for _, decl := range decls {
		pe.parseDecl(&decl)
	}
}
