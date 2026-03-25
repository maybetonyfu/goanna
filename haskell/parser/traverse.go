package parser

type Traverser[T any] struct {
	sub  func(v T, ast AST, parent AST) T
	data T
}

// NewTraverser creates a new traverser with the given visitor function and initial data
func NewTraverser[T any](visitorFunc func(v T, ast AST, parent AST) T, initialData T) Traverser[T] {
	return Traverser[T]{
		sub:  visitorFunc,
		data: initialData,
	}
}

// Visit traverses the AST starting from the given node
func (t Traverser[T]) Visit(ast AST, parent AST) {
	t.visit(ast, parent)
}

func (t Traverser[T]) visit(ast AST, parent AST) {
	t.sub(t.data, ast, parent)
	switch node := ast.(type) {
	case *Module:
		for _, decl := range node.Decls {
			t.visit(decl, node)
		}
	// Misc
	case *DeclHead:
		for _, typeVar := range node.TypeVars {
			t.visit(&typeVar, node)
		}
	case *DataCon:
		for _, ty := range node.Tys {
			t.visit(ty, node)
		}
	case *Alt:
		t.visit(node.Exp, node)
		t.visit(node.Pat, node)
		for _, decl := range node.Binds {
			t.visit(decl, node)
		}

	// Decls
	case *TypeSig:
		t.visit(node.Ty, node)

	case *PatBind:
		t.visit(node.Pat, node)
		t.visit(node.Rhs, node)

	case *InstDecl:
		for _, assertion := range node.Assertions {
			t.visit(assertion, node)
		}
		for _, ty := range node.Types {
			t.visit(ty, node)
		}
		for _, decl := range node.Body {
			t.visit(decl, node)
		}
	case *ClassDecl:
		for _, assertion := range node.Assertions {
			t.visit(assertion, node)
		}
		for _, decl := range node.Decls {
			t.visit(decl, node)
		}
		t.visit(&node.DHead, node)
	case *DataDecl:
		for _, constructor := range node.Constructors {
			t.visit(&constructor, node)
		}
		for _, derive := range node.Deriving {
			t.visit(&derive, node)
		}
		t.visit(&node.DHead, node)
	case *TypeDecl:
		t.visit(node.Ty, node)
		t.visit(&node.DHead, node)

	// Exp
	case *ExpVar:
	case *ExpApp:
		t.visit(node.Exp1, node)
		t.visit(node.Exp2, node)
	case *ExpInfix:
		t.visit(node.Exp1, node)
		t.visit(node.Exp2, node)
		t.visit(&node.Op, node)
	case *ExpLambda:
		for _, pat := range node.Pats {
			t.visit(pat, node)
		}
		t.visit(node.Exp, node)
	case *ExpLet:
		t.visit(node.Exp, node)
		for _, decl := range node.Binds {
			t.visit(decl, node)
		}
	case *ExpIf:
		t.visit(node.Cond, node)
		t.visit(node.IfTrue, node)
		t.visit(node.IfFalse, node)
	case *ExpDo:
		for _, stmt := range node.Stmts {
			t.visit(stmt, node)
		}
	case *ExpCase:
		t.visit(node.Exp, node)
		for _, alt := range node.Alts {
			t.visit(&alt, node)
		}
	case *ExpTuple:
		for _, exp := range node.Exps {
			t.visit(exp, node)
		}
	case *ExpList:
		for _, exp := range node.Exps {
			t.visit(exp, node)
		}
	case *ExpLeftSection:
		t.visit(node.Left, node)
		t.visit(node.Op, node)
	case *ExpRightSection:
		t.visit(node.Right, node)
		t.visit(node.Op, node)
	case *ExpEnumFrom:
		t.visit(node.Exp, node)
	case *ExpEnumFromTo:
		t.visit(node.Exp1, node)
		t.visit(node.Exp2, node)
	case *ExpComprehension:
		t.visit(node.Exp, node)
		for _, gen := range node.Generators {
			t.visit(&gen, node)
		}
		for _, guard := range node.Guards {
			t.visit(guard, node)
		}
	case *Lit:
	// RHS
	case *GuardedRhs:
		for _, branch := range node.Branches {
			t.visit(&branch, node)
		}
		for _, where := range node.Wheres {
			t.visit(where, node)
		}
	case *UnguardedRhs:
		t.visit(node.Exp, node)
		for _, decl := range node.Wheres {
			t.visit(decl, node)
		}
	case *GuardBranch:
		t.visit(node.Exp, node)
		for _, guard := range node.Guards {
			t.visit(guard, node)
		}

	// Statements
	case *Generator:
		t.visit(node.Exp, node)
		t.visit(node.Pat, node)
	case *Qualifier:
		t.visit(node.Exp, node)
	case *LetStmt:
		for _, decl := range node.Binds {
			t.visit(decl, node)
		}

	// Pattern
	case *PWildcard:
	case *PApp:
		t.visit(&node.Constructor, node)
		for _, pat := range node.Pats {
			t.visit(pat, node)
		}
	case *PList:
		for _, pat := range node.Pats {
			t.visit(pat, node)
		}
	case *PTuple:
		for _, pat := range node.Pats {
			t.visit(pat, node)
		}
	case *PVar:
	case *PInfix:
		t.visit(node.Pat1, node)
		t.visit(node.Pat2, node)
		t.visit(&node.Op, node)
	// Types
	case *TyCon:
	case *TyApp:
		t.visit(node.Ty1, node)
		t.visit(node.Ty2, node)
	case *TyFunction:
		t.visit(node.Ty1, node)
		t.visit(node.Ty2, node)
	case *TyTuple:
		for _, ty := range node.Tys {
			t.visit(ty, node)
		}
	case *TyList:
		t.visit(node.Ty, node)
	case *TyVar:
	case *TyForall:
		for _, assertion := range node.Assertions {
			t.visit(assertion, node)
		}
		t.visit(node.Ty, node)
	}

}
