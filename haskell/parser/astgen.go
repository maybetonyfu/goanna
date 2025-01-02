package parser

import (
	treesitter "github.com/tree-sitter/go-tree-sitter"
	"slices"
)

func (pe parseEnv) parseDecl(node *treesitter.Node) Decl {
	if node.IsMissing() {
		panic("Missing declaration")
	}
	if node.IsError() {
		panic("Error")
	}
	if node.HasError() {
		panic("Has Error")
	}

	switch node.Kind() {
	case "signature":
	// 	var nameNodes []treesitter.Node
	// 	if v := pe.child(node, "names"); v == nil {
	// 		nameNodes = pe.children(node, "names:name")
	// 	} else {
	// 		nameNodes = append(nameNodes, *pe.child(node, "name"))
	// 	}
	// 	for _, n := range nameNodes {
	// 		fmt.Println(n)
	// 	}
	// 	fmt.Println("signature")
	case "data_type":
	case "class":
	case "instance":
	case "function", "bind":
		variableNode := node.NamedChild(0)
		pat := pe.parsePat(variableNode)
		rhs := pe.parseRhs(node)
		return Decl(&PatBind{
			pat:  pat,
			rhs:  rhs,
			Node: pe.node(node),
		})
	case "fixity":
	case "type_synomym":
	default:
		panic("Unknown declaration type: " + node.Kind())
	}
	return nil
}

func (pe parseEnv) parsePat(node *treesitter.Node) Pat {
	switch node.Kind() {
	case "qualified":
		module := pe.text(pe.child(node, "module"))
		name := pe.text(pe.child(node, "id"))
		return Pat(&PApp{
			name:      name,
			module:    module,
			canonical: "",
			pats:      []Pat{},
			Node:      pe.node(node),
		})
	case "prefix_id":
		operator := node.NamedChild(0)
		name := pe.text(operator)
		return Pat(&PVar{
			name:      name,
			canonical: "",
			Node:      pe.node(node),
		})
	case "variable":
		name := pe.text(node)
		return Pat(&PVar{
			name:      name,
			canonical: "",
			Node:      pe.node(node),
		})
	case "constructor":
		name := pe.text(node)
		return Pat(&PApp{
			name:      name,
			canonical: "",
			pats:      []Pat{},
			Node:      pe.node(node),
		})
	case "literal":
		return Pat(pe.parseLit(node))

	case "parens":
		return pe.parsePat(node.ChildByFieldName("pattern"))

	case "wildcard":
		return Pat(&PWildcard{
			Node: pe.node(node),
		})
	case "tuple":
		elems := pe.children(node, "element")
		pats := make([]Pat, len(elems))
		for i, elem := range elems {
			pats[i] = pe.parsePat(&elem)
		}
		return Pat(&PTuple{
			pats: pats,
			Node: pe.node(node),
		})
	case "list":
		elems := pe.children(node, "element")
		pats := make([]Pat, len(elems))
		for i, elem := range elems {
			pats[i] = pe.parsePat(&elem)
		}
		return Pat(&PList{
			pats: pats,
			Node: pe.node(node),
		})
	case "apply":
		pats := []Pat{}
		currentNode := node
		var name string
		for {
			if currentNode.Kind() == "apply" {
				h := pe.parsePat(currentNode.Child(1))
				pats = append([]Pat{h}, pats...)
				currentNode = currentNode.Child(0)
			} else {
				name = pe.text(currentNode)
				break
			}
		}
		return Pat(&PApp{
			name:      name,
			canonical: "",
			pats:      pats,
			Node:      pe.node(node),
		})
	case "infix":
		panic("Infix not implemented")
	default:
		panic("Unknown pat type: " + node.Kind())

	}
}

func (pe parseEnv) parseLit(node *treesitter.Node) *Lit {
	return &Lit{
		lit:     node.Child(0).Kind(),
		content: pe.text(node),
		Node:    pe.node(node),
	}
}

func (pe parseEnv) parseRhs(node *treesitter.Node) Rhs {
	rhsNodes := pe.children(node, "match")
	whereNodes := pe.children(node, "binds:decl")
	wheres := make([]Decl, len(whereNodes))

	for i, where := range whereNodes {
		wheres[i] = pe.parseDecl(&where)
	}

	isPatBinding := node.ChildByFieldName("patterns") == nil
	isUnguarded := rhsNodes[0].ChildByFieldName("guards") == nil
	branches := make([]GuardBranch, 0)

	for _, rhsNode := range rhsNodes {
		rhs := &rhsNode
		var rhsExp Exp
		if isPatBinding {
			rhsExp = pe.parseExp(pe.child(rhs, "expression"))
		} else {
			rhsExp = nil
		}
		if isUnguarded {
			return Rhs(&UnguardedRhs{
				exp:    rhsExp,
				wheres: wheres,
				Node:   pe.node(rhs),
			})
		} else {
			guardNodes := pe.children(rhs, "guards:guard")
			guards := make([]Exp, len(guardNodes))
			for i, guardNode := range guardNodes {
				guards[i] = pe.parseExp(&guardNode)
			}
			branches = append(branches, GuardBranch{
				exp:    rhsExp,
				guards: guards,
				Node:   pe.node(rhs),
			})
		}
	}
	return Rhs(&GuardedRhs{
		wheres:   wheres,
		branches: branches,
		Node:     pe.node(node),
	})
}

func (pe parseEnv) parseExp(node *treesitter.Node) Exp {
	switch node.Kind() {
	case "qualified":
		module := pe.text(pe.child(node, "module"))
		id := pe.child(node, "id")
		return Exp(&ExpVar{
			name:      pe.text(id),
			module:    module,
			canonical: "",
			Node:      pe.node(node),
		})

	case "literal":
		return Exp(pe.parseLit(node))

	case "variable", "constructor", "operator", "constructor_operator":
		return Exp(&ExpVar{
			name:      pe.text(node),
			module:    "",
			canonical: "",
			Node:      pe.node(node),
		})

	case "parens":
		return pe.parseExp(pe.child(node, "expression"))

	case "unit":
		return Exp(&ExpVar{
			name:      "unit",
			canonical: "",
			module:    "",
			Node:      pe.node(node),
		})

	case "prefix_id", "infix_id":
		return pe.parseExp(node.NamedChild(0))

	case "apply":
		return Exp(&ExpApp{
			exp1: pe.parseExp(node.Child(0)),
			exp2: pe.parseExp(node.Child(1)),
			Node: pe.node(node),
		})
	case "left_section":
		left := pe.parseExp(pe.child(node, "left_operand"))
		op := pe.parseExp(pe.child(node, "operator"))
		return Exp(&ExpLeftSection{
			left: left,
			op:   op,
			Node: pe.node(node),
		})
	case "right_section":
		right := pe.parseExp(pe.child(node, "right_operand"))
		op := pe.parseExp(pe.child(node, "operator"))
		return Exp(&ExpRightSection{
			right: right,
			op:    op,
			Node:  pe.node(node),
		})
	case "lambda":
		patNodes := pe.children(node, "patterns:*")
		pats := make([]Pat, len(patNodes))
		for i, patNode := range patNodes {
			pats[i] = pe.parsePat(&patNode)
		}
		exp := pe.parseExp(pe.child(node, "expression"))
		return Exp(&ExpLambda{
			pats: pats,
			exp:  exp,
			Node: pe.node(node),
		})
	case "let_in":
		bindNodes := pe.children(node, "binds:*")
		binds := make([]Decl, len(bindNodes))
		for i, bind := range bindNodes {
			binds[i] = pe.parseDecl(&bind)
		}
		exp := pe.parseExp(pe.child(node, "expression"))
		return Exp(&ExpLet{
			exp:   exp,
			binds: binds,
			Node:  pe.node(node),
		})

	case "conditional":
		condExp := pe.parseExp(pe.child(node, "if"))
		thenExp := pe.parseExp(pe.child(node, "then"))
		elseExp := pe.parseExp(pe.child(node, "else"))
		return Exp(&ExpIf{
			cond:    condExp,
			ifTrue:  thenExp,
			ifFalse: elseExp,
			Node:    pe.node(node),
		})

	case "case":
		exp := pe.parseExp(node.Child(1))
		altNodes := pe.children(node, "alternatives:*")
		alts := make([]Alt, len(altNodes))
		for i, alt := range altNodes {
			alts[i] = pe.parseAlt(&alt)
		}
		return Exp(&ExpCase{
			exp:  exp,
			alts: alts,
			Node: pe.node(node),
		})

	case "tuple":
		elems := pe.children(node, "element")
		exps := make([]Exp, len(elems))
		for i, elem := range elems {
			exps[i] = pe.parseExp(&elem)
		}
		return Exp(&ExpTuple{
			exps: exps,
			Node: pe.node(node),
		})

	case "list":
		elems := pe.children(node, "element")
		exps := make([]Pat, len(elems))
		for i, elem := range elems {
			exps[i] = pe.parseExp(&elem)
		}
		return Exp(&ExpList{
			exps: exps,
			Node: pe.node(node),
		})
  case "arithmetic_sequence":
		start := pe.child(node, "from")
		end := pe.child(node, "to")
		if end == nil {
			exp := pe.parseExp(&start)
			return Exp(&ExpEnumFrom{
				exp: exp,
				Node: pe.node(node)
			})
		} else {
			exp1 := pe.parseExp(&start)
			exp2 := pe.parseExp(&end)
			return Exp(&ExpEnumFromTo{
				exp1: exp1,
				exp2: exp2,
				Node: pe.node(node)
			})
		}

	case "infix":
		exps, ops := pe.flattenInfix(node)
		exps, _ = pe.buildInfix(exps, ops)
		return exps[0]
	}
	return nil
}

func (pe parseEnv) flattenInfix(node *treesitter.Node) ([]Exp, []ExpVar) {
	operatorNode := pe.child(node, "operator")
	lhs := pe.parseExp(pe.child(node, "left_operand"))
	rhsNode := pe.child(node, "right_operand")
	operator, ok := pe.parseExp(operatorNode).(*ExpVar)
	if !ok {
		panic("Operator is not an ExpVar node")
	}
	if rhsNode.Kind() == "infix" {
		exps, ops := pe.flattenInfix(rhsNode)
		return append([]Exp{lhs}, exps...), append([]ExpVar{*operator}, ops...)

	} else {
		rhs := pe.parseExp(rhsNode)
		return []Exp{lhs, rhs}, []ExpVar{*operator}
	}
}

func (pe parseEnv) buildInfix(exps []Exp, ops []ExpVar) ([]Exp, []ExpVar) {
	if len(ops) == 0 {
		return exps, ops
	}

	highestIndex := 0
	for i, op := range ops {

		if i == 0 {
			// The highest index is already set to 0
			continue
		}

		prev := ops[i-1]
		if op.name == prev.name && highestIndex == i-1 && pe.assoc(op.name) == "r" {
			// two same operators, use the first unless it's right associative
			highestIndex = i
			continue
		}

		if pe.fix(op.name) > pe.fix(ops[i-1].name) {
			highestIndex = i
			continue
		}
	}

	left := exps[highestIndex]
	right := exps[highestIndex+1]
	op := ops[highestIndex]

	exp := Exp(&ExpInfix{
		exp1: left,
		exp2: right,
		op:   op,
		Node: Node{
			id:  pe.id(),
			loc: mergeLoc(left.loc(), right.loc()),
		},
	})

	ops = slices.Concat(ops[:highestIndex], ops[highestIndex+1:])
	exps = slices.Concat(exps[:highestIndex], []Exp{exp}, exps[highestIndex+2:])
	return pe.buildInfix(exps, ops)
}

func (pe parseEnv) parseAlt(node *treesitter.Node) Alt {
	pat := pe.parsePat(pe.child(node, "pattern"))
	exp := pe.parseExp(pe.child(node, "match:expression"))
	bindNodes := pe.children(node, "binds:decl")
	binds := make([]Decl, len(bindNodes))
	for i, bind := range bindNodes {
		binds[i] = pe.parseDecl(&bind)
	}
	return Alt{
		pat:   pat,
		exp:   exp,
		binds: binds,
		Node:  pe.node(node),
	}
}
