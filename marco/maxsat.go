package marco

import (
	"github.com/crillab/gophersat/maxsat"
	"strconv"
)

type MaxSatSolver struct {
	clauses []maxsat.Constr
	vars    IntSet
	model   map[string]bool
}

func NewMaxsatSolver(vars IntSet) *MaxSatSolver {
	softClauses := make([]maxsat.Constr, len(vars.ToSlice()))
	for i, v := range vars.ToSlice() {
		softClauses[i] = maxsat.SoftClause(maxsat.Var(strconv.Itoa(v)))
	}

	return &MaxSatSolver{
		clauses: softClauses,
		vars:    vars,
		model:   make(map[string]bool),
	}
}

func (s *MaxSatSolver) Solve() bool {
	pb := maxsat.New(s.clauses...)
	model, _ := pb.Solve()
	s.model = model
	return model != nil
}

func (s *MaxSatSolver) Model() IntSet {
	model := NewIntSet()
	for v := range s.vars.Iter() {
		vStr := strconv.Itoa(v)
		if s.model[vStr] {
			model.Add(v)
		}
	}
	return model
}

func (s *MaxSatSolver) AddClause(vars IntSet) {
	clauses := make([]maxsat.Lit, len(vars.ToSlice()))
	for i, v := range vars.ToSlice() {
		if v > 0 {
			vStr := strconv.Itoa(v)
			clauses[i] = maxsat.Var(vStr)
		} else {
			vStr := strconv.Itoa(-v)
			clauses[i] = maxsat.Var(vStr).Negation()
		}
	}
	constr := maxsat.HardClause(clauses...)
	s.clauses = append(s.clauses, constr)
}
