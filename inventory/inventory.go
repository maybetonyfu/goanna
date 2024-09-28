package inventory

import (
	"bytes"
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"mil/prolog-tool"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

var preamble = `
eq(X, Y) :- unify_with_occurs_check(X, Y).
all_equal([_]) :- true.
all_equal([X, Y|XS]) :- X=Y, all_equal([Y|XS]).

member1(L,[L|_]) :- !.
member1(L,[_|RS]) :- member1(L,RS).

test_class([with(Class, Instance)|XS]) :-
    nonvar(Class),
    !,
    call(Class, Instance),
    test_class(XS).
test_class(_).

cons(T, _, _, _, _, _) :-
    T = pair(function(A), B),
    B = pair(function(C), D),
    C = pair(list, A),
    D = pair(list, A).
`
var typeCheckTemplate = NewTemplate("type-check", `
type_check :-
    once((
        {{ joinStr . "" ",\n        " }}
    )),
    test_class(C).
`)

var typeCheckDeclTemplate = NewTemplate("type-check-decl", `
	{{-  .Name  }}(_, [], _, _, [
		{{- range $varName, $classes := .TypeVars -}}
		has([{{ joinStr $classes "" ", " }}], {{ $varName }})
		{{- end -}}
	], C)`)

var classRuleTemplate = NewTemplate("class-rule", `
{{ .Name }}(T) :-
    T = has(Class, _),
    member1({{ .Name }}, Class),
    {{ range .SuperClasses }}
    	member1({{ . }}, Class),
    {{ end }}
    true
`)

var instanceRuleTemp = NewTemplate("instance-rule", `
{{ .Name }}(T) :-
    nonvar(T),
    {{ range .Rules }}
    	{{ . }},
    {{ end }}
    {{ range .SuperClasses }}
    	{{ . }}(T),
    {{ end }}
    true
`)

var functionTemplate1 = NewTemplate("fun1", "{{ . }}(_, Calls, _, _, _, _) :- member1({{ . }}, Calls), !.")
var functionTemplate2 = NewTemplate("fun2", `
{{ .Name }}(T, Calls, Gamma, Zeta, Theta, Classes) :-
	Calls_ = [{{ .Name }} | Calls],
	Gamma = [ {{ joinInt .Captures "_" "," }} ],
	{{ if ne (len .Arguments) 0 }}
		Zeta = [{{ joinStr .Arguments "" "," }} | _],
	{{ end }}
	{{- if ne (len .TypeVars) 0 -}}
		Theta = [{{ joinStr .TypeVars (printf "_%s_" .Name) "," }}],
	{{- end }}
	{{ joinStr .RuleBody "" ",\n        "}}.`)

var mainTemplate = NewTemplate("main", `
main(G) :-
    once((
        {{ range .Declarations -}}
        	{{ . }}(_{{ . }}, [], [{{ joinInt (index $.CaptureByDecl .) "_" "," }}], _, _, C),
        {{ end -}}
        true
    )),
    test_class(C),
    G=[
    {{ range .AllCaptures -}}
          _{{.}},
    {{ end -}}
    {{- range .Declarations -}}
          _{{.}},
    {{ end -}}
    true
    ].`)

func TemplateToString(tmpl *template.Template, data interface{}) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

func joinInt(s []int, prefix, sep string) string {
	parts := make([]string, len(s))
	for i, s := range s {
		parts[i] = prefix + strconv.Itoa(s)
	}
	return strings.Join(parts, sep)
}

func joinStr(s []string, prefix, sep string) string {
	parts := make([]string, len(s))
	for i, s := range s {
		parts[i] = prefix + s
	}
	return strings.Join(parts, sep)
}

func NewTemplate(name, content string) *template.Template {
	tmpl, err := template.New(name).Funcs(
		template.FuncMap{"joinInt": joinInt, "joinStr": joinStr}).Parse(content)
	if err != nil {
		panic(err)
	}
	return tmpl
}

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
	decls := make([]string, 0)
	for _, decl := range inv.Declarations {
		s := TemplateToString(typeCheckDeclTemplate, struct {
			Name     string
			TypeVars map[string][]string
		}{decl, inv.TypeVars[decl]})
		decls = append(decls, s)
	}

	return TemplateToString(typeCheckTemplate, decls)
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
	return nil
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
	return inv.logic.ConsultAndCheck(program, "type_check")
}
