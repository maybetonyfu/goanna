package parser

import (
	treesitter "github.com/tree-sitter/go-tree-sitter"
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
	// case "signature":
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
	// case "data_type":
	// 	fmt.Println("data_type")
	// case "class":
	// 	fmt.Println("class")
	// case "instance":
	// 	fmt.Println("instance")
	case "function", "bind":
		variableNode := node.NamedChild(0)
		pat := pe.parsePat(variableNode)
		rhs := pe.parseRhs(node)
		return Decl(&PatBind{
			pat:  pat,
			rhs:  rhs,
			Node: pe.node(node),
		})
	// case "fixity":
	// 	fmt.Println("fixity")
	// case "type_synomym":
	// 	fmt.Println("type_synonym")
	default:
		panic("Unknown declaration type: " + node.Kind())
	}
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
		return Pat(pe.parseLit(node.NamedChild(0)))

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
	case "literal":
		return Exp(pe.parseLit(node))
	case "variable":
		return Exp(&ExpVar{
			name: pe.text(node),
			module: "",
			canonical: "",
			Node: pe.node(node),
		})
	case "parens":
		return pe.parseExp(pe.child(node, "expression"))
	case "unit":
		return Exp(&ExpCon{
			name: "unit",
			canonical: "",
			module: "",
			Node: pe.node(node),
		})
	case "constructor":
		return Exp(&ExpCon{
			name: pe.text(node),
			canonical: "",
			module: "",
			Node: pe.node(node),
		})
	case "prefix_id":
		return pe.parseExp(node.NamedChild(0))
	case "operator":
		return Exp(&ExpVar{
			name: pe.text(node),
		  module: "",
			canonical: "",
			Node: pe.node(node),
		})
	case "apply":
		return Exp(&ExpApp{
			exp1: pe.parseExp(node.Child(0)),
			exp2: pe.parseExp(node.Child(1)),
			Node: pe.node(node),
		})
	case "infix":
  	lhs := pe.parseExp(pe.child(node, "left_operand"))
		return pe.parseInfix(node, lhs)
	}
	return nil
}


func (pe parseEnv) getOperatorModule(node *treesitter.Node) (string, string) {
	operator := ""
	module := ""
	operatorNode := pe.child(node, "operator")
  switch operatorNode.Kind() {
	case "constructor_operator", "operator":
		operator = pe.text(operatorNode)
	case "qualified":
		operator = pe.text(pe.child(operatorNode, "id"))
		module = pe.text(pe.child(operatorNode, "module"))
	case "infix_id":
		backStickOperator := operatorNode.NamedChild(0)
		switch backStickOperator.Kind() {
		case "variable", "constructor_operator":
			operator = pe.text(backStickOperator)
		case "qualified":
			operator = pe.text(pe.child(backStickOperator, "id"))
  		module = pe.text(pe.child(backStickOperator, "module"))
		}
	}
	return operator, module
}

func (pe parseEnv) parseInfix(node *treesitter.Node, lhs Exp) Exp {
	operatorL, moduleL := pe.getOperatorModule(node)
  operatorNodeL:= pe.child(node, "operator")
	rhsNode := pe.child(node, "right_operand")
	leftPriority := false
	leftAssociative := false
	if rhsNode.Kind() == "infix" {
		operatorR, _ := pe.getOperatorModule(rhsNode)
		leftPriority = pe.fix(operatorR) <= pe.fix(operatorL)
		leftAssociative = operatorL == operatorR && pe.assoc(operatorL) == "r"
	}

	if leftPriority || leftAssociative {
		newLhs := Exp(&ExpInfix{
			exp1: lhs,
			exp2: pe.parseExp(pe.child(rhsNode, "left_operand")),
			op: ExpVar{
				name: operatorL,
				module: moduleL,
				canonical: "",
				Node: pe.node(operatorNodeL),
			},
			Node: Node{
				id: pe.id(),
				loc: mergeLoc(lhs.loc(), pe.loc(operatorNodeL)),
			},
		})
		return pe.parseInfix(rhsNode, newLhs)
	} else {
		rightOperand := pe.child(node, "right_operand")
		return Exp(&ExpInfix{
			exp1: lhs,
			exp2: pe.parseExp(rightOperand),
			op: ExpVar{
				name: operatorL,
				module: moduleL,
				canonical: "",
				Node: pe.node(operatorNodeL),
			},
			Node: pe.node(node),
		})
	}
}
