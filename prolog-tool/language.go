package prolog_tool

import (
	"fmt"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type Term interface {
	term()
}

type Var struct {
	Value string `@Var`
}

type Atom struct {
	Value string `@Atom`
}

type Compound struct {
	Value string `@Atom`
	Args  []Term `"(" @@ ( "," @@)*  ")"`
}

type List struct {
	Values []Term `"[" (@@ ( "," @@)*  ("|" Var)?)? "]"`
}

type Formula struct {
	Formula Term `@@`
}

func (Var) term()      {}
func (Atom) term()     {}
func (List) term()     {}
func (Compound) term() {}

var termLexer = lexer.MustSimple([]lexer.SimpleRule{
	{"Atom", `[a-z]+[a-zA-Z_0-9]*`},
	{"Var", `[A-Z_][a-zA-Z_0-9]*`},
	{Name: "Punct", Pattern: `[-[!@#$%^&*()+={}\|:;"'<,>.?/]|]`},
})

var termParser = participle.MustBuild[Formula](
	participle.Union[Term](Compound{}, Var{}, Atom{}, List{}),
	participle.Lexer(termLexer))

func ParseTerm(s string) (Term, error) {
	g, e := termParser.ParseString("", s)
	return g.Formula, e
}

func TestParser() {
	termParser := participle.MustBuild[Formula](
		participle.Union[Term](Compound{}, Var{}, Atom{}, List{}),
		participle.Lexer(termLexer))

	g, e := termParser.ParseString("Test.file", "_100")
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Printf("%#v\n", g)

	g, e = termParser.ParseString("Test.file", "gello")
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Printf("%#v\n", g)

	g, e = termParser.ParseString("Test.file", "hello(b,c,d)")
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Printf("%#v\n", g)
	g, e = termParser.ParseString("Test.file", "[a,b,c(d,f(g)),d]")
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Printf("%#v\n", g)

	g, e = termParser.ParseString("Test.file", "[]")
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Printf("%#v\n", len((g.Formula).(List).Values))
}
