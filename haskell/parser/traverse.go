package parser

type Traverser[T any] struct {
	sub  func(v T, ast AST, parent int) T
	data T
}

func (t Traverser[T]) visit(ast AST, parentId int) {
	t.sub(t.data, ast, parentId)
	switch node := ast.(type) {
	case *Module:
		for _, decl := range node.decls {
			t.visit(decl, node.id)
		}
	// Misc
	case *DeclHead:
		for _, typeVar := range node.typeVars {
			t.visit(&typeVar, node.id)
		}
	case *DataCon:
		for _, ty := range node.tys {
			t.visit(ty, node.id)
		}
	case *Alt:
		t.visit(node.exp, node.id)
		t.visit(node.pat, node.id)
		for _, decl := range node.binds {
			t.visit(decl, node.id)
		}

	// Decls
	case *TypeSig:
		t.visit(node.ty, node.id)

	case *PatBind:
		t.visit(node.pat, node.id)
		t.visit(node.rhs, node.id)

	case *InstDecl:
		for _, assertion := range node.assertions {
			t.visit(assertion, node.id)
		}
		for _, ty := range node.types {
			t.visit(ty, node.id)
		}
		for _, decl := range node.body {
			t.visit(decl, node.id)
		}
	case *ClassDecl:
		for _, assertion := range node.assertions {
			t.visit(assertion, node.id)
		}
		for _, decl := range node.decls {
			t.visit(decl, node.id)
		}
		t.visit(&node.dHead, node.id)
	case *DataDecl:
		for _, constructor := range node.constructors {
			t.visit(&constructor, node.id)
		}
		for _, derive := range node.deriving {
			t.visit(&derive, node.id)
		}
		t.visit(&node.dHead, node.id)
	case *TypeDecl:
		t.visit(node.ty, node.id)
		t.visit(&node.dHead, node.id)

	// Exp
	case *ExpVar:
	case *ExpApp:
		t.visit(node.exp1, node.id)
		t.visit(node.exp2, node.id)
	case *ExpInfix:
		t.visit(node.exp1, node.id)
		t.visit(node.exp2, node.id)
		t.visit(&node.op, node.id)
	case *ExpLambda:
		for _, pat := range node.pats {
			t.visit(pat, node.id)
		}
		t.visit(node.exp, node.id)
	case *ExpLet:
		t.visit(node.exp, node.id)
		for _, decl := range node.binds {
			t.visit(decl, node.id)
		}
	case *ExpIf:
		t.visit(node.cond, node.id)
		t.visit(node.ifTrue, node.id)
		t.visit(node.ifFalse, node.id)
	case *ExpDo:
		for _, stmt := range node.stmts {
			t.visit(stmt, node.id)
		}
	case *ExpCase:
		t.visit(node.exp, node.id)
		for _, alt := range node.alts {
			t.visit(&alt, node.id)
		}
	case *ExpTuple:
		for _, exp := range node.exps {
			t.visit(exp, node.id)
		}
	case *ExpList:
		for _, exp := range node.exps {
			t.visit(exp, node.id)
		}
	case *ExpLeftSection:
		t.visit(node.left, node.id)
		t.visit(node.op, node.id)
	case *ExpRightSection:
		t.visit(node.right, node.id)
		t.visit(node.op, node.id)
	case *ExpEnumFrom:
		t.visit(node.exp, node.id)
	case *ExpEnumFromTo:
		t.visit(node.exp1, node.id)
		t.visit(node.exp2, node.id)
	case *ExpComprehension:
		t.visit(node.exp, node.id)
		for _, quantifier := range node.quantifiers {
			t.visit(&quantifier, node.id)
		}
		for _, guard := range node.guards {
			t.visit(guard, node.id)
		}
	case *Lit:
	// RHS
	case *GuardedRhs:
		for _, branch := range node.branches {
			t.visit(&branch, node.id)
		}
		for _, where := range node.wheres {
			t.visit(where, node.id)
		}
	case *UnguardedRhs:
		t.visit(node.exp, node.id)
		for _, decl := range node.wheres {
			t.visit(decl, node.id)
		}
	case *GuardBranch:
		t.visit(node.exp, node.id)
		for _, guard := range node.guards {
			t.visit(guard, node.id)
		}

	// Statements
	case *Generator:
		t.visit(node.exp, node.id)
		t.visit(node.pat, node.id)
	case *Qualifier:
		t.visit(node.exp, node.id)
	case *LetStmt:
		for _, decl := range node.binds {
			t.visit(decl, node.id)
		}

	// Pattern
	case *PWildcard:
	case *PApp:
		for _, pat := range node.pats {
			t.visit(pat, node.id)
		}
	case *PList:
		for _, pat := range node.pats {
			t.visit(pat, node.id)
		}
	case *PTuple:
		for _, pat := range node.pats {
			t.visit(pat, node.id)
		}
	case *PVar:
	case *PInfix:
		t.visit(node.pat1, node.id)
		t.visit(node.pat2, node.id)
	// Types
	case *TyCon:
	case *TyApp:
		t.visit(node.ty1, node.id)
		t.visit(node.ty2, node.id)
	case *TyFunction:
		t.visit(node.ty1, node.id)
		t.visit(node.ty2, node.id)
	case *TyTuple:
		for _, ty := range node.tys {
			t.visit(ty, node.id)
		}
	case *TyList:
		t.visit(node.ty, node.id)
	case *TyPrefixFunction:
	case *TyPrefixList:
	case *TyPrefixTuple:
	case *TyVar:
	case *TyForall:
		for _, assertion := range node.assertions {
			t.visit(assertion, node.id)
		}
		t.visit(node.ty, node.id)
	}

}
