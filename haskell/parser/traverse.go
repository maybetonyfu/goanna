package parser

type Traverser[T any] struct {
	enter func(v T, ast AST, parent AST) T
	leave func(v T, ast AST, parent AST)
	data  T
}

// NewTraverser creates a new traverser with enter and leave hooks.
// enter is called before visiting children; its return value is passed to children.
// leave is called after visiting children with the same value returned by enter.
// leave may be nil if no post-order processing is needed.
func NewTraverser[T any](enter func(v T, ast AST, parent AST) T, leave func(v T, ast AST, parent AST), initialData T) Traverser[T] {
	return Traverser[T]{
		enter: enter,
		leave: leave,
		data:  initialData,
	}
}

// Visit traverses the AST starting from the given node
func (t Traverser[T]) Visit(ast AST, parent AST) {
	t.visit(ast, parent, t.data)
}

func (t Traverser[T]) visit(ast AST, parent AST, data T) {
	childData := t.enter(data, ast, parent)
	switch node := ast.(type) {
	case *Module:
		for _, decl := range node.Decls {
			t.visit(decl, node, childData)
		}
	// Misc
	case *DeclHead:
		for i := range node.TypeVars {
			t.visit(&node.TypeVars[i], node, childData)
		}
	case *DataCon:
		for _, ty := range node.Tys {
			t.visit(ty, node, childData)
		}
	case *Alt:
		t.visit(node.Exp, node, childData)
		t.visit(node.Pat, node, childData)
		for _, decl := range node.Binds {
			t.visit(decl, node, childData)
		}

	// Decls
	case *TypeSig:
		t.visit(node.Ty, node, childData)

	case *PatBind:
		t.visit(node.Pat, node, childData)
		t.visit(node.Rhs, node, childData)

	case *InstDecl:
		for i := range node.Assertions {
			t.visit(&node.Assertions[i], node, childData)
		}
		for _, ty := range node.Types {
			t.visit(ty, node, childData)
		}
		for _, decl := range node.Body {
			t.visit(decl, node, childData)
		}
	case *ClassDecl:
		for i := range node.Assertions {
			t.visit(&node.Assertions[i], node, childData)
		}
		for _, decl := range node.Decls {
			t.visit(decl, node, childData)
		}
		t.visit(&node.DHead, node, childData)
	case *DataDecl:
		for i := range node.Constructors {
			t.visit(&node.Constructors[i], node, childData)
		}
		for i := range node.Deriving {
			t.visit(&node.Deriving[i], node, childData)
		}
		t.visit(&node.DHead, node, childData)
	case *TypeDecl:
		t.visit(node.Ty, node, childData)
		t.visit(&node.DHead, node, childData)

	// Exp
	case *ExpVar:
	case *ExpApp:
		t.visit(node.Exp1, node, childData)
		t.visit(node.Exp2, node, childData)
	case *ExpInfix:
		t.visit(node.Exp1, node, childData)
		t.visit(node.Exp2, node, childData)
		t.visit(&node.Op, node, childData)
	case *ExpLambda:
		for _, pat := range node.Pats {
			t.visit(pat, node, childData)
		}
		t.visit(node.Exp, node, childData)
	case *ExpLet:
		for _, decl := range node.Binds {
			t.visit(decl, node, childData)
		}
		t.visit(node.Exp, node, childData)
	case *ExpIf:
		t.visit(node.Cond, node, childData)
		t.visit(node.IfTrue, node, childData)
		t.visit(node.IfFalse, node, childData)
	case *ExpDo:
		for _, stmt := range node.Stmts {
			t.visit(stmt, node, childData)
		}
	case *ExpCase:
		t.visit(node.Exp, node, childData)
		for _, alt := range node.Alts {
			t.visit(&alt, node, childData)
		}
	case *ExpTuple:
		for _, exp := range node.Exps {
			t.visit(exp, node, childData)
		}
	case *ExpList:
		for _, exp := range node.Exps {
			t.visit(exp, node, childData)
		}
	case *ExpLeftSection:
		t.visit(node.Left, node, childData)
		t.visit(node.Op, node, childData)
	case *ExpRightSection:
		t.visit(node.Right, node, childData)
		t.visit(node.Op, node, childData)
	case *ExpEnumFrom:
		t.visit(node.Exp, node, childData)
	case *ExpEnumFromTo:
		t.visit(node.Exp1, node, childData)
		t.visit(node.Exp2, node, childData)
	case *ExpComprehension:
		t.visit(node.Exp, node, childData)
		for _, gen := range node.Generators {
			t.visit(&gen, node, childData)
		}
		for _, guard := range node.Guards {
			t.visit(guard, node, childData)
		}
	case *Lit:
	// RHS
	case *GuardedRhs:
		for _, branch := range node.Branches {
			t.visit(&branch, node, childData)
		}
		for _, where := range node.Wheres {
			t.visit(where, node, childData)
		}
	case *UnguardedRhs:
		t.visit(node.Exp, node, childData)
		for _, decl := range node.Wheres {
			t.visit(decl, node, childData)
		}
	case *GuardBranch:
		t.visit(node.Exp, node, childData)
		for _, guard := range node.Guards {
			t.visit(guard, node, childData)
		}

	// Statements
	case *Generator:
		t.visit(node.Exp, node, childData)
		t.visit(node.Pat, node, childData)
	case *Qualifier:
		t.visit(node.Exp, node, childData)
	case *LetStmt:
		for _, decl := range node.Binds {
			t.visit(decl, node, childData)
		}

	// Pattern
	case *PWildcard:
	case *PApp:
		t.visit(&node.Constructor, node, childData)
		for _, pat := range node.Pats {
			t.visit(pat, node, childData)
		}
	case *PList:
		for _, pat := range node.Pats {
			t.visit(pat, node, childData)
		}
	case *PTuple:
		for _, pat := range node.Pats {
			t.visit(pat, node, childData)
		}
	case *PVar:
	case *PInfix:
		t.visit(node.Pat1, node, childData)
		t.visit(node.Pat2, node, childData)
		t.visit(&node.Op, node, childData)
	// Types
	case *TyCon:
	case *TyApp:
		t.visit(node.Ty1, node, childData)
		t.visit(node.Ty2, node, childData)
	case *TyFunction:
		t.visit(node.Ty1, node, childData)
		t.visit(node.Ty2, node, childData)
	case *TyTuple:
		for _, ty := range node.Tys {
			t.visit(ty, node, childData)
		}
	case *TyList:
		t.visit(node.Ty, node, childData)
	case *TyVar:
	case *TyForall:
		for i := range node.Assertions {
			t.visit(&node.Assertions[i], node, childData)
		}
		t.visit(node.Ty, node, childData)
	}

	if t.leave != nil {
		t.leave(childData, ast, parent)
	}
}
