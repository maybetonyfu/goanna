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
		for _, decl := range node.decls {
			t.visit(decl, node)
		}
	// Misc
	case *DeclHead:
		for _, typeVar := range node.typeVars {
			t.visit(&typeVar, node)
		}
	case *DataCon:
		for _, ty := range node.tys {
			t.visit(ty, node)
		}
	case *Alt:
		t.visit(node.exp, node)
		t.visit(node.pat, node)
		for _, decl := range node.binds {
			t.visit(decl, node)
		}

	// Decls
	case *TypeSig:
		t.visit(node.ty, node)

	case *PatBind:
		t.visit(node.pat, node)
		t.visit(node.rhs, node)

	case *InstDecl:
		for _, assertion := range node.assertions {
			t.visit(assertion, node)
		}
		for _, ty := range node.types {
			t.visit(ty, node)
		}
		for _, decl := range node.body {
			t.visit(decl, node)
		}
	case *ClassDecl:
		for _, assertion := range node.assertions {
			t.visit(assertion, node)
		}
		for _, decl := range node.decls {
			t.visit(decl, node)
		}
		t.visit(&node.dHead, node)
	case *DataDecl:
		for _, constructor := range node.constructors {
			t.visit(&constructor, node)
		}
		for _, derive := range node.deriving {
			t.visit(&derive, node)
		}
		t.visit(&node.dHead, node)
	case *TypeDecl:
		t.visit(node.ty, node)
		t.visit(&node.dHead, node)

	// Exp
	case *ExpVar:
	case *ExpApp:
		t.visit(node.exp1, node)
		t.visit(node.exp2, node)
	case *ExpInfix:
		t.visit(node.exp1, node)
		t.visit(node.exp2, node)
		t.visit(&node.op, node)
	case *ExpLambda:
		for _, pat := range node.pats {
			t.visit(pat, node)
		}
		t.visit(node.exp, node)
	case *ExpLet:
		t.visit(node.exp, node)
		for _, decl := range node.binds {
			t.visit(decl, node)
		}
	case *ExpIf:
		t.visit(node.cond, node)
		t.visit(node.ifTrue, node)
		t.visit(node.ifFalse, node)
	case *ExpDo:
		for _, stmt := range node.stmts {
			t.visit(stmt, node)
		}
	case *ExpCase:
		t.visit(node.exp, node)
		for _, alt := range node.alts {
			t.visit(&alt, node)
		}
	case *ExpTuple:
		for _, exp := range node.exps {
			t.visit(exp, node)
		}
	case *ExpList:
		for _, exp := range node.exps {
			t.visit(exp, node)
		}
	case *ExpLeftSection:
		t.visit(node.left, node)
		t.visit(node.op, node)
	case *ExpRightSection:
		t.visit(node.right, node)
		t.visit(node.op, node)
	case *ExpEnumFrom:
		t.visit(node.exp, node)
	case *ExpEnumFromTo:
		t.visit(node.exp1, node)
		t.visit(node.exp2, node)
	case *ExpComprehension:
		t.visit(node.exp, node)
		for _, gen := range node.generators {
			t.visit(&gen, node)
		}
		for _, guard := range node.guards {
			t.visit(guard, node)
		}
	case *Lit:
	// RHS
	case *GuardedRhs:
		for _, branch := range node.branches {
			t.visit(&branch, node)
		}
		for _, where := range node.wheres {
			t.visit(where, node)
		}
	case *UnguardedRhs:
		t.visit(node.exp, node)
		for _, decl := range node.wheres {
			t.visit(decl, node)
		}
	case *GuardBranch:
		t.visit(node.exp, node)
		for _, guard := range node.guards {
			t.visit(guard, node)
		}

	// Statements
	case *Generator:
		t.visit(node.exp, node)
		t.visit(node.pat, node)
	case *Qualifier:
		t.visit(node.exp, node)
	case *LetStmt:
		for _, decl := range node.binds {
			t.visit(decl, node)
		}

	// Pattern
	case *PWildcard:
	case *PApp:
		t.visit(&node.constructor, node)
		for _, pat := range node.pats {
			t.visit(pat, node)
		}
	case *PList:
		for _, pat := range node.pats {
			t.visit(pat, node)
		}
	case *PTuple:
		for _, pat := range node.pats {
			t.visit(pat, node)
		}
	case *PVar:
	case *PInfix:
		t.visit(node.pat1, node)
		t.visit(node.pat2, node)
		t.visit(&node.op, node)
	// Types
	case *TyCon:
	case *TyApp:
		t.visit(node.ty1, node)
		t.visit(node.ty2, node)
	case *TyFunction:
		t.visit(node.ty1, node)
		t.visit(node.ty2, node)
	case *TyTuple:
		for _, ty := range node.tys {
			t.visit(ty, node)
		}
	case *TyList:
		t.visit(node.ty, node)
	case *TyVar:
	case *TyForall:
		for _, assertion := range node.assertions {
			t.visit(assertion, node)
		}
		t.visit(node.ty, node)
	}

}
