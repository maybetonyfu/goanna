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
func (*TyCon) pretty() string { return "" }
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
func (*TyApp) pretty() string { return "" }
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
func (*TyFunction) pretty() string { return "" }
func (n *TyFunction) Loc() Loc     { return n.Node.loc }
func (n *TyFunction) Id() int      { return n.Node.id }

// TyTuple
type TyTuple struct {
	tys   []Type
	axiom bool
	Node
}

func (*TyTuple) isType()        {}
func (*TyTuple) pretty() string { return "" }
func (n *TyTuple) Loc() Loc     { return n.Node.loc }
func (n *TyTuple) Id() int      { return n.Node.id }

// TyList
type TyList struct {
	ty    Type
	axiom bool
	Node
}

func (*TyList) isType()        {}
func (*TyList) pretty() string { return "" }
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
func (*TyVar) pretty() string { return "" }
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
func (*TyForall) pretty() string { return "" }
func (n *TyForall) Loc() Loc     { return n.Node.loc }
func (n *TyForall) Id() int      { return n.Node.id }

// Pattern
//
//	PWildcard
type PWildcard struct {
	Node
}

func (*PWildcard) isPat()         {}
func (*PWildcard) pretty() string { return "" }
func (n *PWildcard) Loc() Loc     { return n.Node.loc }
func (n *PWildcard) Id() int      { return n.Node.id }

// PApp
type PApp struct {
	constructor PVar
	pats        []Pat
	Node
}

func (*PApp) isPat()         {}
func (*PApp) pretty() string { return "" }
func (n *PApp) Loc() Loc     { return n.Node.loc }
func (n *PApp) Id() int      { return n.Node.id }

// PList
type PList struct {
	pats []Pat
	Node
}

func (*PList) isPat()         {}
func (*PList) pretty() string { return "" }
func (n *PList) Loc() Loc     { return n.Node.loc }
func (n *PList) Id() int      { return n.Node.id }

// PTuple
type PTuple struct {
	pats []Pat
	Node
}

func (*PTuple) isPat()         {}
func (*PTuple) pretty() string { return "" }
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
func (pv *PVar) pretty() string { return pv.name }
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
func (*PInfix) pretty() string { return "" }
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
// 	name      string
// 	canonical string
// 	module    string
// 	Node
// }

// func (*ExpCon) isExp()         {}
// func (c *ExpCon) pretty() string {
// 	if c.name == "unit" {
// 		return "()"
// 	}
// 	return c.name
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
func (*ExpLet) pretty() string { return "" }
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
func (*ExpDo) pretty() string { return "" }
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
func (*ExpTuple) pretty() string { return "" }
func (n *ExpTuple) Loc() Loc     { return n.Node.loc }
func (n *ExpTuple) Id() int      { return n.Node.id }

// ExpList
type ExpList struct {
	exps []Exp
	Node
}

func (*ExpList) isExp()         {}
func (*ExpList) pretty() string { return "" }
func (n *ExpList) Loc() Loc     { return n.Node.loc }
func (n *ExpList) Id() int      { return n.Node.id }

// ExpLeftSection
type ExpLeftSection struct {
	left Exp
	op   Exp
	Node
}

func (*ExpLeftSection) isExp()         {}
func (*ExpLeftSection) pretty() string { return "" }
func (n *ExpLeftSection) Loc() Loc     { return n.Node.loc }
func (n *ExpLeftSection) Id() int      { return n.Node.id }

// ExpRightSection
type ExpRightSection struct {
	right Exp
	op    Exp
	Node
}

func (*ExpRightSection) isExp()         {}
func (*ExpRightSection) pretty() string { return "" }
func (n *ExpRightSection) Loc() Loc     { return n.Node.loc }
func (n *ExpRightSection) Id() int      { return n.Node.id }

// ExpEnumFromTo
type ExpEnumFromTo struct {
	exp1 Exp
	exp2 Exp
	Node
}

func (*ExpEnumFromTo) isExp()         {}
func (*ExpEnumFromTo) pretty() string { return "" }
func (n *ExpEnumFromTo) Loc() Loc     { return n.Node.loc }
func (n *ExpEnumFromTo) Id() int      { return n.Node.id }

// ExpEnumFrom
type ExpEnumFrom struct {
	exp Exp
	Node
}

func (*ExpEnumFrom) isExp()         {}
func (*ExpEnumFrom) pretty() string { return "" }
func (n *ExpEnumFrom) Loc() Loc     { return n.Node.loc }
func (n *ExpEnumFrom) Id() int      { return n.Node.id }

// ExpComprehension
type ExpComprehension struct {
	exp        Exp
	generators []Generator
	guards     []Exp
	Node
}

func (*ExpComprehension) isExp()         {}
func (*ExpComprehension) pretty() string { return "" }
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
	switch l.lit {
	case "integer", "float":
		return l.content
	case "char":
		return "'" + l.content + "'"
	case "string":
		return "\"" + l.content + "\""
	default:
		panic("Unknown kind of Lit: " + l.lit)
	}
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
	return render(`{{.Exp -}} {{ if gt (len .Wheres) 0}} where {{range .Wheres}} {{ . -}}; {{end}}{{end}}`, "UnguardedRhs", struct {
		Exp    string
		Wheres []string
	}{
		Exp:    ur.exp.pretty(),
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
func (*GuardedRhs) pretty() string { return "" }
func (n *GuardedRhs) Loc() Loc     { return n.Node.loc }
func (n *GuardedRhs) Id() int      { return n.Node.id }

// GuardBranch
type GuardBranch struct {
	exp    Exp
	guards []Exp
	Node
}

func (*GuardBranch) pretty() string { return "" }
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
func (*Generator) pretty() string { return "" }
func (n *Generator) Loc() Loc     { return n.Node.loc }
func (n *Generator) Id() int      { return n.Node.id }

// Qualifier
type Qualifier struct {
	exp Exp
	Node
}

func (*Qualifier) isStatement()   {}
func (*Qualifier) pretty() string { return "" }
func (n *Qualifier) Loc() Loc     { return n.Node.loc }
func (n *Qualifier) Id() int      { return n.Node.id }

// LetStmt
type LetStmt struct {
	binds []Decl
	Node
}

func (*LetStmt) isStatement()   {}
func (*LetStmt) pretty() string { return "" }
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
func (*TypeDecl) pretty() string { return "" }
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
func (*DataDecl) pretty() string { return "" }
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
func (*ClassDecl) pretty() string { return "" }
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
func (*InstDecl) pretty() string { return "" }
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
func (*TypeSig) pretty() string { return "" }
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
	bindStrs := make([]string, len(a.binds))
	for i, b := range a.binds {
		bindStrs[i] = b.pretty()
	}
	bindStr := strings.Join(bindStrs, "; ")
	return a.pat.pretty() + " -> " + a.exp.pretty() + "where" + bindStr
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

func (*DataCon) pretty() string { return "" }
func (n *DataCon) Loc() Loc     { return n.Node.loc }
func (n *DataCon) Id() int      { return n.Node.id }

// DeclHead
type DeclHead struct {
	name      string
	canonical string
	typeVars  []TyVar
	Node
}

func (*DeclHead) pretty() string { return "" }
func (n *DeclHead) Loc() Loc     { return n.Node.loc }
func (n *DeclHead) Id() int      { return n.Node.id }

// Module
type Module struct {
	name    string
	decls   []Decl
	imports []string
	Node
}

func (m *Module) pretty() string {
	t := `module {{ .Name }} where
{{- range .Decls }}
{{ . }}
{{- end }}`

	decls := make([]string, len(m.decls))
	for i, decl := range m.decls {
		decls[i] = decl.pretty()
	}
	return render(t, "module", struct {
		Name  string
		Decls []string
	}{
		m.name, decls,
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
