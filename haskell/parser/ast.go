package parser

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

type AST interface {
	Pretty() string
	Loc() Loc
	Id() int
}

type Pat interface {
	AST
	isPat()
}

type Type interface {
	AST
	isType()
}

type Exp interface {
	AST
	isExp()
}

type Statement interface {
	AST
	isStatement()
}

type Decl interface {
	AST
	isDecl()
}

type Rhs interface {
	AST
	isRhs()
}

// Name is implemented by all node types that carry a Canonical field.
type Name interface {
	AST
	SetCanonical(string)
}


type Node struct {
	id  int
	loc Loc
}

// Types
//
//	TyCon
type TyCon struct {
	Name      string
	Module    string
	Canonical string
	Axiom     bool
	Node
}

func (*TyCon) isType()                    {}
func (t *TyCon) SetCanonical(s string)   { t.Canonical = s }
func (t *TyCon) Pretty() string {
	if t.Name == "top" {
		return "()"
	}
	return t.Name
}
func (n *TyCon) Loc() Loc { return n.Node.loc }
func (n *TyCon) Id() int  { return n.Node.id }

// TyApp
type TyApp struct {
	Ty1   Type
	Ty2   Type
	Axiom bool
	Node
}

func (*TyApp) isType() {}
func (t *TyApp) Pretty() string {
	return "(" + t.Ty1.Pretty() + " " + t.Ty2.Pretty() + ")"
}
func (n *TyApp) Loc() Loc { return n.Node.loc }
func (n *TyApp) Id() int  { return n.Node.id }

// TyFunction
type TyFunction struct {
	Ty1   Type
	Ty2   Type
	Axiom bool
	Node
}

func (*TyFunction) isType() {}
func (t *TyFunction) Pretty() string {
	return t.Ty1.Pretty() + " -> (" + t.Ty2.Pretty() + ")"
}
func (n *TyFunction) Loc() Loc { return n.Node.loc }
func (n *TyFunction) Id() int  { return n.Node.id }

// TyTuple
type TyTuple struct {
	Tys   []Type
	Axiom bool
	Node
}

func (*TyTuple) isType() {}
func (t *TyTuple) Pretty() string {
	if len(t.Tys) == 0 {
		return "()"
	}
	parts := make([]string, len(t.Tys))
	for i, ty := range t.Tys {
		parts[i] = ty.Pretty()
	}
	return "(" + strings.Join(parts, ", ") + ")"
}
func (n *TyTuple) Loc() Loc { return n.Node.loc }
func (n *TyTuple) Id() int  { return n.Node.id }

// TyList
type TyList struct {
	Ty    Type
	Axiom bool
	Node
}

func (*TyList) isType() {}
func (t *TyList) Pretty() string {
	return "[" + t.Ty.Pretty() + "]"
}
func (n *TyList) Loc() Loc { return n.Node.loc }
func (n *TyList) Id() int  { return n.Node.id }

// TyVar
type TyVar struct {
	Name      string
	Canonical string
	Axiom     bool
	Node
}

func (*TyVar) isType()                    {}
func (t *TyVar) SetCanonical(s string)   { t.Canonical = s }
func (t *TyVar) Pretty() string { return t.Name }
func (n *TyVar) Loc() Loc       { return n.Node.loc }
func (n *TyVar) Id() int        { return n.Node.id }

// Assertion represents a typeclass constraint, e.g. "Eq a" or "Functor f"
type Assertion struct {
	Name      string
	Module    string
	Canonical string
	Types     []Type
	Node
}

func (a *Assertion) SetCanonical(s string) { a.Canonical = s }
func (a *Assertion) Pretty() string {
	name := a.Name
	if a.Module != "" {
		name = a.Module + "." + a.Name
	}
	if len(a.Types) == 0 {
		return name
	}
	parts := make([]string, len(a.Types))
	for i, t := range a.Types {
		parts[i] = t.Pretty()
	}
	return name + " " + strings.Join(parts, " ")
}
func (n *Assertion) Loc() Loc { return n.Node.loc }
func (n *Assertion) Id() int  { return n.Node.id }

// TyForall
type TyForall struct {
	Assertions []Assertion
	Ty         Type
	Axiom      bool
	Node
}

func (*TyForall) isType() {}
func (tf *TyForall) Pretty() string {
	result := ""

	// Add context/assertions if present (Eq a => ...)
	if len(tf.Assertions) > 0 {
		assertStrs := make([]string, len(tf.Assertions))
		for i := range tf.Assertions {
			assertStrs[i] = tf.Assertions[i].Pretty()
		}
		result += strings.Join(assertStrs, ", ") + " => "
	}

	// Add the actual type
	result += tf.Ty.Pretty()

	return result
}
func (n *TyForall) Loc() Loc { return n.Node.loc }
func (n *TyForall) Id() int  { return n.Node.id }

// Pattern
//
//	PWildcard
type PWildcard struct {
	Node
}

func (*PWildcard) isPat()         {}
func (*PWildcard) Pretty() string { return "_" }
func (n *PWildcard) Loc() Loc     { return n.Node.loc }
func (n *PWildcard) Id() int      { return n.Node.id }

// PApp
type PApp struct {
	Constructor PVar
	Pats        []Pat
	Node
}

func (*PApp) isPat() {}
func (p *PApp) Pretty() string {
	result := p.Constructor.Name
	if len(p.Pats) > 0 {
		patStrs := make([]string, len(p.Pats))
		for i, pat := range p.Pats {
			patStr := pat.Pretty()
			// Wrap PApp patterns in parentheses when they appear as arguments
			if _, isPApp := pat.(*PApp); isPApp {
				patStr = "(" + patStr + ")"
			}
			patStrs[i] = patStr
		}
		result += " " + strings.Join(patStrs, " ")
	}
	return result
}
func (n *PApp) Loc() Loc { return n.Node.loc }
func (n *PApp) Id() int  { return n.Node.id }

// PList
type PList struct {
	Pats []Pat
	Node
}

func (*PList) isPat() {}
func (pl *PList) Pretty() string {
	if len(pl.Pats) == 0 {
		return "[]"
	}
	patStrs := make([]string, len(pl.Pats))
	for i, pat := range pl.Pats {
		patStrs[i] = pat.Pretty()
	}
	return "[" + strings.Join(patStrs, ", ") + "]"
}
func (n *PList) Loc() Loc { return n.Node.loc }
func (n *PList) Id() int  { return n.Node.id }

// PTuple
type PTuple struct {
	Pats []Pat
	Node
}

func (*PTuple) isPat() {}
func (pt *PTuple) Pretty() string {
	if len(pt.Pats) == 0 {
		return "()"
	}
	patStrs := make([]string, len(pt.Pats))
	for i, pat := range pt.Pats {
		patStrs[i] = pat.Pretty()
	}
	return "(" + strings.Join(patStrs, ", ") + ")"
}
func (n *PTuple) Loc() Loc { return n.Node.loc }
func (n *PTuple) Id() int  { return n.Node.id }

// PVar
type PVar struct {
	Name      string
	Canonical string
	Module    string
	Node
}

func (*PVar) isPat()                      {}
func (p *PVar) SetCanonical(s string)    { p.Canonical = s }
func (pv *PVar) Pretty() string {
	if isOperator(pv.Name) {
		return "(" + pv.Name + ")"
	}
	return pv.Name
}
func (n *PVar) Loc() Loc { return n.Node.loc }
func (n *PVar) Id() int  { return n.Node.id }

// PInfix
type PInfix struct {
	Pat1 Pat
	Op   PVar
	Pat2 Pat
	Node
}

func (*PInfix) isPat() {}
func (pi *PInfix) Pretty() string {
	pat1Str := pi.Pat1.Pretty()
	pat2Str := pi.Pat2.Pretty()
	return "(" + pat1Str + " " + pi.Op.Name + " " + pat2Str + ")"
}
func (n *PInfix) Loc() Loc { return n.Node.loc }
func (n *PInfix) Id() int  { return n.Node.id }

// ExpVar
type ExpVar struct {
	Name      string
	Module    string
	Canonical string
	Node
}

func (*ExpVar) isExp()                    {}
func (v *ExpVar) SetCanonical(s string)  { v.Canonical = s }
func (v *ExpVar) Pretty() string {
	if v.Name == "unit" {
		return "()"
	}
	return v.Name
}
func (n *ExpVar) Loc() Loc { return n.Node.loc }
func (n *ExpVar) Id() int  { return n.Node.id }

// ExpCon
// type ExpCon struct {
//	name      string
//	canonical string
//	module    string
//	Node
// }

// func (*ExpCon) isExp()         {}
// func (c *ExpCon) Pretty() string {
//	if c.name == "unit" {
//		return "()"
//	}
//	return c.name
// }
// func (n *ExpCon) Loc() Loc {return n.Node.loc}

// ExpApp
type ExpApp struct {
	Exp1 Exp
	Exp2 Exp
	Node
}

func (*ExpApp) isExp() {}
func (e *ExpApp) Pretty() string {
	return "(" + e.Exp1.Pretty() + " " + e.Exp2.Pretty() + ")"
}
func (n *ExpApp) Loc() Loc { return n.Node.loc }
func (n *ExpApp) Id() int  { return n.Node.id }

// ExpInfix
type ExpInfix struct {
	Exp1 Exp
	Exp2 Exp
	Op   ExpVar
	Node
}

func (*ExpInfix) isExp() {}
func (ei *ExpInfix) Pretty() string {
	return fmt.Sprintf("(%s %s %s)", ei.Exp1.Pretty(), ei.Op.Pretty(), ei.Exp2.Pretty())
}
func (n *ExpInfix) Loc() Loc { return n.Node.loc }
func (n *ExpInfix) Id() int  { return n.Node.id }

// ExpLambda
type ExpLambda struct {
	Pats []Pat
	Exp  Exp
	Node
}

func (*ExpLambda) isExp() {}
func (e *ExpLambda) Pretty() string {
	pats := make([]string, len(e.Pats))
	for i, pat := range e.Pats {
		pats[i] = pat.Pretty()
	}

	return "(\\" + strings.Join(pats, " ") + " -> " + e.Exp.Pretty() + ")"
}
func (n *ExpLambda) Loc() Loc { return n.Node.loc }
func (n *ExpLambda) Id() int  { return n.Node.id }

// ExpLet
type ExpLet struct {
	Binds []Decl
	Exp   Exp
	Node
}

func (*ExpLet) isExp() {}
func (e *ExpLet) Pretty() string {
	bindStrs := make([]string, len(e.Binds))
	for i, bind := range e.Binds {
		bindStrs[i] = bind.Pretty()
	}
	bindStr := "{" + strings.Join(bindStrs, "; ") + "}"
	return "let " + bindStr + " in " + e.Exp.Pretty()
}
func (n *ExpLet) Loc() Loc { return n.Node.loc }
func (n *ExpLet) Id() int  { return n.Node.id }

// ExpIf
type ExpIf struct {
	Cond    Exp
	IfTrue  Exp
	IfFalse Exp
	Node
}

func (*ExpIf) isExp() {}
func (e *ExpIf) Pretty() string {
	return "if " + e.Cond.Pretty() +
		" then " + e.IfTrue.Pretty() +
		" else " + e.IfFalse.Pretty()
}
func (n *ExpIf) Loc() Loc { return n.Node.loc }
func (n *ExpIf) Id() int  { return n.Node.id }

// ExpDo
type ExpDo struct {
	Stmts []Statement
	Node
}

func (*ExpDo) isExp() {}
func (e *ExpDo) Pretty() string {
	stmtStrs := make([]string, len(e.Stmts))
	for i, stmt := range e.Stmts {
		stmtStrs[i] = stmt.Pretty()
	}
	stmtStr := strings.Join(stmtStrs, "; ")
	return "do {" + stmtStr + "}"
}
func (n *ExpDo) Loc() Loc { return n.Node.loc }
func (n *ExpDo) Id() int  { return n.Node.id }

// ExpCase
type ExpCase struct {
	Exp  Exp
	Alts []Alt
	Node
}

func (*ExpCase) isExp() {}
func (ec *ExpCase) Pretty() string {
	alts := make([]string, len(ec.Alts))
	for i, alt := range ec.Alts {
		alts[i] = alt.Pretty()
	}
	altsJoined := strings.Join(alts, "; ")
	return "case " + ec.Exp.Pretty() + " of " + altsJoined
}

func (n *ExpCase) Loc() Loc { return n.Node.loc }
func (n *ExpCase) Id() int  { return n.Node.id }

// ExpTuple
type ExpTuple struct {
	Exps []Exp
	Node
}

func (*ExpTuple) isExp() {}
func (e *ExpTuple) Pretty() string {
	expStrs := make([]string, len(e.Exps))
	for i, exp := range e.Exps {
		expStrs[i] = exp.Pretty()
	}
	expStr := strings.Join(expStrs, ", ")
	return "(" + expStr + ")"
}
func (n *ExpTuple) Loc() Loc { return n.Node.loc }
func (n *ExpTuple) Id() int  { return n.Node.id }

// ExpList
type ExpList struct {
	Exps []Exp
	Node
}

func (*ExpList) isExp() {}
func (e *ExpList) Pretty() string {
	expStrs := make([]string, len(e.Exps))
	for i, exp := range e.Exps {
		expStrs[i] = exp.Pretty()
	}
	expStr := strings.Join(expStrs, ", ")
	return "[" + expStr + "]"
}
func (n *ExpList) Loc() Loc { return n.Node.loc }
func (n *ExpList) Id() int  { return n.Node.id }

// ExpLeftSection
type ExpLeftSection struct {
	Left Exp
	Op   Exp
	Node
}

func (*ExpLeftSection) isExp() {}
func (e *ExpLeftSection) Pretty() string {
	return "(" + e.Left.Pretty() + " " + e.Op.Pretty() + ")"
}
func (n *ExpLeftSection) Loc() Loc { return n.Node.loc }
func (n *ExpLeftSection) Id() int  { return n.Node.id }

// ExpRightSection
type ExpRightSection struct {
	Right Exp
	Op    Exp
	Node
}

func (*ExpRightSection) isExp() {}
func (e *ExpRightSection) Pretty() string {
	return "(" + e.Op.Pretty() + " " + e.Right.Pretty() + ")"
}
func (n *ExpRightSection) Loc() Loc { return n.Node.loc }
func (n *ExpRightSection) Id() int  { return n.Node.id }

// ExpEnumFromTo
type ExpEnumFromTo struct {
	Exp1 Exp
	Exp2 Exp
	Node
}

func (*ExpEnumFromTo) isExp() {}
func (e *ExpEnumFromTo) Pretty() string {
	return "[" + e.Exp1.Pretty() + ".." + e.Exp2.Pretty() + "]"
}
func (n *ExpEnumFromTo) Loc() Loc { return n.Node.loc }
func (n *ExpEnumFromTo) Id() int  { return n.Node.id }

// ExpEnumFrom
type ExpEnumFrom struct {
	Exp Exp
	Node
}

func (*ExpEnumFrom) isExp() {}
func (e *ExpEnumFrom) Pretty() string {
	return "[" + e.Exp.Pretty() + ".." + "]"
}
func (n *ExpEnumFrom) Loc() Loc { return n.Node.loc }
func (n *ExpEnumFrom) Id() int  { return n.Node.id }

// ExpComprehension
type ExpComprehension struct {
	Exp        Exp
	Generators []Generator
	Guards     []Exp
	Node
}

func (*ExpComprehension) isExp() {}
func (e *ExpComprehension) Pretty() string {
	generatorStrs := make([]string, len(e.Generators))
	guardStrs := make([]string, len(e.Guards))
	for i, generator := range e.Generators {
		generatorStrs[i] = generator.Pretty()
	}
	for i, guard := range e.Guards {
		guardStrs[i] = guard.Pretty()
	}
	s := strings.Join(append(generatorStrs, guardStrs...), ", ")
	return "[" + e.Exp.Pretty() + " | " + s + "]"

}
func (n *ExpComprehension) Loc() Loc { return n.Node.loc }
func (n *ExpComprehension) Id() int  { return n.Node.id }

// Lit
type Lit struct {
	Lit     string // integer/char/string/float
	Content string
	Node
}

func (*Lit) isExp() {}
func (*Lit) isPat() {}
func (l *Lit) Pretty() string {
	return l.Content
}
func (n *Lit) Loc() Loc { return n.Node.loc }
func (n *Lit) Id() int  { return n.Node.id }

// RHS
//
//	UnguardedRhs
type UnguardedRhs struct {
	Exp    Exp
	Wheres []Decl
	Node
}

func (*UnguardedRhs) isRhs() {}
func (ur *UnguardedRhs) Pretty() string {
	wheres := make([]string, len(ur.Wheres))
	for i, where := range ur.Wheres {
		wheres[i] = where.Pretty()
	}

	expStr := ""
	if ur.Exp != nil {
		expStr = ur.Exp.Pretty()
	}

	return render(`{{.Exp -}} {{ if gt (len .Wheres) 0}} where {{range .Wheres}} {{ . -}}; {{end}}{{end}}`, "UnguardedRhs", struct {
		Exp    string
		Wheres []string
	}{
		Exp:    expStr,
		Wheres: wheres,
	})
}
func (n *UnguardedRhs) Loc() Loc { return n.Node.loc }
func (n *UnguardedRhs) Id() int  { return n.Node.id }

// GuardedRhs
type GuardedRhs struct {
	Branches []GuardBranch
	Wheres   []Decl
	Node
}

func (*GuardedRhs) isRhs() {}
func (gr *GuardedRhs) Pretty() string {
	branchStrs := make([]string, len(gr.Branches))
	for i, branch := range gr.Branches {
		branchStrs[i] = branch.Pretty()
	}

	result := strings.Join(branchStrs, " ")

	if len(gr.Wheres) > 0 {
		whereStrs := make([]string, len(gr.Wheres))
		for i, where := range gr.Wheres {
			whereStrs[i] = where.Pretty()
		}
		result += " where {" + strings.Join(whereStrs, "; ") + "}"
	}

	return result
}
func (n *GuardedRhs) Loc() Loc { return n.Node.loc }
func (n *GuardedRhs) Id() int  { return n.Node.id }

// GuardBranch
type GuardBranch struct {
	Exp    Exp
	Guards []Exp
	Node
}

func (gb *GuardBranch) Pretty() string {
	guardStrs := make([]string, 0, len(gb.Guards))
	for _, guard := range gb.Guards {
		if guard != nil {
			guardStrs = append(guardStrs, guard.Pretty())
		}
	}

	expStr := ""
	if gb.Exp != nil {
		expStr = gb.Exp.Pretty()
	}

	return "| " + strings.Join(guardStrs, ", ") + " = " + expStr
}
func (n *GuardBranch) Loc() Loc { return n.Node.loc }
func (n *GuardBranch) Id() int  { return n.Node.id }

// Statements

// Generator
type Generator struct {
	Pat Pat
	Exp Exp
	Node
}

func (*Generator) isStatement() {}
func (g *Generator) Pretty() string {
	return g.Pat.Pretty() + " <- " + g.Exp.Pretty()
}
func (n *Generator) Loc() Loc { return n.Node.loc }
func (n *Generator) Id() int  { return n.Node.id }

// Qualifier
type Qualifier struct {
	Exp Exp
	Node
}

func (*Qualifier) isStatement()     {}
func (q *Qualifier) Pretty() string { return q.Exp.Pretty() }
func (n *Qualifier) Loc() Loc       { return n.Node.loc }
func (n *Qualifier) Id() int        { return n.Node.id }

// LetStmt
type LetStmt struct {
	Binds []Decl
	Node
}

func (*LetStmt) isStatement() {}
func (l *LetStmt) Pretty() string {
	bindStrs := make([]string, len(l.Binds))
	for i, bind := range l.Binds {
		bindStrs[i] = bind.Pretty()
	}
	return "let " + strings.Join(bindStrs, "; ")
}
func (n *LetStmt) Loc() Loc { return n.Node.loc }
func (n *LetStmt) Id() int  { return n.Node.id }

// Declarations

// TypeDecl
type TypeDecl struct {
	DHead DeclHead
	Ty    Type
	Node
}

func (*TypeDecl) isDecl() {}
func (td *TypeDecl) Pretty() string {
	return "type " + td.DHead.Pretty() + " = " + td.Ty.Pretty()
}
func (n *TypeDecl) Loc() Loc { return n.Node.loc }
func (n *TypeDecl) Id() int  { return n.Node.id }

// DataDecl
type DataDecl struct {
	DHead        DeclHead
	Constructors []DataCon
	Deriving     []TyCon
	Node
}

func (*DataDecl) isDecl() {}
func (dd *DataDecl) Pretty() string {
	// Build the constructors: "Con1 ... | Con2 ..."
	conStrs := make([]string, len(dd.Constructors))
	for i, con := range dd.Constructors {
		conStrs[i] = con.Pretty()
	}
	result := "data " + dd.DHead.Pretty() + " = " + strings.Join(conStrs, " | ")

	// Add deriving clause if present
	if len(dd.Deriving) > 0 {
		derivingStrs := make([]string, len(dd.Deriving))
		for i, d := range dd.Deriving {
			derivingStrs[i] = d.Pretty()
		}
		result += " deriving (" + strings.Join(derivingStrs, ", ") + ")"
	}

	return result
}
func (n *DataDecl) Loc() Loc { return n.Node.loc }
func (n *DataDecl) Id() int  { return n.Node.id }

// ClassDecl
type ClassDecl struct {
	Assertions []Assertion
	DHead      DeclHead
	Decls      []Decl
	Node
}

func (*ClassDecl) isDecl() {}
func (cd *ClassDecl) Pretty() string {
	result := "class "

	// Add context/assertions if present
	if len(cd.Assertions) > 0 {
		assertStrs := make([]string, len(cd.Assertions))
		for i := range cd.Assertions {
			assertStrs[i] = cd.Assertions[i].Pretty()
		}
		result += strings.Join(assertStrs, ", ") + " => "
	}

	// Add class head
	result += cd.DHead.Pretty()

	// Add where clause with declarations
	if len(cd.Decls) > 0 {
		result += " where "
		declStrs := make([]string, len(cd.Decls))
		for i, decl := range cd.Decls {
			declStrs[i] = decl.Pretty()
		}
		result += strings.Join(declStrs, "; ")
	}

	return result
}
func (n *ClassDecl) Loc() Loc { return n.Node.loc }
func (n *ClassDecl) Id() int  { return n.Node.id }

// InstDecl
type InstDecl struct {
	Assertions []Assertion
	Name       string
	Module     string
	Canonical  string
	Types      []Type
	Body       []Decl
	Node
}

func (*InstDecl) isDecl()                    {}
func (id *InstDecl) SetCanonical(s string)   { id.Canonical = s }
func (id *InstDecl) Pretty() string {
	result := "instance "

	// Add context/assertions if present
	if len(id.Assertions) > 0 {
		assertStrs := make([]string, len(id.Assertions))
		for i := range id.Assertions {
			assertStrs[i] = id.Assertions[i].Pretty()
		}
		result += strings.Join(assertStrs, ", ") + " => "
	}

	// Add instance head (class name and types)
	result += id.Name
	if len(id.Types) > 0 {
		result += " "
		tyStrs := make([]string, len(id.Types))
		for i, ty := range id.Types {
			tyStrs[i] = ty.Pretty()
		}
		result += strings.Join(tyStrs, " ")
	}

	// Add where clause with body if present
	if len(id.Body) > 0 {
		result += " where "
		bodyStrs := make([]string, len(id.Body))
		for i, decl := range id.Body {
			bodyStrs[i] = decl.Pretty()
		}
		result += strings.Join(bodyStrs, "; ")
	}

	return result
}
func (n *InstDecl) Loc() Loc { return n.Node.loc }
func (n *InstDecl) Id() int  { return n.Node.id }

// PatBind
type PatBind struct {
	Pat Pat
	Rhs Rhs
	Node
}

func (*PatBind) isDecl() {}
func (pb *PatBind) Pretty() string {
	return pb.Pat.Pretty() + " = " + pb.Rhs.Pretty()
}
func (n *PatBind) Loc() Loc { return n.Node.loc }
func (n *PatBind) Id() int  { return n.Node.id }

// TypeSig
type TypeSig struct {
	Names      []string
	Canonicals []string
	Ty         Type
	Node
}

func (*TypeSig) isDecl() {}
func (t *TypeSig) Pretty() string {
	// Format names, wrapping operator names in parentheses
	formattedNames := make([]string, len(t.Names))
	for i, name := range t.Names {
		if isOperator(name) {
			formattedNames[i] = "(" + name + ")"
		} else {
			formattedNames[i] = name
		}
	}
	return strings.Join(formattedNames, ", ") + " :: " + t.Ty.Pretty()
}

func (n *TypeSig) Loc() Loc { return n.Node.loc }
func (n *TypeSig) Id() int  { return n.Node.id }

// Misc

// Alt
type Alt struct {
	Pat   Pat
	Exp   Exp
	Binds []Decl
	Node
}

func (a *Alt) Pretty() string {
	var bindStr string = ""
	if len(a.Binds) > 0 {
		bindStrs := make([]string, len(a.Binds))
		for i, b := range a.Binds {
			bindStrs[i] = b.Pretty()
		}
		bindStr = " where {" + strings.Join(bindStrs, "; ") + "}"
	}
	return a.Pat.Pretty() + " -> " + a.Exp.Pretty() + bindStr
}

func (n *Alt) Loc() Loc { return n.Node.loc }
func (n *Alt) Id() int  { return n.Node.id }

// DataCon
type DataCon struct {
	Name      string
	Canonical string
	Tys       []Type
	Node
}

func (dc *DataCon) SetCanonical(s string) { dc.Canonical = s }
func (dc *DataCon) Pretty() string {
	result := dc.Name
	if len(dc.Tys) > 0 {
		tyStrs := make([]string, len(dc.Tys))
		for i, ty := range dc.Tys {
			tyStrs[i] = ty.Pretty()
		}
		result += " " + strings.Join(tyStrs, " ")
	}
	return result
}
func (n *DataCon) Loc() Loc { return n.Node.loc }
func (n *DataCon) Id() int  { return n.Node.id }

// DeclHead
type DeclHead struct {
	Name      string
	Canonical string
	TypeVars  []TyVar
	Node
}

func (dh *DeclHead) SetCanonical(s string) { dh.Canonical = s }
func (dh *DeclHead) Pretty() string {
	result := dh.Name
	if len(dh.TypeVars) > 0 {
		vars := make([]string, len(dh.TypeVars))
		for i, v := range dh.TypeVars {
			vars[i] = v.Pretty()
		}
		result += " " + strings.Join(vars, " ")
	}
	return result
}
func (n *DeclHead) Loc() Loc { return n.Node.loc }
func (n *DeclHead) Id() int  { return n.Node.id }

// Import represents a single import statement
type Import struct {
	Module    string   // Module name (e.g., "Data.List")
	Qualified bool     // true for "import qualified"
	Alias     string   // Alias name for "import ... as X" (empty if not present)
	Items     []string // Imported items (empty if importing everything)
	Hiding    bool     // true for "import ... hiding (...)"
	Node
}

func (i *Import) Pretty() string {
	result := "import "
	if i.Qualified {
		result += "qualified "
	}
	result += i.Module

	if i.Alias != "" {
		result += " as " + i.Alias
	}

	if len(i.Items) > 0 {
		if i.Hiding {
			result += " hiding ("
		} else {
			result += " ("
		}
		result += strings.Join(i.Items, ", ")
		result += ")"
	}

	return result
}
func (n *Import) Loc() Loc { return n.Node.loc }
func (n *Import) Id() int  { return n.Node.id }

// Module
type Module struct {
	Name    string
	Decls   []Decl
	Imports []Import
	Node
}

func (m *Module) Pretty() string {
	t := `module {{ .Name }} where
{{- range .Imports }}
{{ . }}
{{- end }}
{{- range .Decls }}
{{ . }}
{{- end }}`

	imports := make([]string, len(m.Imports))
	for i, imp := range m.Imports {
		imports[i] = imp.Pretty()
	}

	decls := make([]string, len(m.Decls))
	for i, decl := range m.Decls {
		decls[i] = decl.Pretty()
	}
	return render(t, "module", struct {
		Name    string
		Imports []string
		Decls   []string
	}{
		m.Name, imports, decls,
	})
}
func (n *Module) Loc() Loc { return n.Node.loc }
func (n *Module) Id() int  { return n.Node.id }

func render(temp string, name string, data any) string {
	var buf bytes.Buffer

	t := template.Must(template.New(name).Parse(temp))
	if err := t.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

// Helper function to check if a name is an operator (contains special characters)
func isOperator(name string) bool {
	if len(name) == 0 {
		return false
	}
	// Operators start with symbols, not alphanumeric characters
	firstChar := rune(name[0])
	return !((firstChar >= 'a' && firstChar <= 'z') ||
		(firstChar >= 'A' && firstChar <= 'Z') ||
		firstChar == '_')
}

