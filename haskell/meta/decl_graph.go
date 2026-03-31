package meta

import (
	"goanna/haskell/parser"
)

type edge struct {
	parent string
	child  string
}


type dgEnv struct {
  parents []string
	pairs []edge
}

func (de *dgEnv) visitNodes ( ast parser.AST) {
	switch node := ast.(type) {
	case *parser.PatBind:
		newDecl := canonicalFromPat(node.Pat)
		for _, parent := range de.parents {
			de.pairs = append(de.pairs, edge {parent, newDecl})
		}
		de.parents = append(de.parents, newDecl)
	}
}

func (de *dgEnv) leaveNodes ( ast parser.AST) {
	switch ast.(type) {
	case *parser.PatBind:
		if len(de.parents) > 0 {
			de.parents = de.parents[:len(de.parents)-1]
		}
	}
}


func GetDeclGraph(modules []*parser.Module) map[string][]string {
	env := dgEnv {
		parents: make([]string, 0),
  	pairs: make([]edge, 0),
	}
	traverser := parser.NewTraverser(
		func(v int, ast parser.AST, parent parser.AST) int {
			env.visitNodes(ast)
			return v
		},
		func(_ int, ast parser.AST, parent parser.AST) {
			env.leaveNodes(ast)
		},
	  0)

	for _, m:= range modules {
		traverser.Visit(m, nil)
	}


	return pairsToGraph(env.pairs)
}


func pairsToGraph(pairs []edge) map[string][]string {
	graph := make(map[string][]string)
	for _, e := range pairs {
		if e.parent == "" {
			if _, ok := graph[e.child]; !ok {
				graph[e.child] = []string{}
			}
		} else {
			graph[e.child] = append(graph[e.child], e.parent)
		}
	}
	return graph
}


func canonicalFromPat(pat parser.Pat) string {
	if pat == nil {
		return ""
	}
	switch p := pat.(type) {
	case *parser.PVar:
		return p.Canonical
	case *parser.PApp:
		return p.Constructor.Canonical
	default:
		return ""
	}
}
