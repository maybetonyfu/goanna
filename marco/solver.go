package marco

import (
	"github.com/irifrance/gini"
	"github.com/irifrance/gini/z"
)

type Solver interface {
	Solve() bool
	Model() IntSet
	AddClause(IntSet)
}

type GiniSolver struct {
	solver *gini.Gini
	vars   IntSet
}

func NewGiniSolver(vars IntSet) *GiniSolver {
	return &GiniSolver{
		solver: gini.New(),
		vars:   vars,
	}
}

func (s *GiniSolver) Solve() bool {
	return s.solver.Solve() == 1
}

func (s *GiniSolver) Model() IntSet {
	result := NewIntSet()
	for v := range s.vars.Iter() {
		if !s.solver.Value(z.Var(v).Neg()) {
			result.Add(v)
		}
	}
	return result
}

func (s *GiniSolver) AddClause(vs IntSet) {
	for v := range vs.Iter() {
		if v < 0 {
			s.solver.Add(z.Var(-v).Neg())
		} else if v > 0 {
			s.solver.Add(z.Var(v).Pos())
		} else {
			panic("propositional variable cannot be zero")
		}
	}
	s.solver.Add(0)
}
