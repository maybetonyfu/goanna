package parser

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// PrintAST prints the AST in an indented tree format showing type, ID, location, and children
func PrintAST(ast AST) {
	printASTWithIndent(ast, 0)
}

// printASTWithIndent recursively prints the AST with indentation
func printASTWithIndent(ast AST, indent int) {
	if ast == nil {
		return
	}

	indentStr := strings.Repeat("  ", indent)
	typeName := getTypeName(ast)
	id := ast.Id()
	loc := ast.Loc()
	name := getNodeName(ast)

	// Print the node with type, ID, name (if available), and location
	if name != "" {
		fmt.Printf("%s%s [id=%d, name=\"%s\"] (line %d-%d, col %d-%d)\n",
			indentStr, typeName, id, name,
			loc.FromLine(), loc.ToLine(), loc.FromCol(), loc.ToCol())
	} else {
		fmt.Printf("%s%s [id=%d] (line %d-%d, col %d-%d)\n",
			indentStr, typeName, id,
			loc.FromLine(), loc.ToLine(), loc.FromCol(), loc.ToCol())
	}

	// Print children with one additional level of indentation
	printChildren(ast, indent+1)
}

// getTypeName returns the type name of an AST node without the package prefix
func getTypeName(ast AST) string {
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
			return fmt.Sprintf("%s%s", node.Module, node.Name)
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
	default:
		return ""
	}
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
		for _, typeVar := range node.TypeVars {
			printASTWithIndent(&typeVar, indent)
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
		for _, assertion := range node.Assertions {
			printASTWithIndent(assertion, indent)
		}
		for _, ty := range node.Types {
			printASTWithIndent(ty, indent)
		}
		for _, decl := range node.Body {
			printASTWithIndent(decl, indent)
		}

	case *ClassDecl:
		for _, assertion := range node.Assertions {
			printASTWithIndent(assertion, indent)
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

	case *TyForall:
		for _, assertion := range node.Assertions {
			printASTWithIndent(assertion, indent)
		}
		printASTWithIndent(node.Ty, indent)
	}
}

// PrintASTFromFile parses a Haskell file and prints its AST
func PrintASTFromFile(filePath string) error {
	code, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	module := Parse(code, filePath)
	if module == nil {
		return fmt.Errorf("failed to parse file: %s", filePath)
	}

	PrintAST(module)
	return nil
}
