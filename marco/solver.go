package marco

import (
	"fmt"
)

type Solver interface {
	Solve() bool
	Model() IntSet
	AddClause(IntSet)
}

func TestSolver() {
	s := NewMaxsatSolver(NewIntSet(1, 2, 3, 4))
	s.AddClause(NewIntSet(-2, -4))
	//s.AddClause(NewIntSet(1, 2))
	s.AddClause(NewIntSet(4))

	if s.Solve() {
		m := s.Model()
		fmt.Printf("%+v\n", m)
	} else {
		fmt.Println("unast")
	}

}
