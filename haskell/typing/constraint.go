package typing

import (
	"fmt"
	prolog "goanna/prolog-tool"
	"goanna/haskell/parser"
)

// ConstraintGenState holds per-traversal state, wrapping the global TypingEnv.
// Mirrors constraint.py's ConstraintGenState.
type ConstraintGenState struct {
	freshCounter int
	module       string
	global       *TypingEnv
}

func NewConstraintGenState(global *TypingEnv) *ConstraintGenState {
	return &ConstraintGenState{global: global}
}

func (s *ConstraintGenState) SetModuleName(module string) {
	s.module = module
}

func (s *ConstraintGenState) declarations() []string {
	return s.global.Declarations
}

func (s *ConstraintGenState) addRule(body prolog.LTerm, head RuleHead, nodeID int) {
	s.global.AddRule(&Rule{Head: head, Body: body, Axiom: false, NodeID: &nodeID})
}

func (s *ConstraintGenState) addRules(bodies []prolog.LTerm, head RuleHead, nodeID int) {
	for _, body := range bodies {
		s.addRule(body, head, nodeID)
	}
}

func (s *ConstraintGenState) addAxiom(body prolog.LTerm, head RuleHead) {
	s.global.AddRule(&Rule{Head: head, Body: body, Axiom: true, NodeID: nil})
}

func (s *ConstraintGenState) fresh() prolog.LVar {
	s.freshCounter++
	return prolog.LVar{Value: fmt.Sprintf("_f%d", s.freshCounter)}
}

func (s *ConstraintGenState) headOfTypingRule(name string) RuleHead {
	return RuleHead{Kind: RuleKindType, Name: name, Module: s.module}
}

func (s *ConstraintGenState) headOfInstanceRule(name string, instanceID int) RuleHead {
	return RuleHead{Kind: RuleKindInstance, Name: name, Module: s.module, ID: &instanceID}
}

// typeOf mirrors ConstraintGenState.type_of: returns the rule(s) that look up
// the type of a named declaration, handling the parent-scope case.
func (s *ConstraintGenState) typeOf(name string, v prolog.LVar, head RuleHead) []prolog.LTerm {
	collector := s.fresh()
	s.global.AddClassVar(head.Name, collector.Value)
	if s.global.IsParentOf(head.Name, name) {
		w := s.fresh()
		rule1 := prolog.LStruct{Functor: name, Args: []prolog.LTerm{
			v, prolog.Call_, prolog.Wildcard, w, prolog.Wildcard, collector,
		}}
		rule2 := prolog.Once(prolog.LStruct{Functor: "append", Args: []prolog.LTerm{
			prolog.ZetaVar, prolog.Wildcard, w,
		}})
		return []prolog.LTerm{rule1, rule2}
	}
	rule := prolog.LStruct{Functor: name, Args: []prolog.LTerm{
		v, prolog.Call_, prolog.Wildcard, prolog.Wildcard, prolog.Wildcard, collector,
	}}
	return []prolog.LTerm{rule}
}

// ---------------------------------------------------------------------------
// generateType: translates a Type AST node into a Prolog term.
// Mirrors constraint.py's generate_type.
// ---------------------------------------------------------------------------

func (s *ConstraintGenState) generateType(ty parser.Type) prolog.LTerm {
	if ty == nil {
		return s.fresh()
	}
	switch t := ty.(type) {
	case *parser.TyCon:
		name := t.Canonical
		if name == "" {
			name = t.Name
		}
		return prolog.LAtom{Value: name}

	case *parser.TyVar:
		name := t.Canonical
		if name == "" {
			name = t.Name
		}
		return prolog.LVar{Value: name}

	case *parser.TyApp:
		f := s.generateType(t.Ty1)
		arg := s.generateType(t.Ty2)
		return prolog.LStruct{Functor: "tyapp", Args: []prolog.LTerm{f, arg}}

	case *parser.TyFunction:
		from := s.generateType(t.Ty1)
		to := s.generateType(t.Ty2)
		return prolog.LStruct{Functor: "->", Args: []prolog.LTerm{from, to}}

	case *parser.TyTuple:
		parts := make([]prolog.LTerm, len(t.Tys))
		for i, ty := range t.Tys {
			parts[i] = s.generateType(ty)
		}
		return prolog.LStruct{Functor: "tuple", Args: parts}

	case *parser.TyList:
		elem := s.generateType(t.Ty)
		return prolog.LStruct{Functor: "list", Args: []prolog.LTerm{elem}}

	case *parser.TyForall:
		// Strip the forall/context wrapper; constraints are handled separately.
		return s.generateType(t.Ty)

	default:
		return s.fresh()
	}
}

// ---------------------------------------------------------------------------
// generateConstraint: translates an expression into Prolog constraint terms.
// Mirrors constraint.py's generate_constraint.
// ---------------------------------------------------------------------------

func (s *ConstraintGenState) generateConstraint(exp parser.Exp, v prolog.LTerm, head RuleHead) []prolog.LTerm {
	if exp == nil {
		return nil
	}
	switch e := exp.(type) {

	case *parser.ExpVar:
		name := e.Canonical
		if name == "" {
			name = e.Name
		}
		// check if this decl is in scope
		for _, d := range s.declarations() {
			if d == name {
				return s.typeOf(name, v.(prolog.LVar), head)
			}
		}
		// Axiom / unknown — unify with a fresh var
		w := s.fresh()
		return []prolog.LTerm{prolog.LStruct{Functor: "=", Args: []prolog.LTerm{v, w}}}

	case *parser.ExpApp:
		argTy := s.fresh()
		funTy := prolog.LStruct{Functor: "->", Args: []prolog.LTerm{argTy, v}}
		funW := s.fresh()
		c1 := s.generateConstraint(e.Exp1, funW, head)
		c2 := s.generateConstraint(e.Exp2, argTy, head)
		unify := prolog.LStruct{Functor: "=", Args: []prolog.LTerm{funW, funTy}}
		return append(append(c1, c2...), unify)

	case *parser.ExpInfix:
		argTy1 := s.fresh()
		argTy2 := s.fresh()
		funTy := prolog.LStruct{Functor: "->", Args: []prolog.LTerm{
			argTy1, prolog.LStruct{Functor: "->", Args: []prolog.LTerm{argTy2, v}},
		}}
		opW := s.fresh()
		c1 := s.generateConstraint(e.Exp1, argTy1, head)
		c2 := s.generateConstraint(e.Exp2, argTy2, head)
		cop := s.generateConstraint(&e.Op, opW, head)
		unify := prolog.LStruct{Functor: "=", Args: []prolog.LTerm{opW, funTy}}
		return append(append(append(c1, c2...), cop...), unify)

	case *parser.ExpLambda:
		// Build a chain: T1 -> T2 -> ... -> Tbody
		body := s.fresh()
		var chain prolog.LTerm = body
		paramTypes := make([]prolog.LVar, len(e.Pats))
		for i := len(e.Pats) - 1; i >= 0; i-- {
			pt := s.fresh()
			paramTypes[i] = pt
			chain = prolog.LStruct{Functor: "->", Args: []prolog.LTerm{pt, chain}}
		}
		unify := prolog.LStruct{Functor: "=", Args: []prolog.LTerm{v, chain}}
		bodyC := s.generateConstraint(e.Exp, body, head)
		return append([]prolog.LTerm{unify}, bodyC...)

	case *parser.ExpLet:
		var cs []prolog.LTerm
		for _, bind := range e.Binds {
			pb, ok := bind.(*parser.PatBind)
			if !ok {
				continue
			}
			w := s.fresh()
			cs = append(cs, s.generateConstraintPatBind(pb, w, head)...)
		}
		cs = append(cs, s.generateConstraint(e.Exp, v, head)...)
		return cs

	case *parser.ExpIf:
		boolTy := prolog.LAtom{Value: "bool"}
		condW := s.fresh()
		c1 := s.generateConstraint(e.Cond, condW, head)
		unify := prolog.LStruct{Functor: "=", Args: []prolog.LTerm{condW, boolTy}}
		c2 := s.generateConstraint(e.IfTrue, v, head)
		c3 := s.generateConstraint(e.IfFalse, v, head)
		return append(append(append(c1, unify), c2...), c3...)

	case *parser.ExpCase:
		scrutW := s.fresh()
		cs := s.generateConstraint(e.Exp, scrutW, head)
		for _, alt := range e.Alts {
			altW := s.fresh()
			cs = append(cs, s.generateConstraint(alt.Exp, altW, head)...)
			cs = append(cs, prolog.LStruct{Functor: "=", Args: []prolog.LTerm{altW, v}})
		}
		return cs

	case *parser.ExpTuple:
		parts := make([]prolog.LTerm, len(e.Exps))
		var cs []prolog.LTerm
		for i, ex := range e.Exps {
			w := s.fresh()
			parts[i] = w
			cs = append(cs, s.generateConstraint(ex, w, head)...)
		}
		tupleTy := prolog.LStruct{Functor: "tuple", Args: parts}
		cs = append(cs, prolog.LStruct{Functor: "=", Args: []prolog.LTerm{v, tupleTy}})
		return cs

	case *parser.ExpList:
		elemTy := s.fresh()
		listTy := prolog.LStruct{Functor: "list", Args: []prolog.LTerm{elemTy}}
		cs := []prolog.LTerm{prolog.LStruct{Functor: "=", Args: []prolog.LTerm{v, listTy}}}
		for _, ex := range e.Exps {
			w := s.fresh()
			cs = append(cs, s.generateConstraint(ex, w, head)...)
			cs = append(cs, prolog.LStruct{Functor: "=", Args: []prolog.LTerm{w, elemTy}})
		}
		return cs

	case *parser.ExpDo:
		var cs []prolog.LTerm
		for i, stmt := range e.Stmts {
			isLast := i == len(e.Stmts)-1
			switch st := stmt.(type) {
			case *parser.Generator:
				w := s.fresh()
				cs = append(cs, s.generateConstraint(st.Exp, w, head)...)
			case *parser.Qualifier:
				w := s.fresh()
				if isLast {
					cs = append(cs, s.generateConstraint(st.Exp, v, head)...)
				} else {
					cs = append(cs, s.generateConstraint(st.Exp, w, head)...)
				}
			case *parser.LetStmt:
				for _, bind := range st.Binds {
					pb, ok := bind.(*parser.PatBind)
					if !ok {
						continue
					}
					w := s.fresh()
					cs = append(cs, s.generateConstraintPatBind(pb, w, head)...)
				}
			}
		}
		return cs

	case *parser.ExpComprehension:
		elemTy := s.fresh()
		listTy := prolog.LStruct{Functor: "list", Args: []prolog.LTerm{elemTy}}
		cs := []prolog.LTerm{prolog.LStruct{Functor: "=", Args: []prolog.LTerm{v, listTy}}}
		cs = append(cs, s.generateConstraint(e.Exp, elemTy, head)...)
		for _, gen := range e.Generators {
			w := s.fresh()
			cs = append(cs, s.generateConstraint(gen.Exp, w, head)...)
		}
		for _, guard := range e.Guards {
			w := s.fresh()
			cs = append(cs, s.generateConstraint(guard, w, head)...)
		}
		return cs

	case *parser.ExpLeftSection:
		argTy := s.fresh()
		opW := s.fresh()
		funTy := prolog.LStruct{Functor: "->", Args: []prolog.LTerm{argTy, v}}
		c1 := s.generateConstraint(e.Left, argTy, head)
		c2 := s.generateConstraint(e.Op, opW, head)
		unify := prolog.LStruct{Functor: "=", Args: []prolog.LTerm{opW, funTy}}
		return append(append(c1, c2...), unify)

	case *parser.ExpRightSection:
		argTy := s.fresh()
		opW := s.fresh()
		funTy := prolog.LStruct{Functor: "->", Args: []prolog.LTerm{argTy, v}}
		c1 := s.generateConstraint(e.Right, argTy, head)
		c2 := s.generateConstraint(e.Op, opW, head)
		unify := prolog.LStruct{Functor: "=", Args: []prolog.LTerm{opW, funTy}}
		return append(append(c1, c2...), unify)

	case *parser.ExpEnumFromTo:
		w1, w2 := s.fresh(), s.fresh()
		c1 := s.generateConstraint(e.Exp1, w1, head)
		c2 := s.generateConstraint(e.Exp2, w2, head)
		return append(c1, c2...)

	case *parser.ExpEnumFrom:
		w := s.fresh()
		return s.generateConstraint(e.Exp, w, head)

	case *parser.Lit:
		var litTy prolog.LTerm
		switch e.Lit {
		case "integer":
			litTy = prolog.LAtom{Value: "int"}
		case "string":
			litTy = prolog.LAtom{Value: "string"}
		case "char":
			litTy = prolog.LAtom{Value: "char"}
		case "float":
			litTy = prolog.LAtom{Value: "float"}
		default:
			litTy = s.fresh()
		}
		return []prolog.LTerm{prolog.LStruct{Functor: "=", Args: []prolog.LTerm{v, litTy}}}

	default:
		return nil
	}
}

// generateConstraintPatBind generates constraints for a PatBind binding.
func (s *ConstraintGenState) generateConstraintPatBind(pb *parser.PatBind, v prolog.LTerm, head RuleHead) []prolog.LTerm {
	var rhsExp parser.Exp
	switch r := pb.Rhs.(type) {
	case *parser.UnguardedRhs:
		rhsExp = r.Exp
	case *parser.GuardedRhs:
		// Use first branch for simplicity
		if len(r.Branches) > 0 {
			rhsExp = r.Branches[0].Exp
		}
	}
	return s.generateConstraint(rhsExp, v, head)
}

// ---------------------------------------------------------------------------
// getAllConstraints: the top-level entry point.
// Mirrors constraint.py's get_all_constraints / generate_constraint on Module.
// ---------------------------------------------------------------------------

func (s *ConstraintGenState) GetAllConstraints(modules []*parser.Module) {
	for _, m := range modules {
		s.module = m.Name
		for _, decl := range m.Decls {
			s.generateDeclConstraints(decl)
		}
	}
}

func (s *ConstraintGenState) generateDeclConstraints(decl parser.Decl) {
	switch d := decl.(type) {
	case *parser.PatBind:
		name := ""
		switch p := d.Pat.(type) {
		case *parser.PVar:
			name = p.Canonical
			if name == "" {
				name = p.Name
			}
		case *parser.PApp:
			name = p.Constructor.Canonical
			if name == "" {
				name = p.Constructor.Name
			}
		}
		if name == "" {
			return
		}
		head := s.headOfTypingRule(name)
		v := prolog.LVar{Value: "T"}
		cs := s.generateConstraintPatBind(d, v, head)
		s.addRules(cs, head, d.Id())

		// Recurse into where-clauses
		var wheres []parser.Decl
		switch r := d.Rhs.(type) {
		case *parser.UnguardedRhs:
			wheres = r.Wheres
		case *parser.GuardedRhs:
			wheres = r.Wheres
		}
		for _, w := range wheres {
			s.generateDeclConstraints(w)
		}

	case *parser.InstDecl:
		instName := d.Canonical
		if instName == "" {
			instName = d.Name
		}
		head := s.headOfInstanceRule(instName, d.Id())
		for _, inner := range d.Body {
			pb, ok := inner.(*parser.PatBind)
			if !ok {
				continue
			}
			v := s.fresh()
			cs := s.generateConstraintPatBind(pb, v, head)
			s.addRules(cs, head, d.Id())
		}

	case *parser.ClassDecl:
		for _, inner := range d.Decls {
			s.generateDeclConstraints(inner)
		}
	}
}
