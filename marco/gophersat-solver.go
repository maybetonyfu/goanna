package marco

import (
	"github.com/crillab/gophersat/solver"
)

type GopherSolver struct {
	solver      *solver.Solver
	ruleIdToLit map[int]int
	litToRuleId map[int]int
}

func NewGopherSolver(vars IntSet) *GopherSolver {
	ruleIdToLit := make(map[int]int)
	litToRuleId := make(map[int]int)

	for i, v := range vars.ToSlice() {
		ruleIdToLit[v] = i + 1
		litToRuleId[i+1] = v
	}

	clauses := [][]int{}
	for v := range litToRuleId {
		clauses = append(clauses, []int{v, -v})
	}
	pb := solver.ParseSlice(clauses)
	return &GopherSolver{
		solver:      solver.New(pb),
		ruleIdToLit: ruleIdToLit,
		litToRuleId: litToRuleId,
	}
}

func (s *GopherSolver) Solve() bool {
	return s.solver.Solve() == solver.Sat
}

func (s *GopherSolver) Model() IntSet {
	m := s.solver.Model()
	model := NewIntSet()
	for i, b := range m {
		if !b {
			ruleId := s.litToRuleId[i+1]
			model.Add(ruleId)
		}
	}
	return model
}

func (s *GopherSolver) AddClause(vars IntSet) {
	lits := make([]solver.Lit, 0)
	for v := range vars.Iter() {
		if v > 0 {
			lit := int32(s.ruleIdToLit[v])
			lits = append(lits, solver.IntToLit(lit).Negation())
		} else {
			lit := int32(s.ruleIdToLit[-v])
			lits = append(lits, solver.IntToLit(lit))
		}
	}
	clause := solver.NewClause(lits)
	s.solver.AppendClause(clause)
}
