package parser

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

type AST interface {
	pretty() string
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

type Loc struct {
	fromLine int
	toLine   int
	fromCol  int
	toCol    int
}

// FromLine returns the starting line number
func (l Loc) FromLine() int { return l.fromLine }

// ToLine returns the ending line number
func (l Loc) ToLine() int { return l.toLine }

// FromCol returns the starting column number
func (l Loc) FromCol() int { return l.fromCol }

// ToCol returns the ending column number
func (l Loc) ToCol() int { return l.toCol }

func mergeLoc(l1 Loc, l2 Loc) Loc {
	return Loc{
		fromLine: l1.fromLine,
		toLine:   l1.toLine,
		fromCol:  l2.fromCol,
		toCol:    l2.toCol,
	}
}

var noloc = Loc{
	0, 0, 0, 0,
}

type Node struct {
	id  int
	loc Loc
}

// Types
//
//	TyCon
type TyCon struct {
	name      string
	module    string
	canonical string
	axiom     bool
	Node
}

func (*TyCon) isType()        {}
func (t *TyCon) pretty() string {
	if t.name == "top" {
		return "()"
	}
	return t.name
}
func (n *TyCon) Loc() Loc     { return n.Node.loc }
func (n *TyCon) Id() int      { return n.Node.id }

// TyApp
type TyApp struct {
	ty1   Type
	ty2   Type
	axiom bool
	Node
}

func (*TyApp) isType()        {}
func (t *TyApp) pretty() string {
	return "(" + t.ty1.pretty() + " " + t.ty2.pretty() + ")"
}
func (n *TyApp) Loc() Loc     { return n.Node.loc }
func (n *TyApp) Id() int      { return n.Node.id }

// TyFunction
type TyFunction struct {
	ty1   Type
	ty2   Type
	axiom bool
	Node
}

func (*TyFunction) isType()        {}
func (t *TyFunction) pretty() string {
	return t.ty1.pretty() + " -> (" + t.ty2.pretty() + ")"
}
func (n *TyFunction) Loc() Loc     { return n.Node.loc }
func (n *TyFunction) Id() int      { return n.Node.id }

// TyTuple
type TyTuple struct {
	tys   []Type
	axiom bool
	Node
}

func (*TyTuple) isType()        {}
func (t *TyTuple) pretty() string {
	if len(t.tys) == 0 {
		return "()"
	}
	parts := make([]string, len(t.tys))
	for i, ty := range t.tys {
		parts[i] = ty.pretty()
	}
	return "(" + strings.Join(parts, ", ") + ")"
}
func (n *TyTuple) Loc() Loc     { return n.Node.loc }
func (n *TyTuple) Id() int      { return n.Node.id }

// TyList
type TyList struct {
	ty    Type
	axiom bool
	Node
}

func (*TyList) isType()        {}
func (t *TyList) pretty() string {
	return "[" + t.ty.pretty() + "]"
}
func (n *TyList) Loc() Loc     { return n.Node.loc }
func (n *TyList) Id() int      { return n.Node.id }

// TyVar
type TyVar struct {
	name      string
	canonical string
	axiom     bool
	Node
}

func (*TyVar) isType()        {}
func (t *TyVar) pretty() string { return t.name }
func (n *TyVar) Loc() Loc     { return n.Node.loc }
func (n *TyVar) Id() int      { return n.Node.id }

// TyForall
type TyForall struct {
	assertions []Type
	ty         Type
	axiom      bool
	Node
}

func (*TyForall) isType()        {}
func (tf *TyForall) pretty() string {
	result := ""
	
	// Add context/assertions if present (Eq a => ...)
	if len(tf.assertions) > 0 {
		assertStrs := make([]string, len(tf.assertions))
		for i, assertion := range tf.assertions {
			assertStrs[i] = assertion.pretty()
		}
		result += strings.Join(assertStrs, ", ") + " => "
	}
	
	// Add the actual type
	result += tf.ty.pretty()
	
	return result
}
func (n *TyForall) Loc() Loc     { return n.Node.loc }
func (n *TyForall) Id() int      { return n.Node.id }

// Pattern
//
//	PWildcard
type PWildcard struct {
	Node
}

func (*PWildcard) isPat()         {}
func (*PWildcard) pretty() string { return "_" }
func (n *PWildcard) Loc() Loc     { return n.Node.loc }
func (n *PWildcard) Id() int      { return n.Node.id }

// PApp
type PApp struct {
	constructor PVar
	pats        []Pat
	Node
}

func (*PApp) isPat()         {}
func (p *PApp) pretty() string {
	result := p.constructor.name
	if len(p.pats) > 0 {
		patStrs := make([]string, len(p.pats))
		for i, pat := range p.pats {
			patStr := pat.pretty()
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
func (n *PApp) Loc() Loc     { return n.Node.loc }
func (n *PApp) Id() int      { return n.Node.id }

// PList
type PList struct {
	pats []Pat
	Node
}

func (*PList) isPat()         {}
func (pl *PList) pretty() string {
	if len(pl.pats) == 0 {
		return "[]"
	}
	patStrs := make([]string, len(pl.pats))
	for i, pat := range pl.pats {
		patStrs[i] = pat.pretty()
	}
	return "[" + strings.Join(patStrs, ", ") + "]"
}
func (n *PList) Loc() Loc     { return n.Node.loc }
func (n *PList) Id() int      { return n.Node.id }

// PTuple
type PTuple struct {
	pats []Pat
	Node
}

func (*PTuple) isPat()         {}
func (pt *PTuple) pretty() string {
	if len(pt.pats) == 0 {
		return "()"
	}
	patStrs := make([]string, len(pt.pats))
	for i, pat := range pt.pats {
		patStrs[i] = pat.pretty()
	}
	return "(" + strings.Join(patStrs, ", ") + ")"
}
func (n *PTuple) Loc() Loc     { return n.Node.loc }
func (n *PTuple) Id() int      { return n.Node.id }

// PVar
type PVar struct {
	name      string
	canonical string
	module    string
	Node
}

func (*PVar) isPat()            {}
func (pv *PVar) pretty() string {
	if isOperator(pv.name) {
		return "(" + pv.name + ")"
	}
	return pv.name
}

// Name returns the variable name
func (pv *PVar) Name() string { return pv.name }
func (n *PVar) Loc() Loc        { return n.Node.loc }
func (n *PVar) Id() int         { return n.Node.id }

// PInfix
type PInfix struct {
	pat1 Pat
	op   PVar
	pat2 Pat
	Node
}

func (*PInfix) isPat()         {}
func (pi *PInfix) pretty() string {
	pat1Str := pi.pat1.pretty()
	pat2Str := pi.pat2.pretty()
	return "(" + pat1Str + " " + pi.op.name + " " + pat2Str + ")"
}
func (n *PInfix) Loc() Loc     { return n.Node.loc }
func (n *PInfix) Id() int      { return n.Node.id }

// ExpVar
type ExpVar struct {
	name      string
	module    string
	canonical string
	Node
}

func (*ExpVar) isExp() {}
func (v *ExpVar) pretty() string {
	if v.name == "unit" {
		return "()"
	}
	return v.name
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
// func (c *ExpCon) pretty() string {
//	if c.name == "unit" {
//		return "()"
//	}
//	return c.name
// }
// func (n *ExpCon) Loc() Loc {return n.Node.loc}

// ExpApp
type ExpApp struct {
	exp1 Exp
	exp2 Exp
	Node
}

func (*ExpApp) isExp() {}
func (e *ExpApp) pretty() string {
	return "(" + e.exp1.pretty() + " " + e.exp2.pretty() + ")"
}
func (n *ExpApp) Loc() Loc { return n.Node.loc }
func (n *ExpApp) Id() int  { return n.Node.id }

// ExpInfix
type ExpInfix struct {
	exp1 Exp
	exp2 Exp
	op   ExpVar
	Node
}

func (*ExpInfix) isExp() {}
func (ei *ExpInfix) pretty() string {
	return fmt.Sprintf("(%s %s %s)", ei.exp1.pretty(), ei.op.pretty(), ei.exp2.pretty())
}
func (n *ExpInfix) Loc() Loc { return n.Node.loc }
func (n *ExpInfix) Id() int  { return n.Node.id }

// ExpLambda
type ExpLambda struct {
	pats []Pat
	exp  Exp
	Node
}

func (*ExpLambda) isExp() {}
func (e *ExpLambda) pretty() string {
	pats := make([]string, len(e.pats))
	for i, pat := range e.pats {
		pats[i] = pat.pretty()
	}

	return "(\\" + strings.Join(pats, " ") + " -> " + e.exp.pretty() + ")"
}
func (n *ExpLambda) Loc() Loc { return n.Node.loc }
func (n *ExpLambda) Id() int  { return n.Node.id }

// ExpLet
type ExpLet struct {
	binds []Decl
	exp   Exp
	Node
}

func (*ExpLet) isExp()         {}
func (e *ExpLet) pretty() string {
	bindStrs := make([]string, len(e.binds))
	for i, bind := range e.binds {
			bindStrs[i] = bind.pretty()
	}
	bindStr := "{" + strings.Join(bindStrs, "; ") + "}"
	return "let " + bindStr + " in " + e.exp.pretty()
}
func (n *ExpLet) Loc() Loc     { return n.Node.loc }
func (n *ExpLet) Id() int      { return n.Node.id }

// ExpIf
type ExpIf struct {
	cond    Exp
	ifTrue  Exp
	ifFalse Exp
	Node
}

func (*ExpIf) isExp() {}
func (e *ExpIf) pretty() string {
	return "if " + e.cond.pretty() +
		" then " + e.ifTrue.pretty() +
		" else " + e.ifFalse.pretty()
}
func (n *ExpIf) Loc() Loc { return n.Node.loc }
func (n *ExpIf) Id() int  { return n.Node.id }

// ExpDo
type ExpDo struct {
	stmts []Statement
	Node
}

func (*ExpDo) isExp()         {}
func (e *ExpDo) pretty() string {
	stmtStrs := make([]string, len(e.stmts))
	for i, stmt := range e.stmts {
		stmtStrs[i] = stmt.pretty()
	}
	stmtStr := strings.Join(stmtStrs, "; ")
	return "do {" + stmtStr + "}"
}
func (n *ExpDo) Loc() Loc     { return n.Node.loc }
func (n *ExpDo) Id() int      { return n.Node.id }

// ExpCase
type ExpCase struct {
	exp  Exp
	alts []Alt
	Node
}

func (*ExpCase) isExp() {}
func (ec *ExpCase) pretty() string {
	alts := make([]string, len(ec.alts))
	for i, alt := range ec.alts {
		alts[i] = alt.pretty()
	}
	altsJoined := strings.Join(alts, "; ")
	return "case " + ec.exp.pretty() + " of " + altsJoined
}

func (n *ExpCase) Loc() Loc { return n.Node.loc }
func (n *ExpCase) Id() int  { return n.Node.id }

// ExpTuple
type ExpTuple struct {
	exps []Exp
	Node
}

func (*ExpTuple) isExp()         {}
func (e *ExpTuple) pretty() string {
	expStrs := make([]string, len(e.exps))
	for i, exp := range e.exps {
		expStrs[i] = exp.pretty()
	}
	expStr := strings.Join(expStrs, ", ")
	return "(" + expStr + ")"
}
func (n *ExpTuple) Loc() Loc     { return n.Node.loc }
func (n *ExpTuple) Id() int      { return n.Node.id }

// ExpList
type ExpList struct {
	exps []Exp
	Node
}

func (*ExpList) isExp()         {}
func (e *ExpList) pretty() string {
	expStrs := make([]string, len(e.exps))
	for i, exp := range e.exps {
		expStrs[i] = exp.pretty()
	}
	expStr := strings.Join(expStrs, ", ")
	return "[" + expStr + "]"
}
func (n *ExpList) Loc() Loc     { return n.Node.loc }
func (n *ExpList) Id() int      { return n.Node.id }

// ExpLeftSection
type ExpLeftSection struct {
	left Exp
	op   Exp
	Node
}

func (*ExpLeftSection) isExp()         {}
func (e *ExpLeftSection) pretty() string {
	return "(" + e.left.pretty() + " " + e.op.pretty() + ")"
}
func (n *ExpLeftSection) Loc() Loc     { return n.Node.loc }
func (n *ExpLeftSection) Id() int      { return n.Node.id }

// ExpRightSection
type ExpRightSection struct {
	right Exp
	op    Exp
	Node
}

func (*ExpRightSection) isExp()         {}
func (e *ExpRightSection) pretty() string {
	return "(" + e.op.pretty() + " " + e.right.pretty() + ")"
}
func (n *ExpRightSection) Loc() Loc     { return n.Node.loc }
func (n *ExpRightSection) Id() int      { return n.Node.id }

// ExpEnumFromTo
type ExpEnumFromTo struct {
	exp1 Exp
	exp2 Exp
	Node
}

func (*ExpEnumFromTo) isExp()         {}
func (e *ExpEnumFromTo) pretty() string {
	return "[" + e.exp1.pretty() + ".." + e.exp2.pretty() + "]"
}
func (n *ExpEnumFromTo) Loc() Loc     { return n.Node.loc }
func (n *ExpEnumFromTo) Id() int      { return n.Node.id }

// ExpEnumFrom
type ExpEnumFrom struct {
	exp Exp
	Node
}

func (*ExpEnumFrom) isExp()         {}
func (e *ExpEnumFrom) pretty() string {
	return "[" + e.exp.pretty() + ".." + "]"
}
func (n *ExpEnumFrom) Loc() Loc     { return n.Node.loc }
func (n *ExpEnumFrom) Id() int      { return n.Node.id }

// ExpComprehension
type ExpComprehension struct {
	exp        Exp
	generators []Generator
	guards     []Exp
	Node
}

func (*ExpComprehension) isExp() {}
func (e *ExpComprehension) pretty() string {
	generatorStrs := make([]string, len(e.generators))
	guardStrs := make([]string, len(e.guards))
	for i, generator := range e.generators {
		generatorStrs[i] = generator.pretty()
	}
	for i, guard := range e.guards {
		guardStrs[i] = guard.pretty()
	}
	s := strings.Join(append(generatorStrs, guardStrs...), ", ")
	return "[" + e.exp.pretty() + " | " + s + "]"

}
func (n *ExpComprehension) Loc() Loc     { return n.Node.loc }
func (n *ExpComprehension) Id() int      { return n.Node.id }

// Lit
type Lit struct {
	lit     string // integer/char/string/float
	content string
	Node
}

func (*Lit) isExp() {}
func (*Lit) isPat() {}
func (l *Lit) pretty() string {
	return l.content
}
func (n *Lit) Loc() Loc { return n.Node.loc }
func (n *Lit) Id() int  { return n.Node.id }

// RHS
//
//	UnguardedRhs
type UnguardedRhs struct {
	exp    Exp
	wheres []Decl
	Node
}

func (*UnguardedRhs) isRhs() {}
func (ur *UnguardedRhs) pretty() string {
	wheres := make([]string, len(ur.wheres))
	for i, where := range ur.wheres {
		wheres[i] = where.pretty()
	}

	expStr := ""
	if ur.exp != nil {
		expStr = ur.exp.pretty()
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
	branches []GuardBranch
	wheres   []Decl
	Node
}

func (*GuardedRhs) isRhs()         {}
func (gr *GuardedRhs) pretty() string {
	branchStrs := make([]string, len(gr.branches))
	for i, branch := range gr.branches {
		branchStrs[i] = branch.pretty()
	}

	result := strings.Join(branchStrs, " ")

	if len(gr.wheres) > 0 {
		whereStrs := make([]string, len(gr.wheres))
		for i, where := range gr.wheres {
			whereStrs[i] = where.pretty()
		}
		result += " where {" + strings.Join(whereStrs, "; ") + "}"
	}

	return result
}
func (n *GuardedRhs) Loc() Loc     { return n.Node.loc }
func (n *GuardedRhs) Id() int      { return n.Node.id }

// GuardBranch
type GuardBranch struct {
	exp    Exp
	guards []Exp
	Node
}

func (gb *GuardBranch) pretty() string {
	guardStrs := make([]string, 0, len(gb.guards))
	for _, guard := range gb.guards {
		if guard != nil {
			guardStrs = append(guardStrs, guard.pretty())
		}
	}

	expStr := ""
	if gb.exp != nil {
		expStr = gb.exp.pretty()
	}

	return "| " + strings.Join(guardStrs, ", ") + " = " + expStr
}
func (n *GuardBranch) Loc() Loc     { return n.Node.loc }
func (n *GuardBranch) Id() int      { return n.Node.id }

// Statements

// Generator
type Generator struct {
	pat Pat
	exp Exp
	Node
}

func (*Generator) isStatement()   {}
func (g *Generator) pretty() string {
	return g.pat.pretty() + " <- " + g.exp.pretty()
}
func (n *Generator) Loc() Loc     { return n.Node.loc }
func (n *Generator) Id() int      { return n.Node.id }

// Qualifier
type Qualifier struct {
	exp Exp
	Node
}

func (*Qualifier) isStatement()   {}
func (q *Qualifier) pretty() string { return q.exp.pretty() }
func (n *Qualifier) Loc() Loc     { return n.Node.loc }
func (n *Qualifier) Id() int      { return n.Node.id }

// LetStmt
type LetStmt struct {
	binds []Decl
	Node
}

func (*LetStmt) isStatement()   {}
func (l *LetStmt) pretty() string {
	bindStrs := make([]string, len(l.binds))
	for i, bind := range l.binds {
		bindStrs[i] = bind.pretty()
	}
	return "let " + strings.Join(bindStrs, "; ")
}
func (n *LetStmt) Loc() Loc     { return n.Node.loc }
func (n *LetStmt) Id() int      { return n.Node.id }

// Declarations

// TypeDecl
type TypeDecl struct {
	dHead DeclHead
	ty    Type
	Node
}

func (*TypeDecl) isDecl()        {}
func (td *TypeDecl) pretty() string {
	return "type " + td.dHead.pretty() + " = " + td.ty.pretty()
}
func (n *TypeDecl) Loc() Loc     { return n.Node.loc }
func (n *TypeDecl) Id() int      { return n.Node.id }

// DataDecl
type DataDecl struct {
	dHead        DeclHead
	constructors []DataCon
	deriving     []TyCon
	Node
}

func (*DataDecl) isDecl()        {}
func (dd *DataDecl) pretty() string {
	// Build the constructors: "Con1 ... | Con2 ..."
	conStrs := make([]string, len(dd.constructors))
	for i, con := range dd.constructors {
		conStrs[i] = con.pretty()
	}
	result := "data " + dd.dHead.pretty() + " = " + strings.Join(conStrs, " | ")

	// Add deriving clause if present
	if len(dd.deriving) > 0 {
		derivingStrs := make([]string, len(dd.deriving))
		for i, d := range dd.deriving {
			derivingStrs[i] = d.pretty()
		}
		result += " deriving (" + strings.Join(derivingStrs, ", ") + ")"
	}

	return result
}
func (n *DataDecl) Loc() Loc     { return n.Node.loc }
func (n *DataDecl) Id() int      { return n.Node.id }

// ClassDecl
type ClassDecl struct {
	assertions []Type
	dHead      DeclHead
	decls      []Decl
	Node
}

func (*ClassDecl) isDecl()        {}
func (cd *ClassDecl) pretty() string {
	result := "class "

	// Add context/assertions if present
	if len(cd.assertions) > 0 {
		assertStrs := make([]string, len(cd.assertions))
		for i, assertion := range cd.assertions {
			assertStrs[i] = assertion.pretty()
		}
		result += strings.Join(assertStrs, ", ") + " => "
	}

	// Add class head
	result += cd.dHead.pretty()

	// Add where clause with declarations
	if len(cd.decls) > 0 {
		result += " where "
		declStrs := make([]string, len(cd.decls))
		for i, decl := range cd.decls {
			declStrs[i] = decl.pretty()
		}
		result += strings.Join(declStrs, "; ")
	}

	return result
}
func (n *ClassDecl) Loc() Loc     { return n.Node.loc }
func (n *ClassDecl) Id() int      { return n.Node.id }

// InstDecl
type InstDecl struct {
	assertions []Type
	name       string
	module     string
	canonical  string
	types      []Type
	body       []Decl
	Node
}

func (*InstDecl) isDecl()        {}
func (id *InstDecl) pretty() string {
	result := "instance "

	// Add context/assertions if present
	if len(id.assertions) > 0 {
		assertStrs := make([]string, len(id.assertions))
		for i, assertion := range id.assertions {
			assertStrs[i] = assertion.pretty()
		}
		result += strings.Join(assertStrs, ", ") + " => "
	}

	// Add instance head (class name and types)
	result += id.name
	if len(id.types) > 0 {
		result += " "
		tyStrs := make([]string, len(id.types))
		for i, ty := range id.types {
			tyStrs[i] = ty.pretty()
		}
		result += strings.Join(tyStrs, " ")
	}

	// Add where clause with body if present
	if len(id.body) > 0 {
		result += " where "
		bodyStrs := make([]string, len(id.body))
		for i, decl := range id.body {
			bodyStrs[i] = decl.pretty()
		}
		result += strings.Join(bodyStrs, "; ")
	}

	return result
}
func (n *InstDecl) Loc() Loc     { return n.Node.loc }
func (n *InstDecl) Id() int      { return n.Node.id }

// PatBind
type PatBind struct {
	pat Pat
	rhs Rhs
	Node
}

func (*PatBind) isDecl() {}
func (pb *PatBind) pretty() string {
	return pb.pat.pretty() + " = " + pb.rhs.pretty()
}
func (n *PatBind) Loc() Loc { return n.Node.loc }
func (n *PatBind) Id() int  { return n.Node.id }

// TypeSig
type TypeSig struct {
	names      []string
	canonicals []string
	ty         Type
	Node
}

func (*TypeSig) isDecl()        {}
func (t *TypeSig) pretty() string {
	// Format names, wrapping operator names in parentheses
	formattedNames := make([]string, len(t.names))
	for i, name := range t.names {
		if isOperator(name) {
			formattedNames[i] = "(" + name + ")"
		} else {
			formattedNames[i] = name
		}
	}
	return strings.Join(formattedNames, ", ") + " :: " + t.ty.pretty()
}

// Names returns the names declared in this type signature
func (t *TypeSig) Names() []string { return t.names }

func (n *TypeSig) Loc() Loc     { return n.Node.loc }
func (n *TypeSig) Id() int      { return n.Node.id }

// Misc

// Alt
type Alt struct {
	pat   Pat
	exp   Exp
	binds []Decl
	Node
}

func (a *Alt) pretty() string {
	var bindStr string = "";
	if len(a.binds) > 0 {
		bindStrs := make([]string, len(a.binds))
		for i, b := range a.binds {
			bindStrs[i] = b.pretty()
		}
		bindStr = " where {" + strings.Join(bindStrs, "; ") + "}"
	}
	return a.pat.pretty() + " -> " + a.exp.pretty() + bindStr
}

func (n *Alt) Loc() Loc { return n.Node.loc }
func (n *Alt) Id() int  { return n.Node.id }

// DataCon
type DataCon struct {
	name      string
	canonical string
	tys       []Type
	Node
}

func (dc *DataCon) pretty() string {
	result := dc.name
	if len(dc.tys) > 0 {
		tyStrs := make([]string, len(dc.tys))
		for i, ty := range dc.tys {
			tyStrs[i] = ty.pretty()
		}
		result += " " + strings.Join(tyStrs, " ")
	}
	return result
}
func (n *DataCon) Loc() Loc     { return n.Node.loc }
func (n *DataCon) Id() int      { return n.Node.id }

// DeclHead
type DeclHead struct {
	name      string
	canonical string
	typeVars  []TyVar
	Node
}

func (dh *DeclHead) pretty() string {
	result := dh.name
	if len(dh.typeVars) > 0 {
		vars := make([]string, len(dh.typeVars))
		for i, v := range dh.typeVars {
			vars[i] = v.pretty()
		}
		result += " " + strings.Join(vars, " ")
	}
	return result
}
func (n *DeclHead) Loc() Loc     { return n.Node.loc }
func (n *DeclHead) Id() int      { return n.Node.id }

// Import represents a single import statement
type Import struct {
	module    string          // Module name (e.g., "Data.List")
	qualified bool            // true for "import qualified"
	alias     string          // Alias name for "import ... as X" (empty if not present)
	items     []string        // Imported items (empty if importing everything)
	hiding    bool            // true for "import ... hiding (...)"
	Node
}

func (i *Import) pretty() string {
	result := "import "
	if i.qualified {
		result += "qualified "
	}
	result += i.module

	if i.alias != "" {
		result += " as " + i.alias
	}

	if len(i.items) > 0 {
		if i.hiding {
			result += " hiding ("
		} else {
			result += " ("
		}
		result += strings.Join(i.items, ", ")
		result += ")"
	}

	return result
}
func (n *Import) Loc() Loc { return n.Node.loc }
func (n *Import) Id() int  { return n.Node.id }

// Module
type Module struct {
	name    string
	decls   []Decl
	imports []Import
	Node
}

// Name returns the module name
func (m *Module) Name() string { return m.name }

// Decls returns the module declarations
func (m *Module) Decls() []Decl { return m.decls }

// Imports returns the module imports
func (m *Module) Imports() []Import { return m.imports }

func (m *Module) pretty() string {
	t := `module {{ .Name }} where
{{- range .Imports }}
{{ . }}
{{- end }}
{{- range .Decls }}
{{ . }}
{{- end }}`

	imports := make([]string, len(m.imports))
	for i, imp := range m.imports {
		imports[i] = imp.pretty()
	}

	decls := make([]string, len(m.decls))
	for i, decl := range m.decls {
		decls[i] = decl.pretty()
	}
	return render(t, "module", struct {
		Name    string
		Imports []string
		Decls   []string
	}{
		m.name, imports, decls,
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

// Pattern accessor methods
func (p *PApp) Pats() []Pat { return p.pats }
func (p *PList) Pats() []Pat { return p.pats }
func (p *PTuple) Pats() []Pat { return p.pats }
func (p *PInfix) Pat1() Pat { return p.pat1 }
func (p *PInfix) Pat2() Pat { return p.pat2 }

// Accessor methods for declarations and expressions
func (pb *PatBind) Pat() Pat { return pb.pat }
func (dc *DataCon) Name() string { return dc.name }
func (dd *DataDecl) DeclHead() DeclHead { return dd.dHead }
func (cd *ClassDecl) DeclHead() DeclHead { return cd.dHead }
func (ec *ExpComprehension) Generators() []Generator { return ec.generators }
func (ed *ExpDo) Stmts() []Statement { return ed.stmts }
func (el *ExpLambda) Pats() []Pat { return el.pats }
func (a *Alt) Pat() Pat { return a.pat }
func (g *Generator) Pat() Pat { return g.pat }
func (dh *DeclHead) Name() string { return dh.name }
func (pb *PatBind) Rhs() Rhs { return pb.rhs }
