package marco

import (
	"github.com/irifrance/gini"
	"github.com/irifrance/gini/z"
)

type GiniSolver struct {
	solver      *gini.Gini
	vars        IntSet
	ruleIdToLit map[int]int
	litToRuleId map[int]int
}

func NewGiniSolver(vars IntSet) *GiniSolver {
	c := len(vars.ToSlice())

	ruleIdToLit := make(map[int]int)
	litToRuleId := make(map[int]int)

	for i, v := range vars.ToSlice() {
		ruleIdToLit[v] = i + 1
		litToRuleId[i+1] = v
	}

	return &GiniSolver{
		solver:      gini.NewV(c),
		vars:        vars,
		ruleIdToLit: ruleIdToLit,
		litToRuleId: litToRuleId,
	}
}

func (s *GiniSolver) Solve() bool {
	return s.solver.Solve() == 1
}

func (s *GiniSolver) Model() IntSet {
	result := NewIntSet()
	for v := range s.litToRuleId {
		if !s.solver.Value(z.Var(v).Neg()) {
			result.Add(s.litToRuleId[v])
		}
	}
	return result
}

func (s *GiniSolver) AddClause(vs IntSet) {
	for v := range vs.Iter() {
		if v < 0 {
			lit := s.ruleIdToLit[-v]
			s.solver.Add(z.Var(lit).Neg())
		} else if v > 0 {
			lit := s.ruleIdToLit[v]

			s.solver.Add(z.Var(lit).Pos())
		} else {
			panic("propositional variable cannot be zero")
		}
	}
	s.solver.Add(0)
}
