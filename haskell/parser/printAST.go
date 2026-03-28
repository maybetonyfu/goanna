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
// Fields inside {} are omitted if empty. The {} block is omitted entirely if all fields are empty.
func formatNodeLine(ast AST, withCanonical bool) string {
	typeName := getTypeName(ast)
	id := ast.Id()
	loc := ast.Loc()

	locStr := mutedColor.Sprintf("(%d,%d)-(%d,%d)", loc.FromLine(), loc.FromCol(), loc.ToLine(), loc.ToCol())

	// Build inner fields: id always present, name and canonical optional
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

// getCanonical returns the Canonical field of a Name node, or "" if it has none.
func getCanonical(ast AST) string {
	if n, ok := ast.(Name); ok {
		// Retrieve via type assertion to the concrete field
		switch node := n.(type) {
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
	}
	return ""
}

// printASTWithIndentAndCanonicals is like printASTWithIndent but also prints
// the Canonical name alongside the Name when present.
func printASTWithIndentAndCanonicals(ast AST, indent int) {
	if ast == nil {
		return
	}
	fmt.Printf("%s%s\n", strings.Repeat("  ", indent), formatNodeLine(ast, true))
	printChildrenWithCanonicals(ast, indent+1)
}

// PrintASTWithCanonicals prints the AST of a module, showing Canonical names
// alongside Name fields where they differ.
func PrintASTWithCanonicals(ast AST) {
	printASTWithIndentAndCanonicals(ast, 0)
}

func PrintAST(ast AST) {
	printASTWithIndent(ast, 0)
}

// printASTWithIndent recursively prints the AST with indentation
func printASTWithIndent(ast AST, indent int) {
	if ast == nil {
		return
	}
	fmt.Printf("%s%s\n", strings.Repeat("  ", indent), formatNodeLine(ast, false))
	printChildren(ast, indent+1)
}

// getTypeName returns the type name of an AST node without the package prefix
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

// getNodeName extracts the Name field if it exists in the AST node
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
		} else {
			return node.Name
		}
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
	default:
		return ""
	}
}

// getTypeSigCanonicals returns the Canonicals of a TypeSig joined with ", ", or "".
func getTypeSigCanonicals(ast AST) string {
	if ts, ok := ast.(*TypeSig); ok && len(ts.Canonicals) > 0 {
		return strings.Join(ts.Canonicals, ", ")
	}
	return ""
}

// printChildren prints all child nodes of an AST node
func printChildren(ast AST, indent int) {
	switch node := ast.(type) {
	// Module
	case *Module:
		for _, decl := range node.Decls {
			printASTWithIndent(decl, indent)
		}
		for _, imp := range node.Imports {
			printASTWithIndent(&imp, indent)
		}

	// Import
	case *Import:
		// Imports don't have AST children, but we could print their info
		// For now, just the node itself is shown

	// Misc nodes
	case *DeclHead:
		for i := range node.TypeVars {
			printASTWithIndent(&node.TypeVars[i], indent)
		}

	case *DataCon:
		for _, ty := range node.Tys {
			printASTWithIndent(ty, indent)
		}

	case *Alt:
		printASTWithIndent(node.Pat, indent)
		printASTWithIndent(node.Exp, indent)
		for _, decl := range node.Binds {
			printASTWithIndent(decl, indent)
		}

	// Declarations
	case *TypeSig:
		printASTWithIndent(node.Ty, indent)

	case *PatBind:
		printASTWithIndent(node.Pat, indent)
		printASTWithIndent(node.Rhs, indent)

	case *InstDecl:
		for i := range node.Assertions {
			printASTWithIndent(&node.Assertions[i], indent)
		}
		for _, ty := range node.Types {
			printASTWithIndent(ty, indent)
		}
		for _, decl := range node.Body {
			printASTWithIndent(decl, indent)
		}

	case *ClassDecl:
		for i := range node.Assertions {
			printASTWithIndent(&node.Assertions[i], indent)
		}
		printASTWithIndent(&node.DHead, indent)
		for _, decl := range node.Decls {
			printASTWithIndent(decl, indent)
		}

	case *DataDecl:
		printASTWithIndent(&node.DHead, indent)
		for _, constructor := range node.Constructors {
			printASTWithIndent(&constructor, indent)
		}
		for _, derive := range node.Deriving {
			printASTWithIndent(&derive, indent)
		}

	case *TypeDecl:
		printASTWithIndent(&node.DHead, indent)
		printASTWithIndent(node.Ty, indent)

	// Expressions
	case *ExpVar:
		// Leaf node

	case *ExpApp:
		printASTWithIndent(node.Exp1, indent)
		printASTWithIndent(node.Exp2, indent)

	case *ExpInfix:
		printASTWithIndent(node.Exp1, indent)
		printASTWithIndent(&node.Op, indent)
		printASTWithIndent(node.Exp2, indent)

	case *ExpLambda:
		for _, pat := range node.Pats {
			printASTWithIndent(pat, indent)
		}
		printASTWithIndent(node.Exp, indent)

	case *ExpLet:
		for _, decl := range node.Binds {
			printASTWithIndent(decl, indent)
		}
		printASTWithIndent(node.Exp, indent)

	case *ExpIf:
		printASTWithIndent(node.Cond, indent)
		printASTWithIndent(node.IfTrue, indent)
		printASTWithIndent(node.IfFalse, indent)

	case *ExpDo:
		for _, stmt := range node.Stmts {
			printASTWithIndent(stmt, indent)
		}

	case *ExpCase:
		printASTWithIndent(node.Exp, indent)
		for _, alt := range node.Alts {
			printASTWithIndent(&alt, indent)
		}

	case *ExpTuple:
		for _, exp := range node.Exps {
			printASTWithIndent(exp, indent)
		}

	case *ExpList:
		for _, exp := range node.Exps {
			printASTWithIndent(exp, indent)
		}

	case *ExpLeftSection:
		printASTWithIndent(node.Left, indent)
		printASTWithIndent(node.Op, indent)

	case *ExpRightSection:
		printASTWithIndent(node.Op, indent)
		printASTWithIndent(node.Right, indent)

	case *ExpEnumFrom:
		printASTWithIndent(node.Exp, indent)

	case *ExpEnumFromTo:
		printASTWithIndent(node.Exp1, indent)
		printASTWithIndent(node.Exp2, indent)

	case *ExpComprehension:
		printASTWithIndent(node.Exp, indent)
		for _, gen := range node.Generators {
			printASTWithIndent(&gen, indent)
		}
		for _, guard := range node.Guards {
			printASTWithIndent(guard, indent)
		}

	case *Lit:
		// Leaf node

	// RHS
	case *UnguardedRhs:
		if node.Exp != nil {
			printASTWithIndent(node.Exp, indent)
		}
		for _, decl := range node.Wheres {
			printASTWithIndent(decl, indent)
		}

	case *GuardedRhs:
		for _, branch := range node.Branches {
			printASTWithIndent(&branch, indent)
		}
		for _, where := range node.Wheres {
			printASTWithIndent(where, indent)
		}

	case *GuardBranch:
		for _, guard := range node.Guards {
			printASTWithIndent(guard, indent)
		}
		printASTWithIndent(node.Exp, indent)

	// Statements
	case *Generator:
		printASTWithIndent(node.Pat, indent)
		printASTWithIndent(node.Exp, indent)

	case *Qualifier:
		printASTWithIndent(node.Exp, indent)

	case *LetStmt:
		for _, decl := range node.Binds {
			printASTWithIndent(decl, indent)
		}

	// Patterns
	case *PWildcard:
		// Leaf node

	case *PApp:
		printASTWithIndent(&node.Constructor, indent)
		for _, pat := range node.Pats {
			printASTWithIndent(pat, indent)
		}

	case *PList:
		for _, pat := range node.Pats {
			printASTWithIndent(pat, indent)
		}

	case *PTuple:
		for _, pat := range node.Pats {
			printASTWithIndent(pat, indent)
		}

	case *PVar:
		// Leaf node

	case *PInfix:
		printASTWithIndent(node.Pat1, indent)
		printASTWithIndent(&node.Op, indent)
		printASTWithIndent(node.Pat2, indent)

	// Types
	case *TyCon:
		// Leaf node

	case *TyApp:
		printASTWithIndent(node.Ty1, indent)
		printASTWithIndent(node.Ty2, indent)

	case *TyFunction:
		printASTWithIndent(node.Ty1, indent)
		printASTWithIndent(node.Ty2, indent)

	case *TyTuple:
		for _, ty := range node.Tys {
			printASTWithIndent(ty, indent)
		}

	case *TyList:
		printASTWithIndent(node.Ty, indent)

	case *TyVar:
		// Leaf node

	case *Assertion:
		for _, ty := range node.Types {
			printASTWithIndent(ty, indent)
		}
	case *TyForall:
		for i := range node.Assertions {
			printASTWithIndent(&node.Assertions[i], indent)
		}
		printASTWithIndent(node.Ty, indent)
	}
}

// printChildrenWithCanonicals is like printChildren but uses printASTWithIndentAndCanonicals.
func printChildrenWithCanonicals(ast AST, indent int) {
	switch node := ast.(type) {
	case *Module:
		for _, decl := range node.Decls {
			printASTWithIndentAndCanonicals(decl, indent)
		}
		for _, imp := range node.Imports {
			printASTWithIndentAndCanonicals(&imp, indent)
		}
	case *Import:
	case *DeclHead:
		for i := range node.TypeVars {
			printASTWithIndentAndCanonicals(&node.TypeVars[i], indent)
		}
	case *DataCon:
		for _, ty := range node.Tys {
			printASTWithIndentAndCanonicals(ty, indent)
		}
	case *Alt:
		printASTWithIndentAndCanonicals(node.Pat, indent)
		printASTWithIndentAndCanonicals(node.Exp, indent)
		for _, decl := range node.Binds {
			printASTWithIndentAndCanonicals(decl, indent)
		}
	case *TypeSig:
		printASTWithIndentAndCanonicals(node.Ty, indent)
	case *PatBind:
		printASTWithIndentAndCanonicals(node.Pat, indent)
		printASTWithIndentAndCanonicals(node.Rhs, indent)
	case *InstDecl:
		for i := range node.Assertions {
			printASTWithIndentAndCanonicals(&node.Assertions[i], indent)
		}
		for _, ty := range node.Types {
			printASTWithIndentAndCanonicals(ty, indent)
		}
		for _, decl := range node.Body {
			printASTWithIndentAndCanonicals(decl, indent)
		}
	case *ClassDecl:
		for i := range node.Assertions {
			printASTWithIndentAndCanonicals(&node.Assertions[i], indent)
		}
		printASTWithIndentAndCanonicals(&node.DHead, indent)
		for _, decl := range node.Decls {
			printASTWithIndentAndCanonicals(decl, indent)
		}
	case *DataDecl:
		printASTWithIndentAndCanonicals(&node.DHead, indent)
		for i := range node.Constructors {
			printASTWithIndentAndCanonicals(&node.Constructors[i], indent)
		}
		for i := range node.Deriving {
			printASTWithIndentAndCanonicals(&node.Deriving[i], indent)
		}
	case *TypeDecl:
		printASTWithIndentAndCanonicals(&node.DHead, indent)
		printASTWithIndentAndCanonicals(node.Ty, indent)
	case *ExpVar:
	case *ExpApp:
		printASTWithIndentAndCanonicals(node.Exp1, indent)
		printASTWithIndentAndCanonicals(node.Exp2, indent)
	case *ExpInfix:
		printASTWithIndentAndCanonicals(node.Exp1, indent)
		printASTWithIndentAndCanonicals(&node.Op, indent)
		printASTWithIndentAndCanonicals(node.Exp2, indent)
	case *ExpLambda:
		for _, pat := range node.Pats {
			printASTWithIndentAndCanonicals(pat, indent)
		}
		printASTWithIndentAndCanonicals(node.Exp, indent)
	case *ExpLet:
		for _, decl := range node.Binds {
			printASTWithIndentAndCanonicals(decl, indent)
		}
		printASTWithIndentAndCanonicals(node.Exp, indent)
	case *ExpIf:
		printASTWithIndentAndCanonicals(node.Cond, indent)
		printASTWithIndentAndCanonicals(node.IfTrue, indent)
		printASTWithIndentAndCanonicals(node.IfFalse, indent)
	case *ExpDo:
		for _, stmt := range node.Stmts {
			printASTWithIndentAndCanonicals(stmt, indent)
		}
	case *ExpCase:
		printASTWithIndentAndCanonicals(node.Exp, indent)
		for _, alt := range node.Alts {
			printASTWithIndentAndCanonicals(&alt, indent)
		}
	case *ExpTuple:
		for _, exp := range node.Exps {
			printASTWithIndentAndCanonicals(exp, indent)
		}
	case *ExpList:
		for _, exp := range node.Exps {
			printASTWithIndentAndCanonicals(exp, indent)
		}
	case *ExpLeftSection:
		printASTWithIndentAndCanonicals(node.Left, indent)
		printASTWithIndentAndCanonicals(node.Op, indent)
	case *ExpRightSection:
		printASTWithIndentAndCanonicals(node.Op, indent)
		printASTWithIndentAndCanonicals(node.Right, indent)
	case *ExpEnumFrom:
		printASTWithIndentAndCanonicals(node.Exp, indent)
	case *ExpEnumFromTo:
		printASTWithIndentAndCanonicals(node.Exp1, indent)
		printASTWithIndentAndCanonicals(node.Exp2, indent)
	case *ExpComprehension:
		printASTWithIndentAndCanonicals(node.Exp, indent)
		for _, gen := range node.Generators {
			printASTWithIndentAndCanonicals(&gen, indent)
		}
		for _, guard := range node.Guards {
			printASTWithIndentAndCanonicals(guard, indent)
		}
	case *Lit:
	case *UnguardedRhs:
		if node.Exp != nil {
			printASTWithIndentAndCanonicals(node.Exp, indent)
		}
		for _, decl := range node.Wheres {
			printASTWithIndentAndCanonicals(decl, indent)
		}
	case *GuardedRhs:
		for _, branch := range node.Branches {
			printASTWithIndentAndCanonicals(&branch, indent)
		}
		for _, where := range node.Wheres {
			printASTWithIndentAndCanonicals(where, indent)
		}
	case *GuardBranch:
		for _, guard := range node.Guards {
			printASTWithIndentAndCanonicals(guard, indent)
		}
		printASTWithIndentAndCanonicals(node.Exp, indent)
	case *Generator:
		printASTWithIndentAndCanonicals(node.Pat, indent)
		printASTWithIndentAndCanonicals(node.Exp, indent)
	case *Qualifier:
		printASTWithIndentAndCanonicals(node.Exp, indent)
	case *LetStmt:
		for _, decl := range node.Binds {
			printASTWithIndentAndCanonicals(decl, indent)
		}
	case *PWildcard:
	case *PApp:
		printASTWithIndentAndCanonicals(&node.Constructor, indent)
		for _, pat := range node.Pats {
			printASTWithIndentAndCanonicals(pat, indent)
		}
	case *PList:
		for _, pat := range node.Pats {
			printASTWithIndentAndCanonicals(pat, indent)
		}
	case *PTuple:
		for _, pat := range node.Pats {
			printASTWithIndentAndCanonicals(pat, indent)
		}
	case *PVar:
	case *PInfix:
		printASTWithIndentAndCanonicals(node.Pat1, indent)
		printASTWithIndentAndCanonicals(&node.Op, indent)
		printASTWithIndentAndCanonicals(node.Pat2, indent)
	case *TyCon:
	case *TyApp:
		printASTWithIndentAndCanonicals(node.Ty1, indent)
		printASTWithIndentAndCanonicals(node.Ty2, indent)
	case *TyFunction:
		printASTWithIndentAndCanonicals(node.Ty1, indent)
		printASTWithIndentAndCanonicals(node.Ty2, indent)
	case *TyTuple:
		for _, ty := range node.Tys {
			printASTWithIndentAndCanonicals(ty, indent)
		}
	case *TyList:
		printASTWithIndentAndCanonicals(node.Ty, indent)
	case *TyVar:
	case *Assertion:
		for _, ty := range node.Types {
			printASTWithIndentAndCanonicals(ty, indent)
		}
	case *TyForall:
		for i := range node.Assertions {
			printASTWithIndentAndCanonicals(&node.Assertions[i], indent)
		}
		printASTWithIndentAndCanonicals(node.Ty, indent)
	}
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
