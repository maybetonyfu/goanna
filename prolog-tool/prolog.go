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

func (p *Logic) ConsultAndCheck(program string, query string) bool {
	if err := p.prolog.Exec(program); err != nil {
		fmt.Printf(program)
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
	hasSolution := solutions.Next()
	return hasSolution
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
	program := "x(X) :- X = f(a)."
	query := "x(f(X))."
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
		fmt.Printf("%s\n", s)
	}
}
