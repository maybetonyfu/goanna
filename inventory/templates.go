package inventory

var preamble = `
eq(X, Y) :- unify_with_occurs_check(X, Y).
all_equal([_]) :- true.
all_equal([X, Y|XS]) :- eq(X, Y), all_equal([Y|XS]).

appendAll([], []).
appendAll([X|Y],Z) :- append(X, U, Z), appendAll(Y, U).

member1(L,[L|_]) :- !.
member1(L,[_|RS]) :- member1(L,RS).

test_class([with(Class, Instance)|XS]) :-
		nonvar(Class), !,
		call(Class, Instance),
		test_class(XS).
test_class(_).

builtin_cons(T, _, _, _, _, _) :-
		T = pair(pair(function, A), pair(pair(function, AS), AS)),
		AS = pair(list, A).

`
var typeCheckTemplate = NewTemplate("type-check", `
type_check :-
	{{ range . -}}
		{{-  .Name  }}(_, [], _, _, [
			{{- range .VarClasses -}}
				has([{{ joinStr .Classes "" ", " }}], {{ .VarName }}){{ if not .IsLast }},{{ end }}
			{{- end -}}], C_{{.Name}}),
	{{ end -}}
		{{- range . }}
		test_class(C_{{ .Name }}),
		{{- end }}
		true`)

var classRuleTemplate = NewTemplate("class-rule", `
{{ .Name }}(T) :-
		T = has(Class, _), !,
		member1({{ .Name }}, Class),
		{{ range .SuperClasses -}}
				member1({{ . }}, Class),
		{{ end -}}
		true`)

var instanceRuleTemp = NewTemplate("instance-rule", `
{{ .Name }}(T) :-
		nonvar(T),
		{{ range .Rules -}}
				{{ . }},
		{{ end -}}
		{{ range .SuperClasses -}}
				{{ . }}(T),
		{{ end -}}
		true`)

var functionTemplate1 = NewTemplate("fun1", "{{ . }}(_, Calls, _, _, _, _) :- member1({{ . }}, Calls), !")
var functionTemplate2 = NewTemplate("fun2", `
{{- .Name }}(T, Calls, Gamma, Zeta, Theta, Classes) :-
		Calls_ = [{{ .Name }} | Calls],
		{{ if ne (len .Captures) 0 -}}
		Gamma = [ {{ joinInt .Captures "_" "," }} ],
		{{ end -}}
		{{ if ne (len .Arguments) 0 -}}
		Zeta = [{{ joinStr .Arguments "_" "," }} ],
		{{ end -}}
		{{- if ne (len .TypeVars) 0 -}}
	Theta = [{{ joinStr .TypeVars (printf "_%s_" .Name) "," }}],
		{{ end -}}
		{{ range .RuleBody   -}}
		{{ . }},
		{{ end -}}
	once(appendAll([{{ joinStr .CollectorVars "" ","}}], Classes))`)

var mainTemplate = NewTemplate("main", `
main(G, L) :-
		{{- range .Declarations }}
		{{ . }}(_{{- . -}}, [], [
		{{- joinInt (index $.CaptureByDecl .) "_" "," -}}], _, [
		{{- range (index $.TypeVarsByDecl .) -}}
				has([{{ joinStr .Classes "" ", " }}], {{ .VarName }}){{ if not .IsLast }},{{ end }}
		{{- end -}}
		], C_{{.}}),
		{{- end }}
		{{- range .Declarations }}
		test_class(C_{{.}}),
		{{- end }}
		L=[ {{ joinInt .AllCaptures "_" "," }} ],
		G=[ {{ joinStr .Declarations "_" "," }} ]`)
