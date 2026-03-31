package meta

import (
	"goanna/haskell/parser"
)

type paramsEnv struct {
	// stack of current decl scope names (innermost last)
	declStack []string
	result    map[string][]string
}

func (pe *paramsEnv) currentDecl() string {
	if len(pe.declStack) == 0 {
		return ""
	}
	return pe.declStack[len(pe.declStack)-1]
}

func (pe *paramsEnv) addParams(decl string, params []string) {
	if decl == "" || len(params) == 0 {
		return
	}
	pe.result[decl] = append(pe.result[decl], params...)
}

// patsToVars extracts all variable names bound by a pattern, recursively.
func patsToVars(pat parser.Pat) []string {
	if pat == nil {
		return nil
	}
	switch p := pat.(type) {
	case *parser.PVar:
		if p.Canonical != "" {
			return []string{p.Canonical}
		}
		return []string{p.Name}
	case *parser.PWildcard:
		return []string{"_"}
	case *parser.PApp:
		var vars []string
		for _, sub := range p.Pats {
			vars = append(vars, patsToVars(sub)...)
		}
		return vars
	case *parser.PTuple:
		var vars []string
		for _, sub := range p.Pats {
			vars = append(vars, patsToVars(sub)...)
		}
		return vars
	case *parser.PList:
		var vars []string
		for _, sub := range p.Pats {
			vars = append(vars, patsToVars(sub)...)
		}
		return vars
	case *parser.PInfix:
		vars := patsToVars(p.Pat1)
		vars = append(vars, patsToVars(p.Pat2)...)
		return vars
	case *parser.Lit:
		return nil
	default:
		return nil
	}
}

func (pe *paramsEnv) enter(ast parser.AST, parent parser.AST) {
	switch node := ast.(type) {
	case *parser.PatBind:
		switch p := node.Pat.(type) {
		case *parser.PApp:
			// Function definition: `f a b = ...`
			// The constructor of PApp is the function name; its Pats are the parameters.
			declName := p.Constructor.Canonical
			if declName == "" {
				declName = p.Constructor.Name
			}
			pe.declStack = append(pe.declStack, declName)
			var params []string
			for _, pat := range p.Pats {
				params = append(params, patsToVars(pat)...)
			}
			pe.addParams(declName, params)
		case *parser.PVar:
			// Simple binding: `x = ...` — no params at this level, but push scope
			// for nested lambdas/alts to associate with this decl.
			declName := p.Canonical
			if declName == "" {
				declName = p.Name
			}
			pe.declStack = append(pe.declStack, declName)
			// Ensure the decl appears in the result even with no params.
			if _, ok := pe.result[declName]; !ok {
				pe.result[declName] = []string{}
			}
		default:
			pe.declStack = append(pe.declStack, "")
		}

	case *parser.Alt:
		// `case x of Pat -> ...`: the pat's variables are params of this alt,
		// attributed to the enclosing decl.
		decl := pe.currentDecl()
		vars := patsToVars(node.Pat)
		pe.addParams(decl, vars)

	case *parser.ExpLambda:
		// `\a b -> ...`: lambda params attributed to the enclosing decl.
		decl := pe.currentDecl()
		for _, pat := range node.Pats {
			vars := patsToVars(pat)
			pe.addParams(decl, vars)
		}
	}
}

func (pe *paramsEnv) leave(ast parser.AST) {
	switch ast.(type) {
	case *parser.PatBind:
		if len(pe.declStack) > 0 {
			pe.declStack = pe.declStack[:len(pe.declStack)-1]
		}
	}
}

// InheritParams takes a params map (decl -> own params) and a decl graph
// (child -> [parents], as returned by GetDeclGraph) and returns a new params
// map where each decl's param list is prefixed with all ancestor params,
// outermost ancestor first.
func InheritParams(params map[string][]string, graph map[string][]string) map[string][]string {
	// memo caches the fully-inherited param list for each decl.
	memo := make(map[string][]string)
	// visiting tracks the current DFS path to detect cycles.
	visiting := make(map[string]bool)

	var inherited func(decl string) []string
	inherited = func(decl string) []string {
		if result, ok := memo[decl]; ok {
			return result
		}
		if visiting[decl] {
			// cycle — return just own params to break it
			return params[decl]
		}
		visiting[decl] = true
		defer func() { visiting[decl] = false }()

		var ancestorParams []string
		for _, parent := range graph[decl] {
			ancestorParams = append(ancestorParams, inherited(parent)...)
		}
		own := params[decl]
		result := append(ancestorParams, own...)
		memo[decl] = result
		return result
	}

	out := make(map[string][]string, len(params))
	for decl := range params {
		out[decl] = inherited(decl)
	}
	return out
}

// GetDeclParams returns a map from declaration canonical name to the list of
// parameter variable names bound in that declaration, including:
//   - function parameters: `f a b = ...`
//   - case alt patterns:   `case x of Just a -> ...`
//   - lambda parameters:   `\a b -> ...`
func GetDeclParams(modules []*parser.Module) map[string][]string {
	env := &paramsEnv{
		declStack: make([]string, 0),
		result:    make(map[string][]string),
	}

	traverser := parser.NewTraverser(
		func(v int, ast parser.AST, parent parser.AST) int {
			env.enter(ast, parent)
			return v
		},
		func(_ int, ast parser.AST, parent parser.AST) {
			env.leave(ast)
		},
		0,
	)

	for _, m := range modules {
		traverser.Visit(m, nil)
	}

	return env.result
}
