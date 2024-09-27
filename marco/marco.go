package marco

import (
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
)

type IntSet mapset.Set[int]

func NewIntSet(vals ...int) IntSet {
	return IntSet(mapset.NewSet[int](vals...))
}

type Marco struct {
	Rules       IntSet
	MUSs        []IntSet
	MCSs        []IntSet
	MSSs        []IntSet
	MaxLoop     int
	LoopCounter int
	SatFunc     func([]int) bool
	Solver      Solver
}

func NewMarco(rules []int, satFunc func([]int) bool) *Marco {
	marco := Marco{
		Rules:       mapset.NewSet[int](rules...),
		MUSs:        []IntSet{},
		MCSs:        []IntSet{},
		MSSs:        []IntSet{},
		MaxLoop:     10,
		LoopCounter: 0,
		SatFunc:     satFunc,
		Solver:      NewGiniSolver(NewIntSet(rules...)),
	}
	return &marco
}

func (m *Marco) Grow(seed IntSet) IntSet {
	for elem := range (m.Rules.Difference(seed)).Iter() {
		newSet := seed.Clone()
		newSet.Add(elem)
		if m.Sat(newSet) {
			seed.Add(elem)
		}
	}
	return seed
}

func (m *Marco) Shrink(seed IntSet) IntSet {
	newSeed := seed.Clone()
	for elem := range newSeed.Iter() {
		newSet := seed.Difference(NewIntSet(elem))
		if !m.Sat(newSet) {
			seed.Remove(elem)
		}
	}
	return seed
}

func (m *Marco) Sat(rules IntSet) bool {
	return m.SatFunc(rules.ToSlice())
}

func (m *Marco) Run() {
	successful := m.Solver.Solve()
	for successful {
		//println("Loop Number", m.LoopCounter)
		if m.LoopCounter >= m.MaxLoop {
			panic("Too many loops")
		}

		seed := m.Solver.Model()
		//fmt.Printf("Seed: %d\n", seed.ToSlice())

		if m.Sat(seed) {
			mss := m.Grow(seed)
			m.MSSs = append(m.MSSs, mss)
			//fmt.Printf("Found MSS: %s \n", mss)

			mcs := m.Rules.Difference(mss)
			//fmt.Printf("Add Clause: %s \n", mcs)
			m.Solver.AddClause(mcs)
		} else {
			//fmt.Println("Unsat")
			mus := m.Shrink(seed)
			m.MUSs = append(m.MUSs, mus)
			//fmt.Printf("Found MUS: %s \n", mus)
			var negs IntSet = NewIntSet()
			for v := range mus.Iter() {
				negs.Add(-v)
			}
			//fmt.Printf("Add Clause: %s \n", negs)
			m.Solver.AddClause(negs)
		}
		successful = m.Solver.Solve()
		m.LoopCounter = m.LoopCounter + 1
		//fmt.Println("Success: ", successful)
	}
}

func TestMarco() {
	satFunc := func(rules []int) bool {
		solver := NewGiniSolver(NewIntSet(1, 2))
		allProls := [][]int{
			{1},
			{-1},
			{2},
			{-2},
			{1, 2},
		}
		for _, rule := range rules {
			solver.AddClause(NewIntSet(allProls[rule-1]...))
		}
		return solver.Solve()
	}
	mc := NewMarco([]int{1, 2, 3, 4, 5}, satFunc)
	mc.Run()
	for _, mus := range mc.MUSs {
		fmt.Println("MUS: ", mus)
	}

	for _, mss := range mc.MSSs {
		fmt.Println("MSS: ", mss)
	}
}
