package inventory

import (
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"mil/prolog-tool"
	"slices"
	"strings"
)

type VarClass struct {
	VarName string
	Classes []string
	IsLast  bool
}

type RuleHead struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Module string `json:"module"`
	Type   string `json:"type"`
}

type Rule struct {
	Id      int      `json:"id"`
	Head    RuleHead `json:"head"`
	Body    string   `json:"body"`
	IsAxiom bool     `json:"is_axiom"`
}

type NodePair struct {
	Parent int `json:"parent"`
	Child  int `json:"child"`
}

type Range struct {
	FromLine int `json:"from_line"`
	ToLine   int `json:"to_line"`
	FromCol  int `json:"from_col"`
	ToCol    int `json:"to_col"`
}

type Identifier struct {
	NodeId    int    `json:"node_id"`
	Name      string `json:"name"`
	NodeRange Range  `json:"node_range"`
	IsType    bool   `json:"is_type"`
	IsTerm    bool   `json:"is_term"`
}

type Input struct {
	BaseModules   []string                       `json:"base_modules"`
	ParsingErrors []Range                        `json:"parsing_errors"`
	ImportErrors  []Identifier                   `json:"import_errors"`
	Rules         []Rule                         `json:"rules"`
	Declarations  []string                       `json:"declarations"`
	TypeVars      map[string]map[string][]string `json:"type_vars"`
	Arguments     map[string][]string            `json:"arguments"`
	NodeDepth     map[int]int                    `json:"node_depth"`
	Classes       map[string][]string            `json:"classes"`
	NodeTable     []NodePair                     `json:"node_graph"`
	MaxLevel      int                            `json:"max_depth"`
	NodeRange     map[int]Range                  `json:"node_range,"`
}

type Inventory struct {
	Input
	AxiomaticRules []int
	EffectiveRules []int
	CurrentLevel   int
	TypingRules    map[string][]Rule
	InstanceRules  map[string]map[int][]string
	logic          *prolog_tool.Logic
}

func (inv *Inventory) getVarClasses() map[string][]VarClass {
	result := make(map[string][]VarClass)
	for _, decl := range inv.Declarations {
		varClasses := make([]VarClass, 0)
		for varName, classes := range inv.TypeVars[decl] {
			varClasses = append(varClasses, VarClass{varName + "__" + decl, classes, false})
		}

		slices.SortFunc(varClasses, func(a, b VarClass) int {
			if a.VarName > b.VarName {
				return 1
			} else if a.VarName < b.VarName {
				return -1
			} else {
				return 0
			}
		})
		if len(varClasses) > 0 {
			varClasses[len(varClasses)-1].IsLast = true
		}
		result[decl] = varClasses
	}
	return result
}

func NewInventory(input Input) *Inventory {
	tyingRules := make(map[string][]Rule)
	for _, rule := range input.Rules {
		if rule.Head.Type == "type" {
			tyingRules[rule.Head.Name] = append(tyingRules[rule.Head.Name], rule)
		}
	}

	instanceRules := make(map[string]map[int][]string)
	for _, rule := range input.Rules {
		if rule.Head.Type == "instance" {
			if instanceRules[rule.Head.Name] == nil {
				instanceRules[rule.Head.Name] = make(map[int][]string)
			}
			instanceRules[rule.Head.Name][rule.Head.Id] = append(instanceRules[rule.Head.Name][rule.Head.Id], rule.Body)
		}
	}
	return &Inventory{
		Input:          input,
		AxiomaticRules: make([]int, 0),
		EffectiveRules: make([]int, 0),
		CurrentLevel:   input.MaxLevel,
		TypingRules:    tyingRules,
		InstanceRules:  instanceRules,
		logic:          prolog_tool.NewProlog(),
	}
}

func (inv *Inventory) Generalize(currentLevel int) {
	fmt.Printf("Current Level: %d\n", currentLevel)
	parents := mapset.NewSet[int]()
	nodes := mapset.NewSet[int]()
	for _, pair := range inv.NodeTable {
		child := pair.Child
		parent := pair.Parent
		if inv.NodeDepth[child] <= currentLevel {
			parents.Add(parent)
			nodes.Add(parent)
			nodes.Add(child)
		}
	}

	leafNodes := nodes.Difference(parents)
	axiomRules := make([]int, 0)
	effectiveRules := make([]int, 0)
	for _, rule := range inv.Rules {
		fromOtherModule := slices.Contains(inv.BaseModules, rule.Head.Module)
		//isOrphanNode := inv.NodeDepth[rule.Id] > currentLevel
		//if isOrphanNode && !fromOtherModule {
		//	fmt.Println("Orphan node detected, ", rule.Id)
		//	continue
		//}
		isLeafNode := leafNodes.Contains(rule.Id)
		if rule.IsAxiom ||
			fromOtherModule ||
			!isLeafNode {
			axiomRules = append(axiomRules, rule.Id)
		} else {
			effectiveRules = append(effectiveRules, rule.Id)
		}
	}
	slices.Sort(effectiveRules)
	inv.AxiomaticRules = axiomRules
	inv.EffectiveRules = effectiveRules
}

func (inv *Inventory) RenderTypeChecking() string {
	type Context struct {
		Name       string
		VarClasses []VarClass
		IsLast     bool
	}
	varClasses := inv.getVarClasses()
	context := make([]Context, 0)
	for _, decl := range inv.Declarations {
		if strings.HasPrefix(decl, "p_") {
			continue
		}
		context = append(context, Context{
			Name:       decl,
			VarClasses: varClasses[decl],
			IsLast:     false,
		})
	}

	for i, _ := range context {
		if i == len(context)-1 {
			context[i].IsLast = true
		}
	}

	return TemplateToString(typeCheckTemplate, context)
}

func (inv *Inventory) RenderMain(captures []int) string {
	slices.Sort(captures)
	captureByDecl := make(map[string][]int)
	ownDecals := make([]string, 0)
	for _, decl := range inv.Declarations {
		if !strings.HasPrefix(decl, "p_") {
			ownDecals = append(ownDecals, decl)
		}
	}
	for _, rule := range inv.Rules {
		if slices.Contains(captures, rule.Id) {
			captureByDecl[rule.Head.Name] = append(captureByDecl[rule.Head.Name], rule.Id)
		}
	}
	typeVarsByDecl := make(map[string][]string)
	for decl, tvs := range inv.TypeVars {
		declVars := make([]string, 0)
		for varName, _ := range tvs {
			declVars = append(declVars, varName)
		}
		typeVarsByDecl[decl] = declVars
	}

	return TemplateToString(mainTemplate, struct {
		Declarations   []string
		CaptureByDecl  map[string][]int
		TypeVarsByDecl map[string][]VarClass
		AllCaptures    []int
	}{ownDecals, captureByDecl, inv.getVarClasses(), captures})
}

func (inv *Inventory) RenderClassRules() []string {
	classRules := make([]string, 0)
	for className, superClasses := range inv.Classes {
		rule1 := TemplateToString(classRuleTemplate, struct {
			Name         string
			SuperClasses []string
		}{className, superClasses})
		classRules = append(classRules, rule1)
		for _, rules := range inv.InstanceRules[className] {
			r := TemplateToString(instanceRuleTemp, struct {
				Name         string
				SuperClasses []string
				Rules        []string
			}{
				className, superClasses, rules,
			})
			classRules = append(classRules, r)
		}
	}
	return classRules
}

func (inv *Inventory) RenderTypingRules(rules, captures []int) []string {
	var result []string
	for _, name := range inv.Declarations {
		ownTypingRule := inv.TypingRules[name]
		ownTypingRuleBody := make([]string, 0)
		capturedNodes := make([]int, 0)
		ownArguments := inv.Arguments[name]
		owenTypeVars := make([]string, 0)
		for varName := range inv.TypeVars[name] {
			owenTypeVars = append(owenTypeVars, varName)
		}
		for _, rule := range ownTypingRule {
			if slices.Contains(rules, rule.Id) || slices.Contains(inv.AxiomaticRules, rule.Id) {
				ownTypingRuleBody = append(ownTypingRuleBody, rule.Body)
			}
			if slices.Contains(captures, rule.Id) {
				capturedNodes = append(capturedNodes, rule.Id)
			}
		}
		slices.Sort(owenTypeVars)
		result = append(result, TemplateToString(functionTemplate1, name))
		result = append(result, TemplateToString(functionTemplate2,
			struct {
				Name      string
				Arguments []string
				Captures  []int
				RuleBody  []string
				TypeVars  []string
			}{name, ownArguments, capturedNodes, ownTypingRuleBody, owenTypeVars}))
	}
	return result
}

func (inv *Inventory) renderChangedTypingRules(names []string, rules []int) []string {
	var result []string
	for _, name := range names {
		ownTypingRule := inv.TypingRules[name]
		ownTypingRuleBody := make([]string, 0)
		ownArguments := inv.Arguments[name]
		owenTypeVars := make([]string, 0)
		for varName := range inv.TypeVars[name] {
			owenTypeVars = append(owenTypeVars, varName)
		}
		for _, rule := range ownTypingRule {
			if slices.Contains(rules, rule.Id) || slices.Contains(inv.AxiomaticRules, rule.Id) {
				ownTypingRuleBody = append(ownTypingRuleBody, rule.Body)
			}
		}
		slices.Sort(owenTypeVars)
		result = append(result, TemplateToString(functionTemplate1, name))
		result = append(result, TemplateToString(functionTemplate2,
			struct {
				Name      string
				Arguments []string
				Captures  []int
				RuleBody  []string
				TypeVars  []string
			}{name, ownArguments, []int{}, ownTypingRuleBody, owenTypeVars}))
	}
	return result
}

func (inv *Inventory) findDeclarationsByRules(rules []int) []string {
	var decls = mapset.NewSet[string]()
	for _, rule := range inv.Rules {
		if rule.Head.Type == "instance" {
			continue
		}
		if slices.Contains(rules, rule.Id) {
			decls.Add(rule.Head.Name)
		}
	}
	return decls.ToSlice()
}

func (inv *Inventory) RenderProlog() string {
	typingRules := inv.RenderTypingRules(inv.EffectiveRules, inv.EffectiveRules)
	classRules := inv.RenderClassRules()
	typeCheckPredicate := terminateClause(inv.RenderTypeChecking())
	mainPredicate := terminateClause(inv.RenderMain(inv.EffectiveRules))
	terminateClauses(classRules)
	terminateClauses(typingRules)
	parts := []string{
		preamble,
		strings.Join(typingRules, "\n"),
		strings.Join(classRules, "\n"),
		typeCheckPredicate,
		mainPredicate,
	}
	return strings.Join(parts, "\n")
}

func terminateClause(clause string) string {
	return clause + "."
}

func terminateClauses(clauses []string) {
	for i, clause := range clauses {
		clauses[i] = terminateClause(clause)
	}
}

func (inv *Inventory) AxiomCheck() bool {
	typingRules := inv.RenderTypingRules(inv.AxiomaticRules, nil)
	classRules := inv.RenderClassRules()
	typeCheckPredicate := terminateClause(inv.RenderTypeChecking())
	terminateClauses(classRules)
	terminateClauses(typingRules)
	parts := []string{
		preamble,
		strings.Join(typingRules, "\n"),
		strings.Join(classRules, "\n"),
		typeCheckPredicate,
	}
	program := strings.Join(parts, "\n")
	return inv.logic.ConsultAndCheck(program, "type_check.")
}

func (inv *Inventory) TypeCheck() bool {
	typingRules := inv.RenderTypingRules(inv.EffectiveRules, nil)
	classRules := inv.RenderClassRules()
	typeCheckPredicate := terminateClause(inv.RenderTypeChecking())
	terminateClauses(classRules)
	terminateClauses(typingRules)

	parts := []string{
		preamble,
		strings.Join(typingRules, "\n"),
		strings.Join(classRules, "\n"),
		typeCheckPredicate,
	}
	program := strings.Join(parts, "\n")
	return inv.logic.ConsultAndCheck(program, "type_check.")
}

func (inv *Inventory) QueryTypes(rules, captures []int) map[string]string {
	typingRules := inv.RenderTypingRules(rules, captures)
	classRules := inv.RenderClassRules()
	mainPredicate := terminateClause(inv.RenderMain(captures))
	terminateClauses(classRules)
	terminateClauses(typingRules)
	parts := []string{
		preamble,
		strings.Join(typingRules, "\n"),
		strings.Join(classRules, "\n"),
		mainPredicate,
	}
	program := strings.Join(parts, "\n")
	succeed, result := inv.logic.ConsultAndQuery1(program, "main(G, L).")
	if !succeed {
		panic("Provided MSS is unsatisfiable")
	}
	return result
}

func (inv *Inventory) ConsultAxioms() {
	names := inv.findDeclarationsByRules(inv.EffectiveRules)
	directives := make([]string, len(names))
	for i, name := range names {
		directives[i] = ":- dynamic(" + name + "/6)"
	}
	typingRules := inv.RenderTypingRules(nil, nil)
	classRules := inv.RenderClassRules()

	typeCheckPredicate := terminateClause(inv.RenderTypeChecking())
	terminateClauses(classRules)
	terminateClauses(typingRules)
	terminateClauses(directives)
	parts := []string{
		strings.Join(directives, "\n"),
		preamble,
		strings.Join(typingRules, "\n"),
		strings.Join(classRules, "\n"),
		typeCheckPredicate,
	}
	text := strings.Join(parts, "\n")
	inv.logic.Consult(text)
}

func (inv *Inventory) Satisfiable(rules []int) bool {
	changedNames := inv.findDeclarationsByRules(rules)
	typingRules := inv.renderChangedTypingRules(changedNames, rules)

	for _, name := range changedNames {
		inv.logic.Abolish(name, 6)
	}

	for _, rule := range typingRules {
		inv.logic.Assertz(rule)
	}

	return inv.logic.Query("type_check.")
}
