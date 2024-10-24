package prolog_tool

import (
	"fmt"
	"github.com/ichiban/prolog"
)

type Logic struct {
	prolog *prolog.Interpreter
}

func NewProlog() *Logic {
	return &Logic{
		prolog: prolog.New(nil, nil),
	}
}

func (p *Logic) Consult(program string) {
	if err := p.prolog.Exec(program); err != nil {
		fmt.Println(program)
		panic(err)
	}
}

func (p *Logic) Query(query string) bool {
	solutions, err := p.prolog.Query(query)
	defer func() {
		if err := solutions.Close(); err != nil {
			panic(err)
		}
	}()
	if err != nil {
		panic(err)
	}
	hasSolution := solutions.Next()
	return hasSolution
}

func (p *Logic) ConsultAndCheck(program string, query string) bool {
	p.Consult(program)
	return p.Query(query)
}

func (p *Logic) Abolish(name string, n int) {
	retract := fmt.Sprintf("abolish(%s/%d).", name, n)
	fmt.Println(retract)
	if err := p.prolog.QuerySolution(retract).Err(); err != nil {
		panic(err)
	}
}

func (p *Logic) Assertz(clause string) {
	//fmt.Println(clause)
	if err := p.prolog.QuerySolution(fmt.Sprintf("assertz((%s)).", clause)).Err(); err != nil {
		panic(err)
	}
}

func (p *Logic) Query1(query string) (bool, map[string]string) {
	solutions, err := p.prolog.Query(query)
	defer func() {
		if err := solutions.Close(); err != nil {
			panic(err)
		}
	}()
	if err != nil {
		panic(err)
	}
	if solutions.Next() {
		var s = make(map[string]prolog.TermString)
		if err := solutions.Scan(&s); err != nil {
			panic(err)
		}
		var result = make(map[string]string)
		for k, v := range s {
			result[k] = string(v)
		}
		return true, result
	}
	return false, nil
}

func (p *Logic) ConsultAndQuery1(program string, query string) (bool, map[string]string) {
	if err := p.prolog.Exec(program); err != nil {
		panic(err)
	}
	solutions, err := p.prolog.Query(query)
	defer func() {
		if err := solutions.Close(); err != nil {
			panic(err)
		}
	}()
	if err != nil {
		panic(err)
	}
	if solutions.Next() {
		var s = make(map[string]prolog.TermString)
		if err := solutions.Scan(&s); err != nil {
			panic(err)
		}
		var result = make(map[string]string)
		for k, v := range s {
			result[k] = string(v)
		}
		return true, result
	}
	return false, nil
}

func TestProlog() {
	p := NewProlog()
	program := `
:- dynamic(m0_x/6).

eq(X, Y) :- unify_with_occurs_check(X, Y).
all_equal([_]) :- true.
all_equal([X, Y|XS]) :- eq(X, Y), all_equal([Y|XS]).

member1(L,[L|_]) :- !.
member1(L,[_|RS]) :- member1(L,RS).

test_class([with(Class, Instance)|XS]) :-
    nonvar(Class), !,
    call(Class, Instance),
    test_class(XS).
test_class(_).

cons(T, _, _, _, _, _) :-
    T = pair(function(A), B),
    B = pair(function(C), D),
    C = pair(list, A),
    D = pair(list, A).

m0_x(_, Calls, _, _, _, _) :- member1(m0_x, Calls), !.
m0_x(T, Calls, Gamma, Zeta, _, Classes) :-
    Calls_ = [m0_x | Calls],
    eq(T, _2),                                                                                    
    eq(T, _5),                                                                                    
    eq(_5, _4),                                                                                    
    true.


type_check :-
    once((
        m0_x(_, [], _, _, [], C_m0_x)
        )),
    test_class(C_m0_x),
    true.

`
	p.Consult(program)

	//p1 := `test_assert(X) :- X = yes`

	p.Abolish("m0_x", 6)
	_, r := p.Query1("type_check.")
	fmt.Println(r)

	//p.Assertz(p1)
	//p.Assertz(p2)
	//r1 := p.Query("type_check.")
	//b, r := p.Query1("test_assert(G).")
	//if b {
	//	fmt.Println(r)
	//
	//}
	//
	//p3 := `m0_x(_, Calls, _, _, _, _) :- member1(m0_x, Calls), !`
	//p4 := `m0_x(T, Calls, Gamma, Zeta, _, Classes) :-
	//	Calls_ = [m0_x | Calls],
	//eq(T, _2),
	//eq(_2, builtin_Char),
	//eq(T, _5),
	//eq(_5, _4),
	//true`
	//p.Abolish("m0_x")
	//p.Assertz(p3)
	//p.Assertz(p4)
	//r2 := p.Query("type_check.")
	//fmt.Println(r2)
}
