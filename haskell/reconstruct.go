package haskell

import (
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"mil/inventory"
	prolog_tool "mil/prolog-tool"
	"slices"
	"strings"
	"text/template"
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

type MetaVar struct {
	tmpl           string
	skolem         bool
	preferredName  string
	appearedInJobs mapset.Set[int]
	typeClasses    mapset.Set[string]
	friendlyName   string
}

type Printer struct {
	varMapping map[string]*MetaVar
	currentJob int
}

func NewPrinter() *Printer {
	varmapping := make(map[string]*MetaVar)
	return &Printer{
		varMapping: varmapping,
		currentJob: 0,
	}
}

func (p *Printer) getOrCreateVarName(lookupName string, preferSameName bool) string {
	if mv, ok := p.varMapping[lookupName]; ok {
		p.varMapping[lookupName].appearedInJobs.Add(p.currentJob)
		return mv.tmpl
	} else {
		newVar := MetaVar{
			tmpl:           fmt.Sprintf("{{ $.%s }}", lookupName),
			skolem:         false,
			preferredName:  "",
			appearedInJobs: mapset.NewSet[int](p.currentJob),
			typeClasses:    mapset.NewSet[string](),
			friendlyName:   "",
		}

		if preferSameName {
			newVar.skolem = true
			newVar.preferredName = strings.Split(lookupName, "__")[0] // Remove the namespace of type var
		}
		p.varMapping[lookupName] = &newVar
		return newVar.tmpl
	}
}

func (p *Printer) printVar(term prolog_tool.Var) string {
	return p.getOrCreateVarName(term.Value, false)
}

func (p *Printer) printSkolemVar(term prolog_tool.Atom) string {
	return p.getOrCreateVarName(term.Value, true)
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
				return Pair{adt, firstArg, secondArg}
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
		}, unrollFunction(pair.second))
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
		}, unrollTuple(pair.second))
	} else {
		return []prolog_tool.Term{
			term,
		}
	}
}

func unrollADT(term prolog_tool.Term) []prolog_tool.Term {
	pair := makePair(term)
	if pair.conType == adt {
		return append(unrollADT(pair.first), pair.second)

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
	//fmt.Printf("Term: %v\n", term)
	switch {
	case term.Value == "has":
		typeClasses := term.Args[0].(prolog_tool.List)
		var typeVar string
		var lookupString string
		switch term.Args[1].(type) {
		case prolog_tool.Atom:
			// This is skolemized constant, we need to turn it into a type var, preferraby using the same letter
			typeVar = p.printSkolemVar(term.Args[1].(prolog_tool.Atom))
			lookupString = term.Args[1].(prolog_tool.Atom).Value
		case prolog_tool.Var:
			typeVar = p.printVar(term.Args[1].(prolog_tool.Var))
			lookupString = term.Args[1].(prolog_tool.Var).Value
		default:
			panic("has([..], X) where X is neither var or atom")
		}
		for _, class := range typeClasses.Values {
			className := p.printAtom(class.(prolog_tool.Atom))
			p.varMapping[lookupString].typeClasses.Add(className)
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

func findSuitableTypeVarName(preferredName string, usedNames []string) string {
	if slices.Contains(usedNames, preferredName) {
		var finalName string
		for n := range 100 {
			name := fmt.Sprintf("%s__%d", preferredName, n)
			if slices.Contains(usedNames, preferredName) {
				continue
			}
			finalName = name
			break
		}
		return finalName
	} else {
		return preferredName
	}
}

func findAvailableTypeVarName(usedNames []string) string {
	for _, c := range "abcdefghijklmnopqrstuvwxyz" {
		letter := string(c)
		if slices.Contains(usedNames, letter) {
			continue
		}
		return letter
	}

	for _, c := range "abcdefghijklmnopqrstuvwxyz" {
		letter := string(c) + "__0"
		if slices.Contains(usedNames, letter) {
			continue
		}
		return letter
	}

	for _, c := range "abcdefghijklmnopqrstuvwxyz" {
		letter := string(c) + "__1"
		if slices.Contains(usedNames, letter) {
			continue
		}
		return letter
	}

	panic("Could not find available type variable")
}

func allAssignedNames(varMap map[string]*MetaVar) []string {
	result := make([]string, 0)
	for _, meta := range varMap {
		if meta.friendlyName != "" {
			result = append(result, meta.friendlyName)
		}
	}
	return result
}

func (p *Printer) PrepareType(term prolog_tool.Term, jobId int) string {
	p.currentJob = jobId
	typeString := p.PrintTerm(term)
	return typeString
}

func (p *Printer) AssignVars() {
	for lookupName, metaVar := range p.varMapping {
		names := allAssignedNames(p.varMapping)
		if metaVar.skolem {
			if slices.Contains(names, metaVar.preferredName) {
				p.varMapping[lookupName].friendlyName = findSuitableTypeVarName(metaVar.preferredName, names)
			}
			metaVar.friendlyName = metaVar.preferredName
			continue
		}
	}

	for lookupName, metaVar := range p.varMapping {
		names := allAssignedNames(p.varMapping)
		if metaVar.skolem {
			continue
		}

		if metaVar.typeClasses.Contains("Monad") {
			p.varMapping[lookupName].friendlyName = findSuitableTypeVarName("m", names)
			continue
		}

		if metaVar.typeClasses.Contains("Applicative") ||
			metaVar.typeClasses.Contains("Alternative") ||
			metaVar.typeClasses.Contains("Functor") {
			p.varMapping[lookupName].friendlyName = findSuitableTypeVarName("f", names)
			continue
		}

		if metaVar.typeClasses.Contains("Foldable") {
			p.varMapping[lookupName].friendlyName = findSuitableTypeVarName("t", names)
			continue
		}

		p.varMapping[lookupName].friendlyName = findAvailableTypeVarName(names)
	}

}

func (p *Printer) CompileType(templateStr string, jobId int) string {
	classRequirements := make([]string, 0)
	for _, metaVar := range p.varMapping {
		classes := metaVar.typeClasses
		if !metaVar.appearedInJobs.Contains(jobId) {
			continue
		}
		if classes == nil {
			continue
		}
		for c := range classes.Iter() {
			classRequirements = append(classRequirements, c+" "+metaVar.tmpl)
		}
	}

	var context string

	if len(classRequirements) == 0 {
		context = ""
	} else if len(classRequirements) == 1 {
		context = classRequirements[0] + "=>"
	} else {
		slices.Sort(classRequirements)
		context = "(" + strings.Join(classRequirements, ",") + ")=>"
	}

	var varFriendlyNames = make(map[string]string)
	for lookupName, metaVar := range p.varMapping {
		varFriendlyNames[lookupName] = metaVar.friendlyName
	}

	tmpl := template.Must(template.New("").Parse(context + templateStr))
	printType := inventory.TemplateToString(tmpl, varFriendlyNames)
	return printType
}

func (p *Printer) GetType(term prolog_tool.Term) string {
	templateStr := p.PrepareType(term, 0)
	p.AssignVars()
	return p.CompileType(templateStr, 0)
}

func TestReconstruct() {
	prologStr := "pair(pair(p_Either,builtin_Int),builtin_Char)"
	prologTerm, err := prolog_tool.ParseTerm(prologStr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("prolog: %+v\n", prologTerm)
	adts := unrollADT(prologTerm)
	for _, adt := range adts {
		fmt.Printf("adt: %+v\n", adt)
	}
	printer := NewPrinter()
	s := printer.GetType(prologTerm)
	fmt.Println(s)

}
