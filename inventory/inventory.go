package inventory

import (
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"mil/prolog-tool"
	"slices"
	"strings"
)

type RuleHead struct {
	Id     int    `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Module string `json:"module,omitempty"`
	Type   string `json:"type,omitempty"`
}

type Rule struct {
	Id      int      `json:"id,omitempty"`
	Head    RuleHead `json:"head"`
	Body    string   `json:"body,omitempty"`
	IsAxiom bool     `json:"is_axiom,omitempty"`
}

type NodePair struct {
	Parent int `json:"parent,omitempty"`
	Child  int `json:"child,omitempty"`
}

type Range struct {
	FromLine int `json:"from_line,omitempty"`
	ToLine   int `json:"to_line,omitempty"`
	FromCol  int `json:"from_col,omitempty"`
	ToCol    int `json:"to_col,omitempty"`
}

type Input struct {
	BaseModules  []string                       `json:"base_modules,omitempty"`
	Rules        []Rule                         `json:"rules,omitempty"`
	Declarations []string                       `json:"declarations,omitempty"`
	TypeVars     map[string]map[string][]string `json:"type_vars,omitempty"`
	Arguments    map[string][]string            `json:"arguments,omitempty"`
	NodeDepth    map[int]int                    `json:"node_depth,omitempty"`
	Classes      map[string][]string            `json:"classes,omitempty"`
	NodeTable    []NodePair                     `json:"node_graph,omitempty"`
	MaxLevel     int                            `json:"max_depth,omitempty"`
	NodeRange    map[int]Range                  `json:"node_range,omitempty"`
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
		if rule.IsAxiom || slices.Contains(inv.BaseModules, rule.Head.Module) || !leafNodes.Contains(rule.Id) {
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
		Name     string
		TypeVars map[string][]string
		IsLast   bool
	}

	context := make([]Context, 0)

	for i, decl := range inv.Declarations {
		//goal := TemplateToString(typeCheckDeclTemplate, struct {
		//	Name     string
		//	TypeVars map[string][]string
		//}{decl, inv.TypeVars[decl]})
		//decls = append(decls, )
		context = append(context, Context{
			Name:     decl,
			TypeVars: inv.TypeVars[decl],
			IsLast:   i == len(inv.Declarations)-1,
		})
	}

	return TemplateToString(typeCheckTemplate, context)
}

func (inv *Inventory) RenderMain(captures []int) string {
	slices.Sort(captures)
	captureByDecl := make(map[string][]int)
	for _, rule := range inv.Rules {
		if slices.Contains(captures, rule.Id) {
			captureByDecl[rule.Head.Name] = append(captureByDecl[rule.Head.Name], rule.Id)
		}
	}
	return TemplateToString(mainTemplate, struct {
		Declarations  []string
		CaptureByDecl map[string][]int
		AllCaptures   []int
	}{inv.Declarations, captureByDecl, captures})
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
		fmt.Printf("%s: %v\n", name, ownArguments)
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

func (inv *Inventory) RenderProlog() string {
	typingRules := inv.RenderTypingRules(inv.EffectiveRules, inv.EffectiveRules)
	classRules := inv.RenderClassRules()
	typeCheckPredicate := inv.RenderTypeChecking()
	mainPredicate := inv.RenderMain(inv.EffectiveRules)
	parts := []string{
		preamble,
		strings.Join(typingRules, "\n"),
		strings.Join(classRules, "\n"),
		typeCheckPredicate,
		mainPredicate,
	}
	return strings.Join(parts, "\n")
}

func (inv *Inventory) AxiomCheck() bool {
	typingRules := inv.RenderTypingRules(inv.AxiomaticRules, nil)
	classRules := inv.RenderClassRules()
	typeCheckPredicate := inv.RenderTypeChecking()
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
	typeCheckPredicate := inv.RenderTypeChecking()
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
	mainPredicate := inv.RenderMain(captures)
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

func (inv *Inventory) Satisfiable(rules []int) bool {
	typingRules := inv.RenderTypingRules(rules, nil)
	classRules := inv.RenderClassRules()
	typeCheckPredicate := inv.RenderTypeChecking()
	parts := []string{
		preamble,
		strings.Join(typingRules, "\n"),
		strings.Join(classRules, "\n"),
		typeCheckPredicate,
	}
	program := strings.Join(parts, "\n")
	return inv.logic.ConsultAndCheck(program, "type_check.")
}
