package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/fatih/color"
)

var mutedColor = color.New(color.FgHiBlack)

// formatNodeLine formats a single AST node as a compact line.
// withCanonical controls whether canonical names are included.
// Format: TypeName {#id, name, canonical} (fromLine,fromCol)-(toLine,toCol)
func formatNodeLine(ast AST, withCanonical bool) string {
	typeName := getTypeName(ast)
	id := ast.Id()
	loc := ast.Loc()

	locStr := mutedColor.Sprintf("(%d,%d)-(%d,%d)", loc.FromLine(), loc.FromCol(), loc.ToLine(), loc.ToCol())

	inner := fmt.Sprintf("#%d", id)

	name := getNodeName(ast)
	if name != "" {
		inner += ", " + name
		if withCanonical {
			var canonical string
			if _, ok := ast.(*TypeSig); ok {
				canonical = getTypeSigCanonicals(ast)
			} else {
				canonical = getCanonical(ast)
			}
			if canonical != "" && canonical != name {
				inner += ", " + canonical
			}
		}
	}

	return fmt.Sprintf("%s {%s} %s", typeName, inner, locStr)
}

// getCanonical returns the Canonical field of a Name node, or "".
func getCanonical(ast AST) string {
	switch node := ast.(type) {
	case *TyCon:
		return node.Canonical
	case *TyVar:
		return node.Canonical
	case *PVar:
		return node.Canonical
	case *ExpVar:
		return node.Canonical
	case *InstDecl:
		return node.Canonical
	case *DataCon:
		return node.Canonical
	case *DeclHead:
		return node.Canonical
	case *Assertion:
		return node.Canonical
	}
	return ""
}

// getTypeSigCanonicals returns the Canonicals of a TypeSig joined with ", ", or "".
func getTypeSigCanonicals(ast AST) string {
	if ts, ok := ast.(*TypeSig); ok && len(ts.Canonicals) > 0 {
		return strings.Join(ts.Canonicals, ", ")
	}
	return ""
}

// getTypeName returns the struct type name of an AST node.
func getTypeName(ast AST) string {
	if _, ok := ast.(*Assertion); ok {
		return "Assertion"
	}
	t := reflect.TypeOf(ast)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// getNodeName extracts the Name field if it exists in the AST node.
func getNodeName(ast AST) string {
	switch node := ast.(type) {
	case *TyCon:
		return node.Name
	case *TyVar:
		return node.Name
	case *PVar:
		return node.Name
	case *ExpVar:
		if node.Module != "" {
			return fmt.Sprintf("%s.%s", node.Module, node.Name)
		}
		return node.Name
	case *InstDecl:
		return node.Name
	case *DataCon:
		return node.Name
	case *DeclHead:
		return node.Name
	case *Module:
		return node.Name
	case *TypeSig:
		return strings.Join(node.Names, ", ")
	case *Assertion:
		if node.Module != "" {
			return node.Module + "." + node.Name
		}
		return node.Name
	}
	return ""
}

// printFn is the type for a recursive AST print function.
type printFn func(ast AST, indent int)

// printLabel prints a labelled list header.
func printLabel(label string, indent int) {
	fmt.Printf("%s%s:\n", strings.Repeat("  ", indent), label)
}

// printList prints a labelled list of items, each indented one level deeper.
// Nothing is printed if the list is empty.
func printList[T any](label string, items []T, indent int, toAST func(i int, item T) AST, fn printFn) {
	if len(items) == 0 {
		return
	}
	printLabel(label, indent)
	for i, item := range items {
		fn(toAST(i, item), indent+1)
	}
}

// printChildrenWith prints child nodes of ast using fn, labelling list fields.
func printChildrenWith(ast AST, indent int, fn printFn) {
	switch node := ast.(type) {
	case *Module:
		printList("imports", node.Imports, indent, func(i int, _ Import) AST { return &node.Imports[i] }, fn)
		printList("decls", node.Decls, indent, func(_ int, d Decl) AST { return d }, fn)

	case *Import: // leaf

	case *DeclHead:
		printList("typeVars", node.TypeVars, indent, func(i int, _ TyVar) AST { return &node.TypeVars[i] }, fn)

	case *DataCon:
		printList("types", node.Tys, indent, func(_ int, t Type) AST { return t }, fn)

	case *Assertion:
		printList("types", node.Types, indent, func(_ int, t Type) AST { return t }, fn)

	case *Alt:
		fn(node.Pat, indent)
		fn(node.Exp, indent)
		printList("binds", node.Binds, indent, func(_ int, d Decl) AST { return d }, fn)

	case *TypeSig:
		fn(node.Ty, indent)

	case *PatBind:
		fn(node.Pat, indent)
		fn(node.Rhs, indent)

	case *InstDecl:
		printList("assertions", node.Assertions, indent, func(i int, _ Assertion) AST { return &node.Assertions[i] }, fn)
		printList("types", node.Types, indent, func(_ int, t Type) AST { return t }, fn)
		printList("body", node.Body, indent, func(_ int, d Decl) AST { return d }, fn)

	case *ClassDecl:
		printList("assertions", node.Assertions, indent, func(i int, _ Assertion) AST { return &node.Assertions[i] }, fn)
		fn(&node.DHead, indent)
		printList("decls", node.Decls, indent, func(_ int, d Decl) AST { return d }, fn)

	case *DataDecl:
		fn(&node.DHead, indent)
		printList("constructors", node.Constructors, indent, func(i int, _ DataCon) AST { return &node.Constructors[i] }, fn)
		printList("deriving", node.Deriving, indent, func(i int, _ TyCon) AST { return &node.Deriving[i] }, fn)

	case *TypeDecl:
		fn(&node.DHead, indent)
		fn(node.Ty, indent)

	case *ExpVar: // leaf

	case *ExpApp:
		fn(node.Exp1, indent)
		fn(node.Exp2, indent)

	case *ExpInfix:
		fn(node.Exp1, indent)
		fn(&node.Op, indent)
		fn(node.Exp2, indent)

	case *ExpLambda:
		printList("pats", node.Pats, indent, func(_ int, p Pat) AST { return p }, fn)
		fn(node.Exp, indent)

	case *ExpLet:
		printList("binds", node.Binds, indent, func(_ int, d Decl) AST { return d }, fn)
		fn(node.Exp, indent)

	case *ExpIf:
		fn(node.Cond, indent)
		fn(node.IfTrue, indent)
		fn(node.IfFalse, indent)

	case *ExpDo:
		printList("stmts", node.Stmts, indent, func(_ int, s Statement) AST { return s }, fn)

	case *ExpCase:
		fn(node.Exp, indent)
		printList("alts", node.Alts, indent, func(i int, _ Alt) AST { return &node.Alts[i] }, fn)

	case *ExpTuple:
		printList("exps", node.Exps, indent, func(_ int, e Exp) AST { return e }, fn)

	case *ExpList:
		printList("exps", node.Exps, indent, func(_ int, e Exp) AST { return e }, fn)

	case *ExpLeftSection:
		fn(node.Left, indent)
		fn(node.Op, indent)

	case *ExpRightSection:
		fn(node.Op, indent)
		fn(node.Right, indent)

	case *ExpEnumFrom:
		fn(node.Exp, indent)

	case *ExpEnumFromTo:
		fn(node.Exp1, indent)
		fn(node.Exp2, indent)

	case *ExpComprehension:
		fn(node.Exp, indent)
		printList("generators", node.Generators, indent, func(i int, _ Generator) AST { return &node.Generators[i] }, fn)
		printList("guards", node.Guards, indent, func(_ int, e Exp) AST { return e }, fn)

	case *Lit: // leaf

	case *UnguardedRhs:
		if node.Exp != nil {
			fn(node.Exp, indent)
		}
		printList("wheres", node.Wheres, indent, func(_ int, d Decl) AST { return d }, fn)

	case *GuardedRhs:
		printList("branches", node.Branches, indent, func(i int, _ GuardBranch) AST { return &node.Branches[i] }, fn)
		printList("wheres", node.Wheres, indent, func(_ int, d Decl) AST { return d }, fn)

	case *GuardBranch:
		printList("guards", node.Guards, indent, func(_ int, e Exp) AST { return e }, fn)
		fn(node.Exp, indent)

	case *Generator:
		fn(node.Pat, indent)
		fn(node.Exp, indent)

	case *Qualifier:
		fn(node.Exp, indent)

	case *LetStmt:
		printList("binds", node.Binds, indent, func(_ int, d Decl) AST { return d }, fn)

	case *PWildcard: // leaf

	case *PApp:
		fn(&node.Constructor, indent)
		printList("pats", node.Pats, indent, func(_ int, p Pat) AST { return p }, fn)

	case *PList:
		printList("pats", node.Pats, indent, func(_ int, p Pat) AST { return p }, fn)

	case *PTuple:
		printList("pats", node.Pats, indent, func(_ int, p Pat) AST { return p }, fn)

	case *PVar: // leaf

	case *PInfix:
		fn(node.Pat1, indent)
		fn(&node.Op, indent)
		fn(node.Pat2, indent)

	case *TyCon: // leaf

	case *TyApp:
		fn(node.Ty1, indent)
		fn(node.Ty2, indent)

	case *TyFunction:
		fn(node.Ty1, indent)
		fn(node.Ty2, indent)

	case *TyTuple:
		printList("types", node.Tys, indent, func(_ int, t Type) AST { return t }, fn)

	case *TyList:
		fn(node.Ty, indent)

	case *TyVar: // leaf

	case *TyForall:
		printList("assertions", node.Assertions, indent, func(i int, _ Assertion) AST { return &node.Assertions[i] }, fn)
		fn(node.Ty, indent)
	}
}

func printChildren(ast AST, indent int) {
	printChildrenWith(ast, indent, printASTWithIndent)
}

func printChildrenWithCanonicals(ast AST, indent int) {
	printChildrenWith(ast, indent, printASTWithIndentAndCanonicals)
}

func printASTWithIndent(ast AST, indent int) {
	if ast == nil {
		return
	}
	fmt.Printf("%s%s\n", strings.Repeat("  ", indent), formatNodeLine(ast, false))
	printChildren(ast, indent+1)
}

func printASTWithIndentAndCanonicals(ast AST, indent int) {
	if ast == nil {
		return
	}
	fmt.Printf("%s%s\n", strings.Repeat("  ", indent), formatNodeLine(ast, true))
	printChildrenWithCanonicals(ast, indent+1)
}

// PrintAST prints the AST in an indented tree format.
func PrintAST(ast AST) {
	printASTWithIndent(ast, 0)
}

// PrintASTWithCanonicals prints the AST showing Canonical names alongside Name fields.
func PrintASTWithCanonicals(ast AST) {
	printASTWithIndentAndCanonicals(ast, 0)
}

// PrintASTFromFile parses a Haskell file and prints its AST
func PrintASTFromFile(filePath string) error {
	code, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	baseDir := filepath.Dir(filePath)
	moduleName := GuessModuleName(filePath, baseDir)
	if moduleName == "" {
		moduleName = "Main"
	}

	module := Parse(code, moduleName)
	if module == nil {
		return fmt.Errorf("failed to parse file: %s", filePath)
	}

	PrintAST(module)
	return nil
}
