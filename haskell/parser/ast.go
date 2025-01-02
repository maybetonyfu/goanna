package parser

import (
	"fmt"
	"strings"
	"bytes"
	"text/template"
)

type AST interface {
	pretty() string
	loc () Loc
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
		toLine: l1.toLine,
		fromCol: l2.fromCol,
		toCol: l2.toCol,
	}
}

var noloc = Loc {
	0, 0, 0, 0,
}

type Node struct {
	id  int
	loc Loc
}

// Types
//  TyCon
type TyCon struct {
	name      string
	module    string
	canonical string
	axiom     bool
	Node
}

func (*TyCon) isType()        {}
func (*TyCon) pretty() string { return "" }
func (n *TyCon) loc() Loc {return n.Node.loc}

//  TyApp
type TyApp struct {
	ty1   Type
	ty2   Type
	axiom bool
	Node
}

func (*TyApp) isType()        {}
func (*TyApp) pretty() string { return "" }
func (n *TyApp) loc() Loc {return n.Node.loc}

//  TyFunction
type TyFunction struct {
	ty1   Type
	ty2   Type
	axiom bool
	Node
}

func (*TyFunction) isType()        {}
func (*TyFunction) pretty() string { return "" }
func (n *TyFunction) loc() Loc {return n.Node.loc}

//  TyTuple
type TyTuple struct {
	tys   []Type
	axiom bool
	Node
}

func (*TyTuple) isType()        {}
func (*TyTuple) pretty() string { return "" }
func (n *TyTuple) loc() Loc {return n.Node.loc}

//  TyList
type TyList struct {
	ty    Type
	axiom bool
	Node
}

func (*TyList) isType()        {}
func (*TyList) pretty() string { return "" }
func (n *TyList) loc() Loc {return n.Node.loc}

//  TyPrefixList
type TyPrefixList struct {
	axiom bool
	Node
}

func (*TyPrefixList) isType()        {}
func (*TyPrefixList) pretty() string { return "" }
func (n *TyPrefixList) loc() Loc {return n.Node.loc}

//  TyPrefixTuple
type TyPrefixTuple struct {
	axiom bool
	Node
}

func (*TyPrefixTuple) isType()        {}
func (*TyPrefixTuple) pretty() string { return "" }
func (n *TyPrefixTuple) loc() Loc {return n.Node.loc}

//  TyPrefixFunction
type TyPrefixFunction struct {
	axiom bool
	Node
}

func (*TyPrefixFunction) isType()        {}
func (*TyPrefixFunction) pretty() string { return "" }
func (n *TyPrefixFunction) loc() Loc {return n.Node.loc}

//  TyVar
type TyVar struct {
	name      string
	canonical string
	axiom     bool
	Node
}

func (*TyVar) isType()        {}
func (*TyVar) pretty() string { return "" }
func (n *TyVar) loc() Loc {return n.Node.loc}

//  TyForall
type TyForall struct {
	assertions []Type
	ty         Type
	axiom      bool
	Node
}

func (*TyForall) isType()        {}
func (*TyForall) pretty() string { return "" }
func (n *TyForall) loc() Loc {return n.Node.loc}

// Pattern
//  PWildcard
type PWildcard struct {
	Node
}

func (*PWildcard) isPat()         {}
func (*PWildcard) pretty() string { return "" }
func (n *PWildcard) loc() Loc {return n.Node.loc}

//  PApp
type PApp struct {
	name      string
	module    string
	canonical string
	pats      []Pat
	Node
}

func (*PApp) isPat()         {}
func (*PApp) pretty() string { return "" }
func (n *PApp) loc() Loc {return n.Node.loc}

//  PList
type PList struct {
	pats []Pat
	Node
}

func (*PList) isPat()         {}
func (*PList) pretty() string { return "" }
func (n *PList) loc() Loc {return n.Node.loc}

//  PTuple
type PTuple struct {
	pats []Pat
	Node
}

func (*PTuple) isPat()         {}
func (*PTuple) pretty() string { return "" }
func (n *PTuple) loc() Loc {return n.Node.loc}

//  PVar
type PVar struct {
	name      string
	canonical string
	Node
}

func (*PVar) isPat()         {}
func (pv *PVar) pretty() string { return pv.name }
func (n *PVar) loc() Loc {return n.Node.loc}

//  PInfix
type PInfix struct {
	pat1      Pat
	name      string
	module    string
	canonical string
	pat2      Pat
	Node
}

func (*PInfix) isPat()         {}
func (*PInfix) pretty() string { return "" }
func (n *PInfix) loc() Loc {return n.Node.loc}

// ExpVar
type ExpVar struct {
	name      string
	module    string
	canonical string
	Node
}

func (*ExpVar) isExp()         {}
func (v *ExpVar) pretty() string {
	if v.name == "unit" {
		return "()"
	}
	return v.name
}
func (n *ExpVar) loc() Loc {return n.Node.loc}

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
// func (n *ExpCon) loc() Loc {return n.Node.loc}


// ExpApp
type ExpApp struct {
	exp1 Exp
	exp2 Exp
	Node
}

func (*ExpApp) isExp()         {}
func (e *ExpApp) pretty() string {
	return "(" + e.exp1.pretty() + " " + e.exp2.pretty() + ")"
}
func (n *ExpApp) loc() Loc {return n.Node.loc}

//  ExpInfix
type ExpInfix struct {
	exp1 Exp
	exp2 Exp
	op   ExpVar
	Node
}

func (*ExpInfix) isExp()         {}
func (ei *ExpInfix) pretty() string {
	return fmt.Sprintf("(%s %s %s)", ei.exp1.pretty(), ei.op.pretty(), ei.exp2.pretty())
}
func (n *ExpInfix) loc() Loc {return n.Node.loc}

//  ExpLambda
type ExpLambda struct {
	pats []Pat
	exp  Exp
	Node
}

func (*ExpLambda) isExp()         {}
func (e *ExpLambda) pretty() string {
	pats := make([]string, len(e.pats))
	for i, pat := range e.pats {
		pats[i] = pat.pretty()
	}

	return "(\\"+ strings.Join(pats, " ") + " -> " + e.exp.pretty() + ")"
}
func (n *ExpLambda) loc() Loc {return n.Node.loc}

//  ExpLet
type ExpLet struct {
	binds []Decl
	exp   Exp
	Node
}

func (*ExpLet) isExp()         {}
func (*ExpLet) pretty() string { return "" }
func (n *ExpLet) loc() Loc {return n.Node.loc}

//  ExpIf
type ExpIf struct {
	cond    Exp
	ifTrue  Exp
	ifFalse Exp
	Node
}

func (*ExpIf) isExp()         {}
func (e *ExpIf) pretty() string {
	return "if " + e.cond.pretty() +
		" then " + e.ifTrue.pretty() +
		" else " + e.ifFalse.pretty()
}
func (n *ExpIf) loc() Loc {return n.Node.loc}

//  ExpDo
type ExpDo struct {
	stmts []Statement
	Node
}

func (*ExpDo) isExp()         {}
func (*ExpDo) pretty() string { return "" }
func (n *ExpDo) loc() Loc {return n.Node.loc}

//  ExpCase
type ExpCase struct {
	exp  Exp
	alts []Alt
	Node
}

func (*ExpCase) isExp()         {}
func (*ExpCase) pretty() string { return "" }
func (n *ExpCase) loc() Loc {return n.Node.loc}

//  ExpTuple
type ExpTuple struct {
	exps []Exp
	Node
}

func (*ExpTuple) isExp()         {}
func (*ExpTuple) pretty() string { return "" }
func (n *ExpTuple) loc() Loc {return n.Node.loc}

//  ExpList
type ExpList struct {
	exps []Exp
	Node
}

func (*ExpList) isExp()         {}
func (*ExpList) pretty() string { return "" }
func (n *ExpList) loc() Loc {return n.Node.loc}

//  ExpLeftSection
type ExpLeftSection struct {
	left Exp
	op   Exp
	Node
}

func (*ExpLeftSection) isExp()         {}
func (*ExpLeftSection) pretty() string { return "" }
func (n *ExpLeftSection) loc() Loc {return n.Node.loc}

//  ExpRightSection
type ExpRightSection struct {
	right Exp
	op    Exp
	Node
}

func (*ExpRightSection) isExp()         {}
func (*ExpRightSection) pretty() string { return "" }
func (n *ExpRightSection) loc() Loc {return n.Node.loc}
//  ExpEnumFromTo
type ExpEnumFromTo struct {
	exp1 Exp
	exp2 Exp
	Node
}

func (*ExpEnumFromTo) isExp()         {}
func (*ExpEnumFromTo) pretty() string { return "" }
func (n *ExpEnumFromTo) loc() Loc {return n.Node.loc}

//  ExpEnumFrom
type ExpEnumFrom struct {
	exp Exp
	Node
}

func (*ExpEnumFrom) isExp()         {}
func (*ExpEnumFrom) pretty() string { return "" }
func (n *ExpEnumFrom) loc() Loc {return n.Node.loc}

//  ExpComprehension
type ExpComprehension struct {
	exp         Exp
	quantifiers []Generator
	guards      []Exp
	Node
}

func (*ExpComprehension) isExp()         {}
func (*ExpComprehension) pretty() string { return "" }
func (n *ExpComprehension) loc() Loc {return n.Node.loc}


//  Lit
type Lit struct {
	lit string // integer/char/string/float
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
func (n *Lit) loc() Loc {return n.Node.loc}

// RHS
//  UnguardedRhs
type UnguardedRhs struct {
	exp    Exp
	wheres []Decl
	Node
}

func (*UnguardedRhs) isRhs()         {}
func (ur *UnguardedRhs) pretty() string {
	wheres := make([]string, len(ur.wheres))
	for i, where := range ur.wheres {
		wheres[i] = where.pretty()
	}
	return render(`{{.Exp -}} {{ if gt (len .Wheres) 0}} where {{range .Wheres}} {{ . -}}; {{end}}{{end}}`, "UnguardedRhs", struct {
		Exp string
		Wheres []string
	}{
		Exp: ur.exp.pretty(),
		Wheres: wheres,
	})
}
func (n *UnguardedRhs) loc() Loc {return n.Node.loc}

//  GuardedRhs
type GuardedRhs struct {
  branches []GuardBranch
	wheres []Decl
	Node
}

func (*GuardedRhs) isRhs()         {}
func (*GuardedRhs) pretty() string { return "" }
func (n *GuardedRhs) loc() Loc {return n.Node.loc}

//  GuardBranch
type GuardBranch struct {
	exp Exp
	guards []Exp
	Node
}
func (*GuardBranch) pretty() string { return "" }
func (n *GuardBranch) loc() Loc {return n.Node.loc}

// Statements

//  Generator
type Generator struct {
	pat Pat
	exp Exp
	Node
}

func (*Generator) isStatement()   {}
func (*Generator) pretty() string { return "" }
func (n *Generator) loc() Loc {return n.Node.loc}

//  Qualifier
type Qualifier struct {
	exp Exp
	Node
}

func (*Qualifier) isStatement()   {}
func (*Qualifier) pretty() string { return "" }
func (n *Qualifier) loc() Loc {return n.Node.loc}

//  LetStmt
type LetStmt struct {
	binds []Decl
	Node
}

func (*LetStmt) isStatement()   {}
func (*LetStmt) pretty() string { return "" }
func (n *LetStmt) loc() Loc {return n.Node.loc}

// Declarations

//  TypeDecl
type TypeDecl struct {
	dHead DeclHead
	ty    Type
	Node
}

func (*TypeDecl) isDecl()        {}
func (*TypeDecl) pretty() string { return "" }
func (n *TypeDecl) loc() Loc {return n.Node.loc}

//  DataDecl
type DataDecl struct {
	dHead        DeclHead
	constructors []DataCon
	deriving     []TyCon
	Node
}

func (*DataDecl) isDecl()        {}
func (*DataDecl) pretty() string { return "" }
func (n *DataDecl) loc() Loc {return n.Node.loc}

//  ClassDecl
type ClassDecl struct {
	assertions []Type
	dHead      DeclHead
	decls      []Decl
	Node
}

func (*ClassDecl) isDecl()        {}
func (*ClassDecl) pretty() string { return "" }
func (n *ClassDecl) loc() Loc {return n.Node.loc}

//  InstDecl
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
func (n *InstDecl) loc() Loc {return n.Node.loc}

//  PatBind
type PatBind struct {
	pat Pat
	rhs Rhs
	Node
}

func (*PatBind) isDecl()        {}
func (pb *PatBind) pretty() string {
	return pb.pat.pretty() + " = " + pb.rhs.pretty()
}
func (n *PatBind) loc() Loc {return n.Node.loc}

//  TypeSig
type TypeSig struct {
	names      []string
	canonicals []string
	ty         Type
	Node
}

func (*TypeSig) isDecl()        {}
func (*TypeSig) pretty() string { return "" }
func (n *TypeSig) loc() Loc {return n.Node.loc}

// Misc

//  Alt
type Alt struct {
	pat   Pat
	exp   Exp
	binds []Decl
	Node
}

func (*Alt) pretty() string { return "" }
func (n *Alt) loc() Loc {return n.Node.loc}

//  DataCon
type DataCon struct {
	name      string
	canonical string
	tys       []Type
	Node
}

func (*DataCon) pretty() string { return "" }
func (n *DataCon) loc() Loc {return n.Node.loc}

//  DeclHead
type DeclHead struct {
	name      string
	canonical string
	typeVars  []TyVar
	Node
}

func (*DeclHead) pretty() string { return "" }
func (n *DeclHead) loc() Loc {return n.Node.loc}

//  Module
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
		Name string
		Decls []string
	}{
		m.name, decls,
	})
}
func (n *Module) loc() Loc {return n.Node.loc}

func render(temp string, name string, data any) string {
	var buf bytes.Buffer

	t := template.Must(template.New(name).Parse(temp))
  if err := t.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}
