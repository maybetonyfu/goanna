package haskell

import (
	mapset "github.com/deckarep/golang-set/v2"
	prolog_tool "mil/prolog-tool"
	"slices"
	"strings"
)

type constructorType int

const (
	function constructorType = iota
	tuple
	list
	adt
	unknown
)

type Pair struct {
	conType constructorType
	first   prolog_tool.Term
	second  prolog_tool.Term
}

type Printer struct {
	varNames    []string
	varMapping  map[string]string
	typeClasses map[string]mapset.Set[string]
	jobId       int
	jobVars     map[int]mapset.Set[string]
}

func NewPrinter() *Printer {
	varNames := []string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j",
		"k", "l", "m", "n", "o", "p", "q", "r", "s", "t",
		"u", "v", "w", "x", "y", "z",
		"a0", "b0", "c0", "d0", "e0", "f0", "g0", "h0",
		"i0", "j0", "k0", "l0", "m0", "n0", "o0", "p0",
		"q0", "r0", "s0", "t0", "u0", "v0", "w0", "x0",
		"y0", "z0",
	}
	return &Printer{
		jobId:       0,
		varNames:    varNames,
		varMapping:  make(map[string]string),
		typeClasses: make(map[string]mapset.Set[string]),
		jobVars:     make(map[int]mapset.Set[string]),
	}
}

func (p *Printer) getOrCreateVarName(prologName string) string {
	var v string
	if _, ok := p.varMapping[prologName]; ok {
		v = p.varMapping[prologName]
	} else {
		varName := p.varNames[0]
		p.varNames = p.varNames[1:]
		p.varMapping[prologName] = varName
		p.typeClasses[varName] = mapset.NewSet[string]()
		v = varName
	}

	if _, ok := p.jobVars[p.jobId]; ok {
		p.jobVars[p.jobId].Add(v)
	} else {
		p.jobVars[p.jobId] = mapset.NewSet[string](v)
	}
	return v
}

func (p *Printer) printVar(term prolog_tool.Var) string {
	return p.getOrCreateVarName(term.Value)
}

func makePair(term prolog_tool.Term) Pair {
	switch term.(type) {
	case prolog_tool.Compound:
		termC := term.(prolog_tool.Compound)
		if termC.Value != "pair" {
			return Pair{unknown, nil, nil}
		}
		firstArg := termC.Args[0]
		secondArg := termC.Args[1]
		switch firstArg.(type) {
		case prolog_tool.Compound:
			firstArgC := firstArg.(prolog_tool.Compound)
			firstArgCArg := firstArgC.Args[0]
			switch firstArgC.Value {
			case "function":
				return Pair{function, firstArgCArg, secondArg}
			case "tuple":
				return Pair{tuple, firstArgCArg, secondArg}
			default:
				panic("Unknown compound type")
			}
		case prolog_tool.Atom:
			firstArgC := firstArg.(prolog_tool.Atom)
			if firstArgC.Value == "list" {
				return Pair{list, secondArg, nil}
			}

			return Pair{adt, firstArg, secondArg}

		default:
			return Pair{adt, firstArg, secondArg}
		}
	default:
		return Pair{unknown, nil, nil}
	}
}

func unrollFunction(term prolog_tool.Term) []prolog_tool.Term {
	pair := makePair(term)
	if pair.conType == function {
		return slices.Concat([]prolog_tool.Term{
			pair.first,
		}, unrollTuple(pair.second))
	} else {
		return []prolog_tool.Term{
			term,
		}
	}
}

func unrollTuple(term prolog_tool.Term) []prolog_tool.Term {
	pair := makePair(term)
	if pair.conType == tuple {
		return slices.Concat([]prolog_tool.Term{
			pair.first,
		}, unrollFunction(pair.second))
	} else {
		return []prolog_tool.Term{
			term,
		}
	}
}

func unrollADT(term prolog_tool.Term) []prolog_tool.Term {
	pair := makePair(term)
	if pair.conType == adt {
		return slices.Concat([]prolog_tool.Term{
			pair.first,
		}, unrollADT(pair.second))
	} else {
		return []prolog_tool.Term{
			term,
		}
	}
}

func (p *Printer) printAtom(term prolog_tool.Atom) string {
	parts := strings.Split(term.Value, "_")
	if len(parts) == 1 {
		return parts[0]
	} else {
		return parts[len(parts)-1]
	}
}

func (p *Printer) printCompound(term prolog_tool.Compound) string {
	switch {
	case term.Value == "has":
		typeClasses := term.Args[0].(prolog_tool.List)
		typeVar := p.printVar(term.Args[1].(prolog_tool.Var))
		for _, class := range typeClasses.Values {
			className := p.printAtom(class.(prolog_tool.Atom))
			p.typeClasses[typeVar].Add(className)
		}
		return typeVar

	case makePair(term).conType == function:
		args := unrollFunction(term)
		argsText := make([]string, len(args))
		for i, arg := range args {
			if makePair(arg).conType == function && i != len(args)-1 {
				argsText[i] = "(" + p.PrintTerm(arg) + ")"
			} else {
				argsText[i] = p.PrintTerm(arg)
			}
		}
		return strings.Join(argsText, "->")
	case makePair(term).conType == list:
		content := makePair(term).first
		return "[" + p.PrintTerm(content) + "]"
	case makePair(term).conType == tuple:
		args := unrollTuple(term)
		argsText := make([]string, len(args))
		for i, arg := range args {
			argsText[i] = p.PrintTerm(arg)
		}
		return "(" + strings.Join(argsText, ",") + ")"
	case makePair(term).conType == adt:
		args := unrollADT(term)
		argsText := make([]string, len(args))
		for i, arg := range args {
			if makePair(arg).conType == adt {
				argsText[i] = "(" + p.PrintTerm(arg) + ")"
			} else {
				argsText[i] = p.PrintTerm(arg)
			}

		}
		return strings.Join(argsText, " ")
	default:
		panic("Unknown compound type")
	}
}

func (p *Printer) PrintTerm(term prolog_tool.Term) string {
	switch term.(type) {
	case prolog_tool.Atom:
		return p.printAtom(term.(prolog_tool.Atom))

	case prolog_tool.Var:
		return p.printVar(term.(prolog_tool.Var))

	case prolog_tool.Compound:
		return p.printCompound(term.(prolog_tool.Compound))

	default:
		panic("Unknown term type")
	}
}

func (p *Printer) GetType(term prolog_tool.Term) string {
	typeString := p.PrintTerm(term)
	typeVars := p.jobVars[p.jobId]
	if typeVars == nil {
		return typeString
	}
	classRequirements := make([]string, 0)
	for v := range typeVars.Iter() {
		classes := p.typeClasses[v]
		if classes == nil {
			continue
		}
		for c := range classes.Iter() {
			classRequirements = append(classRequirements, c+" "+v)
		}
	}
	var context string
	if len(classRequirements) == 0 {
		context = ""
	} else if len(classRequirements) == 1 {
		context = classRequirements[0] + "=>"
	} else {
		context = "(" + strings.Join(classRequirements, ",") + ")=>"
	}
	p.jobId += 1
	return context + typeString
}
