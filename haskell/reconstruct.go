package haskell

import (
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"goanna/inventory"
	prolog_tool "goanna/prolog-tool"
	"slices"
	"strings"
	"text/template"
)

type constructorType int

const (
	list = iota
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
	classes    map[string][]string
	currentJob int
}

func NewPrinter(classes map[string][]string) *Printer {
	varmapping := make(map[string]*MetaVar)
	return &Printer{
		classes:    classes,
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
	switch typedTerm := term.(type) {
	case prolog_tool.Compound:
		if typedTerm.Value != "pair" {
			return Pair{unknown, nil, nil}
		}
		firstArg := typedTerm.Args[0]
		secondArg := typedTerm.Args[1]
		switch firstArg.(type) {
		case prolog_tool.Compound:
			return Pair{adt, firstArg, secondArg}

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
	if term.Value == "builtin_Top" {
		return "()"
	}
	parts := strings.Split(term.Value, "_")
	if len(parts) == 1 {
		return parts[0]
	} else {
		return parts[len(parts)-1]
	}
}

func adtIsTuple(term prolog_tool.Term) bool {
	switch t := term.(type) {
	case prolog_tool.Atom:
		return t.Value == "tuple"
	default:
		return false
	}
}

func isFunction(term prolog_tool.Term) bool {
	if makePair(term).conType == adt {
		args := unrollADT(term)
		switch arg0 := args[0].(type) {
		case prolog_tool.Atom:
			return arg0.Value == "function"
		default:
			return false
		}
	}
	return false
}

func unrollFunction(term prolog_tool.Term) []prolog_tool.Term {
	if isFunction(term) {
		args := unrollADT(term)
		arg1 := args[1]
		arg2 := args[2]
		return slices.Concat([]prolog_tool.Term{arg1}, []prolog_tool.Term{arg2})
	}
	return []prolog_tool.Term{term}
}

func (p *Printer) printCompound(term prolog_tool.Compound) string {
	//fmt.Printf("Term: %v\n", term)
	switch {
	case term.Value == "has":
		typeClasses := term.Args[0].(prolog_tool.List)
		var typeVar string
		var lookupString string
		switch arg1 := term.Args[1].(type) {
		case prolog_tool.Atom:
			// This is skolemized constant, we need to turn it into a type var, preferraby using the same letter
			typeVar = p.printSkolemVar(arg1)
			lookupString = arg1.Value
		case prolog_tool.Var:
			typeVar = p.printVar(arg1)
			lookupString = arg1.Value
		default:
			panic("has([..], X) where X is neither var or atom")
		}

		for _, class := range typeClasses.Values {
			className := class.(prolog_tool.Atom).Value
			p.varMapping[lookupString].typeClasses.Add(className)
		}

		return typeVar

	case makePair(term).conType == list:
		content := makePair(term).first
		return "[" + p.PrintTerm(content) + "]"

	case makePair(term).conType == adt:
		args := unrollADT(term)

		if adtIsTuple(args[0]) {
			tupleElems := args[1:]
			argsText := make([]string, len(tupleElems))
			for i, elem := range tupleElems {
				argsText[i] = p.PrintTerm(elem)
			}
			return "(" + strings.Join(argsText, ",") + ")"
		}

		if isFunction(term) {
			argStrings := make([]string, len(term.Args))
			functionArgs := unrollFunction(term)
			for i, arg := range functionArgs {
				if isFunction(arg) && i != len(term.Args)-1 {
					argStrings[i] = "(" + p.PrintTerm(arg) + ")"
				} else {
					argStrings[i] = p.PrintTerm(arg)
				}
			}

			return strings.Join(argStrings, "->")

		}

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

		if metaVar.typeClasses.Contains("p_Monad") {
			p.varMapping[lookupName].friendlyName = findSuitableTypeVarName("m", names)
			continue
		}

		if metaVar.typeClasses.Contains("p_Applicative") ||
			metaVar.typeClasses.Contains("p_Alternative") ||
			metaVar.typeClasses.Contains("p_Functor") {
			p.varMapping[lookupName].friendlyName = findSuitableTypeVarName("f", names)
			continue
		}

		if metaVar.typeClasses.Contains("p_Foldable") {
			p.varMapping[lookupName].friendlyName = findSuitableTypeVarName("t", names)
			continue
		}

		p.varMapping[lookupName].friendlyName = findAvailableTypeVarName(names)
	}
}

func normalizeContext(classes []string, superClassMap map[string][]string) []string {
	toRemove := make([]string, 0)
	for i, class := range classes {
		for j, otherClass := range classes {
			if j <= i {
				continue
			}
			if slices.Contains(superClassMap[class], otherClass) {
				toRemove = append(toRemove, otherClass)
			}
			if slices.Contains(superClassMap[otherClass], class) {
				toRemove = append(toRemove, class)
			}
		}
	}
	toKeep := make([]string, 0)
	for _, class := range classes {
		if slices.Contains(toRemove, class) {
			continue
		}
		toKeep = append(toKeep, class)
	}
	return toKeep
}

func removeModulePrefix(class string) string {
	parts := strings.Split(class, "_")
	if len(parts) == 1 {
		return parts[0]
	} else {
		return parts[len(parts)-1]
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
		reducedClasses := normalizeContext(classes.ToSlice(), p.classes)
		renamedClasses := make([]string, len(reducedClasses))
		for i, class := range reducedClasses {
			renamedClasses[i] = removeModulePrefix(class)
		}

		for _, c := range renamedClasses {
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
	printer := NewPrinter(make(map[string][]string))
	s := printer.GetType(prologTerm)
	fmt.Println(s)

}
