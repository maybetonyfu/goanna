package parser

import (
	"slices"

	treesitter "github.com/tree-sitter/go-tree-sitter"
)

func (pe parseEnv) parseDeclHead(node *treesitter.Node) DeclHead {
	name := pe.text(pe.child(node, "name"))
	types := pe.parseTypes(pe.children(node, "patterns:bind"))
	typeVars := make([]TyVar, len(types))
	for i, t := range types {
		typeVars[i] = *t.(*TyVar)
	}
	return DeclHead{
		Name:      name,
		Canonical: "",
		TypeVars:  typeVars,
		Node:      pe.node(node),
	}
}

func (pe parseEnv) parseImport(node *treesitter.Node) Import {
	// Get module name
	moduleNode := pe.child(node, "module")
	moduleName := ""
	if moduleNode != nil {
		moduleName = pe.text(moduleNode)
	}

	// Check if qualified by looking through all children (not just named ones)
	qualified := false
	cursor := node.Walk()
	allChildren := node.Children(cursor)
	for _, child := range allChildren {
		if child.Kind() == "qualified" {
			qualified = true
			break
		}
	}

	// Get alias if present (the module node after "as")
	alias := ""
	for i, child := range allChildren {
		if child.Kind() == "as" && i+1 < len(allChildren) && allChildren[i+1].IsNamed() {
			alias = pe.text(&allChildren[i+1])
			break
		}
	}

	// Check if hiding
	hiding := false
	for _, child := range allChildren {
		if child.Kind() == "hiding" {
			hiding = true
			break
		}
	}

	// Get import items list
	var items []string
	if itemsNode := pe.child(node, "names"); itemsNode != nil {
		// Get the items (import_name nodes) from the field "name"
		itemNodes := pe.children(itemsNode, "name")
		items = make([]string, len(itemNodes))
		for i, itemNode := range itemNodes {
			// Get the name from inside the import_name
			// The import_name structure has either a name, variable, or type child
			var nameStr string
			nameNode := pe.child(&itemNode, "name")
			if nameNode != nil {
				nameStr = pe.text(nameNode)
			} else {
				nameNode = pe.child(&itemNode, "variable")
				if nameNode != nil {
					nameStr = pe.text(nameNode)
				} else {
					nameNode = pe.child(&itemNode, "type")
					if nameNode != nil {
						nameStr = pe.text(nameNode)
					}
				}
			}
			items[i] = nameStr
		}
	}

	return Import{
		Module:    moduleName,
		Qualified: qualified,
		Alias:     alias,
		Items:     items,
		Hiding:    hiding,
		Node:      pe.node(node),
	}
}

func (pe parseEnv) parseDataCons(nodes []treesitter.Node) []DataCon {
	dataCons := make([]DataCon, len(nodes))
	for i, node := range nodes {
		dataCons[i] = pe.parseDataCon(&node)
	}
	return dataCons
}

func (pe parseEnv) parseDataCon(node *treesitter.Node) DataCon {
	name := pe.text(pe.child(node, "Constructor:name"))
	types := pe.parseTypes(pe.children(node, "Constructor:field"))
	return DataCon{
		Name:      name,
		Canonical: "",
		Tys:       types,
		Node:      pe.node(node),
	}
}

func (pe parseEnv) parseDecls(nodes []treesitter.Node) []Decl {
	decls := make([]Decl, len(nodes))
	for i, node := range nodes {
		decls[i] = pe.parseDecl(&node)
	}
	return decls
}

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
		var nameNodes []treesitter.Node
		if pe.child(node, "names") != nil {
			nameNodes = pe.children(node, "names:name")
		} else {
			nameNodes = []treesitter.Node{*pe.child(node, "name")}
		}

		names := make([]string, len(nameNodes))
		for i, nameNode := range nameNodes {
			switch nameNode.Kind() {
			case "prefix_id":
				names[i] = pe.text(nameNode.NamedChild(0))
			default:
				names[i] = pe.text(&nameNode)
			}
		}

		ty := pe.parseType(pe.child(node, "type"))
		return Decl(&TypeSig{
			Names:      names,
			Ty:         ty,
			Canonicals: []string{},
			Node:       pe.node(node),
		})
	case "data_type":
		dHead := pe.parseDeclHead(node)
		constructorNodes := pe.children(node, "constructors:constructor")
		constructors := pe.parseDataCons(constructorNodes)

		// Parse deriving clause
		var deriving []TyCon
		derivingNode := pe.child(node, "deriving")
		if derivingNode != nil {
			classesNode := pe.child(derivingNode, "classes")
			if classesNode != nil {
				// Parse the type (could be a single name or a tuple)
				ty := pe.parseType(classesNode)
				if tycon, ok := ty.(*TyCon); ok {
					deriving = append(deriving, *tycon)
				} else if tytuple, ok := ty.(*TyTuple); ok {
					// Extract TyCons from the tuple
					for _, innerTy := range tytuple.Tys {
						if tycon, ok := innerTy.(*TyCon); ok {
							deriving = append(deriving, *tycon)
						}
					}
				}
			}
		}

		return Decl(&DataDecl{
			DHead:        dHead,
			Constructors: constructors,
			Deriving:     deriving,
			Node:         pe.node(node),
		})
	case "class":
		assertions := pe.parseAssertions(pe.child(node, "context"))
		dHead := pe.parseDeclHead(node)
		decls := pe.parseDecls(pe.children(node, "declarations:*"))
		return Decl(&ClassDecl{
			Assertions: assertions,
			DHead:      dHead,
			Decls:      decls,
			Node:       pe.node(node),
		})

	case "instance":
		assertions := pe.parseAssertions(pe.child(node, "context"))
		var module, name string
		if pe.child(node, "name").Kind() == "qualified" {
			module = pe.text(pe.child(node, "Name:module"))
			name = pe.text(pe.child(node, "Name:id"))
		} else {
			module = ""
			name = pe.text(pe.child(node, "name"))
		}
		tys := pe.parseTypes(pe.children(node, "patterns:*"))
		decls := pe.parseDecls(pe.children(node, "declarations:*"))
		return Decl(&InstDecl{
			Assertions: assertions,
			Name:       name,
			Module:     module,
			Types:      tys,
			Body:       decls,
			Node:       pe.node(node),
		})

	case "function", "bind":
		// For function bindings, check if there are patterns (function arguments)
		nameNode := pe.child(node, "name")
		patternsNode := pe.child(node, "patterns")

		var pat Pat
		if patternsNode != nil {
			// Function with arguments - create PApp
			// Get the function name
			funcName := pe.text(nameNode)
			funcVar := PVar{
				Name:      funcName,
				Canonical: "",
				Module:    "",
				Node:      pe.node(nameNode),
			}

			// Get the argument patterns
			patNodes := pe.children(patternsNode, "*")
			pats := make([]Pat, len(patNodes))
			for i, patNode := range patNodes {
				pats[i] = pe.parsePat(&patNode)
			}

			pat = Pat(&PApp{
				Constructor: funcVar,
				Pats:        pats,
				Node:        pe.node(node),
			})
		} else {
			// Simple pattern binding (no arguments)
			pat = pe.parsePat(nameNode)
		}

		rhs := pe.parseRhs(node)
		return Decl(&PatBind{
			Pat:  pat,
			Rhs:  rhs,
			Node: pe.node(node),
		})
	case "fixity":
	case "type_synomym":
		dHead := pe.parseDeclHead(node)
		ty := pe.parseType(pe.child(node, "type"))
		return Decl(&TypeDecl{
			DHead: dHead,
			Ty:    ty,
			Node:  pe.node(node),
		})
	case "haddock", "comment":
		// Skip comments and haddock documentation
		return nil
	default:
		panic("Unknown declaration type: " + node.Kind())
	}
	return nil
}

func (pe parseEnv) parsePats(nodes []treesitter.Node) []Pat {
	pats := make([]Pat, len(nodes))
	for i, node := range nodes {
		pats[i] = pe.parsePat(&node)
	}
	return pats
}

func (pe parseEnv) parsePat(node *treesitter.Node) Pat {
	switch node.Kind() {
	case "qualified":
		module := pe.text(pe.child(node, "module"))
		name := pe.text(pe.child(node, "id"))
		return Pat(&PVar{
			Name:      name,
			Module:    module,
			Canonical: "",
			Node:      pe.node(node),
		})

	case "prefix_id":
		operator := node.NamedChild(0)
		name := pe.text(operator)
		return Pat(&PVar{
			Name:      name,
			Canonical: "",
			Module:    "",
			Node:      pe.node(node),
		})

	case "variable", "constructor_operator", "constructor":
		name := pe.text(node)
		return Pat(&PVar{
			Name:      name,
			Canonical: "",
			Module:    "",
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
		pats := pe.parsePats(pe.children(node, "element"))
		return Pat(&PTuple{
			Pats: pats,
			Node: pe.node(node),
		})

	case "list":
		pats := pe.parsePats(pe.children(node, "element"))
		return Pat(&PList{
			Pats: pats,
			Node: pe.node(node),
		})

	case "apply":
		pats := []Pat{}
		currentNode := node
		var constructor PVar
		for {
			if currentNode.Kind() == "apply" {
				h := pe.parsePat(currentNode.Child(1))
				pats = append([]Pat{h}, pats...)
				currentNode = currentNode.Child(0)
			} else {
				constructor = *(pe.parsePat(currentNode).(*PVar))
				break
			}
		}

		return Pat(&PApp{
			Constructor: constructor,
			Pats:        pats,
			Node:        pe.node(node),
		})

	case "infix":
		op := *(pe.parsePat(pe.child(node, "operator")).(*PVar))
		pat1 := pe.parsePat(pe.child(node, "left_operand"))
		pat2 := pe.parsePat(pe.child(node, "right_operand"))
		return Pat(&PInfix{
			Op:   op,
			Pat1: pat1,
			Pat2: pat2,
			Node: pe.node(node),
		})

	default:
		panic("Unknown pat type: " + node.Kind())

	}
}

func (pe parseEnv) parseLit(node *treesitter.Node) *Lit {
	return &Lit{
		Lit:     node.Child(0).Kind(),
		Content: pe.text(node),
		Node:    pe.node(node),
	}
}

func (pe parseEnv) parseRhs(node *treesitter.Node) Rhs {
	rhsNodes := pe.children(node, "match")
	wheres := pe.parseDecls(pe.children(node, "binds:decl"))
	isUnguarded := len(rhsNodes) > 0 && rhsNodes[0].ChildByFieldName("guards") == nil
	branches := make([]GuardBranch, 0)

	for _, rhsNode := range rhsNodes {
		rhs := &rhsNode
		// Parse the expression from the RHS
		rhsExp := pe.parseExp(pe.child(rhs, "expression"))

		if isUnguarded {
			return Rhs(&UnguardedRhs{
				Exp:    rhsExp,
				Wheres: wheres,
				Node:   pe.node(rhs),
			})
		} else {
			// Get the guards node, then parse the guard expressions from it
			guardsNode := pe.child(rhs, "guards")
			var guards []Exp
			if guardsNode != nil {
				guardNodes := pe.children(guardsNode, "guard")
				guards = pe.parseExps(guardNodes)
			}
			branches = append(branches, GuardBranch{
				Exp:    rhsExp,
				Guards: guards,
				Node:   pe.node(rhs),
			})
		}
	}
	return Rhs(&GuardedRhs{
		Wheres:   wheres,
		Branches: branches,
		Node:     pe.node(node),
	})
}

func (pe parseEnv) parseAssertions(node *treesitter.Node) []Type {
	if node == nil {
		return []Type{}
	}
	switch node.Kind() {
	case "context":
		// Extract the actual assertion from the context node
		contextNode := pe.child(node, "context")
		if contextNode != nil {
			return pe.parseAssertions(contextNode)
		}
		// If no context child, try parsing the whole node as a type
		ty := pe.parseType(node)
		if ty != nil {
			return []Type{ty}
		}
		return []Type{}
	case "parens":
		return []Type{
			pe.parseType(node.Child(1)),
		}
	case "tuple":
		return pe.parseTypes(pe.children(node, "*"))

	case "apply":
		return []Type{
			pe.parseType(node),
		}
	default:
		// For single assertion type that doesn't match specific cases, try parsing as type
		ty := pe.parseType(node)
		if ty != nil {
			return []Type{ty}
		}
		return []Type{}
	}
}
func (pe parseEnv) parseTypes(nodes []treesitter.Node) []Type {
	types := make([]Type, 0, len(nodes))
	for _, node := range nodes {
		ty := pe.parseType(&node)
		if ty != nil {
			types = append(types, ty)
		}
	}
	return types
}
func (pe parseEnv) parseType(node *treesitter.Node) Type {
	if node == nil {
		return nil
	}
	switch node.Kind() {
	case "qualified":
		module := pe.text(pe.child(node, "module"))
		name := pe.text(pe.child(node, "id"))
		return Type(&TyCon{
			Name:      name,
			Canonical: "top",
			Module:    module,
			Node:      pe.node(node),
		})

	case "context":
		assertions := pe.parseAssertions(pe.child(node, "context"))
		ty := pe.parseType(pe.child(node, "type"))
		return Type(&TyForall{
			Assertions: assertions,
			Ty:         ty,
			Node:       pe.node(node),
		})

	case "unit":
		return Type(&TyCon{
			Name:      "top",
			Canonical: "top",
			Module:    "",
			Node:      pe.node(node),
		})

	case "name":
		return Type(&TyCon{
			Name:      pe.text(node),
			Canonical: "",
			Module:    "",
			Node:      pe.node(node),
		})

	case "variable":
		return Type(&TyVar{
			Name:      pe.text(node),
			Canonical: "",
			Node:      pe.node(node),
		})
	case "apply":
		ty1 := pe.parseType(pe.child(node, "constructor"))
		ty2 := pe.parseType(pe.child(node, "argument"))
		return Type(&TyApp{
			Ty1:  ty1,
			Ty2:  ty2,
			Node: pe.node(node),
		})
	case "parens":
		// Parenthesized type - parse the inner type
		typeNode := pe.child(node, "type")
		if typeNode == nil {
			// If no "type" field, try the first named child (for deriving clause)
			cursor := node.Walk()
			namedChildren := node.NamedChildren(cursor)
			if len(namedChildren) > 0 {
				return pe.parseType(&namedChildren[0])
			}
		}
		return pe.parseType(typeNode)
	case "function":
		ty1 := pe.parseType(pe.child(node, "parameter"))
		ty2 := pe.parseType(pe.child(node, "result"))
		return Type(&TyFunction{
			Ty1:  ty1,
			Ty2:  ty2,
			Node: pe.node(node),
		})
	case "tuple":
		// Try to get elements from "element" field first, then try direct named children
		elementNodes := pe.children(node, "element")
		if len(elementNodes) == 0 {
			// If no "element" field, parse the named children directly
			cursor := node.Walk()
			elementNodes = node.NamedChildren(cursor)
		}
		types := pe.parseTypes(elementNodes)
		return Type(&TyTuple{
			Tys:  types,
			Node: pe.node(node),
		})
	case "list":
		ty := pe.parseType(pe.child(node, "element"))
		return Type(&TyList{
			Ty:   ty,
			Node: pe.node(node),
		})
	case "prefix_list":
		return Type(&TyCon{
			Name:      "list",
			Canonical: "list",
			Module:    "",
			Node:      pe.node(node),
		})

	case "prefix_tuple":
		return Type(&TyCon{
			Name:      "tuple",
			Canonical: " tuple",
			Module:    "",
			Node:      pe.node(node),
		})

	case "prefix_id":
		opName := pe.text(&pe.children(node, "*")[0])
		if opName == "->" {
			return Type(&TyCon{
				Name:      "function",
				Canonical: "function",
				Module:    "",
				Node:      pe.node(node),
			})
		} else {
			panic("Unknonw op Name: " + opName)
		}
	default:
		panic("Unknown type node: " + node.Kind())
	}
}

func (pe parseEnv) parseExps(nodes []treesitter.Node) []Exp {
	exps := make([]Exp, len(nodes))
	for i, node := range nodes {
		exps[i] = pe.parseExp(&node)
	}
	return exps
}

func (pe parseEnv) parseExp(node *treesitter.Node) Exp {
	switch node.Kind() {
	case "qualified":
		module := pe.text(pe.child(node, "module"))
		id := pe.child(node, "id")
		return Exp(&ExpVar{
			Name:      pe.text(id),
			Module:    module,
			Canonical: "",
			Node:      pe.node(node),
		})

	case "literal":
		return Exp(pe.parseLit(node))

	case "variable", "constructor", "operator", "constructor_operator":
		return Exp(&ExpVar{
			Name:      pe.text(node),
			Module:    "",
			Canonical: "",
			Node:      pe.node(node),
		})

	case "parens":
		return pe.parseExp(pe.child(node, "expression"))

	case "boolean":
		// Boolean expressions (used in guards, conditions, etc.)
		// The actual expression is the child of the boolean node
		return pe.parseExp(node.Child(0))

	case "negation":
		// Negation (e.g., -x, -1)
		innerExp := pe.parseExp(pe.child(node, "expression"))
		return Exp(&ExpApp{
			Exp1: Exp(&ExpVar{
				Name:      "negate",
				Canonical: "",
				Module:    "",
				Node:      pe.node(node),
			}),
			Exp2: innerExp,
			Node: pe.node(node),
		})

	case "unit":
		return Exp(&ExpVar{
			Name:      "unit",
			Canonical: "",
			Module:    "",
			Node:      pe.node(node),
		})

	case "prefix_id", "infix_id":
		return pe.parseExp(node.NamedChild(0))

	case "apply":
		return Exp(&ExpApp{
			Exp1: pe.parseExp(node.Child(0)),
			Exp2: pe.parseExp(node.Child(1)),
			Node: pe.node(node),
		})

	case "left_section":
		left := pe.parseExp(pe.child(node, "left_operand"))
		operator := pe.child(node, "operator")
		op := pe.parseExp(operator)
		return Exp(&ExpLeftSection{
			Left: left,
			Op:   op,
			Node: pe.node(node),
		})
	case "right_section":
		right := pe.parseExp(pe.child(node, "right_operand"))
		operator := node.NamedChild(0)
		op := pe.parseExp(operator)
		return Exp(&ExpRightSection{
			Right: right,
			Op:    op,
			Node:  pe.node(node),
		})
	case "lambda":
		pats := pe.parsePats(pe.children(node, "patterns:*"))
		exp := pe.parseExp(pe.child(node, "expression"))
		return Exp(&ExpLambda{
			Pats: pats,
			Exp:  exp,
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
			Exp:   exp,
			Binds: binds,
			Node:  pe.node(node),
		})

	case "conditional":
		condExp := pe.parseExp(pe.child(node, "if"))
		thenExp := pe.parseExp(pe.child(node, "then"))
		elseExp := pe.parseExp(pe.child(node, "else"))
		return Exp(&ExpIf{
			Cond:    condExp,
			IfTrue:  thenExp,
			IfFalse: elseExp,
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
			Exp:  exp,
			Alts: alts,
			Node: pe.node(node),
		})

	case "tuple":
		elems := pe.children(node, "element")
		exps := make([]Exp, len(elems))
		for i, elem := range elems {
			exps[i] = pe.parseExp(&elem)
		}
		return Exp(&ExpTuple{
			Exps: exps,
			Node: pe.node(node),
		})

	case "list":
		exps := pe.parseExps(pe.children(node, "element"))
		return Exp(&ExpList{
			Exps: exps,
			Node: pe.node(node),
		})

	case "arithmetic_sequence":
		start := pe.child(node, "from")
		end := pe.child(node, "to")
		if end == nil {
			exp := pe.parseExp(start)
			return Exp(&ExpEnumFrom{
				Exp:  exp,
				Node: pe.node(node),
			})
		} else {
			exp1 := pe.parseExp(start)
			exp2 := pe.parseExp(end)
			return Exp(&ExpEnumFromTo{
				Exp1: exp1,
				Exp2: exp2,
				Node: pe.node(node),
			})
		}
	case "do":
		statementNodes := pe.children(node, "statement")
		stmts := make([]Statement, len(statementNodes))
		for i, statementNode := range statementNodes {
			switch statementNode.Kind() {
			case "exp":
				stmts[i] = Statement(&Qualifier{
					Exp:  pe.parseExp(statementNode.NamedChild(0)),
					Node: pe.node(&statementNode),
				})
			case "bind":
				pat := pe.parsePat(pe.child(&statementNode, "pattern"))
				exp := pe.parseExp(pe.child(&statementNode, "expression"))
				stmts[i] = Statement(&Generator{
					Pat:  pat,
					Exp:  exp,
					Node: pe.node(&statementNode),
				})
			case "let":
				declNodes := pe.children(&statementNode, "binds:decl")
				binds := make([]Decl, len(declNodes))
				for i, declNode := range declNodes {
					binds[i] = pe.parseDecl(&declNode)
				}
				stmts[i] = Statement(&LetStmt{
					Binds: binds,
					Node:  pe.node(&statementNode),
				})
			}
		}
		return Exp(&ExpDo{
			Stmts: stmts,
			Node:  pe.node(node),
		})
	case "list_comprehension":
		exp := pe.parseExp(pe.child(node, "expression"))
		generators := make([]Generator, 0)
		guards := make([]Exp, 0)
		for _, qualifierNode := range pe.children(node, "qualifiers:qualifier") {
			switch qualifierNode.Kind() {
			case "generator":
				pat := pe.parsePat(pe.child(&qualifierNode, "pattern"))
				exp := pe.parseExp(pe.child(&qualifierNode, "expression"))
				generators = append(generators, Generator{
					Pat:  pat,
					Exp:  exp,
					Node: pe.node(&qualifierNode),
				})
			case "boolean":
				guards = append(guards, pe.parseExp(qualifierNode.Child(0)))
			}
		}
		return Exp(&ExpComprehension{
			Exp:        exp,
			Generators: generators,
			Guards:     guards,
			Node:       pe.node(node),
		})

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
		if op.Name == prev.Name && highestIndex == i-1 && pe.assoc(op.Name) == "r" {
			// two same operators, use the first unless it's right associative
			highestIndex = i
			continue
		}

		if pe.fix(op.Name) > pe.fix(ops[i-1].Name) {
			highestIndex = i
			continue
		}
	}

	left := exps[highestIndex]
	right := exps[highestIndex+1]
	op := ops[highestIndex]

	exp := Exp(&ExpInfix{
		Exp1: left,
		Exp2: right,
		Op:   op,
		Node: Node{
			id:  pe.id(),
			loc: mergeLoc(left.Loc(), right.Loc()),
		},
	})

	ops = slices.Concat(ops[:highestIndex], ops[highestIndex+1:])
	exps = slices.Concat(exps[:highestIndex], []Exp{exp}, exps[highestIndex+2:])
	return pe.buildInfix(exps, ops)
}

func (pe parseEnv) parseAlts(nodes []treesitter.Node) []Alt {
	alts := make([]Alt, len(nodes))
	for i, node := range nodes {
		alts[i] = pe.parseAlt(&node)
	}
	return alts
}

func (pe parseEnv) parseAlt(node *treesitter.Node) Alt {
	pat := pe.parsePat(pe.child(node, "pattern"))
	exp := pe.parseExp(pe.child(node, "match:expression"))
	binds := pe.parseDecls(pe.children(node, "binds:decl"))
	return Alt{
		Pat:   pat,
		Exp:   exp,
		Binds: binds,
		Node:  pe.node(node),
	}
}
