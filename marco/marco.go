package marco

import (
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"mil/graph"
)

type IntSet mapset.Set[int]

type Cause struct {
	MCS  IntSet
	MSS  IntSet
	Type any
}

type Error struct {
	Causes        []Cause
	CriticalNodes []int
}

func NewIntSet(vals ...int) IntSet {
	return IntSet(mapset.NewSet[int](vals...))
}

type Marco struct {
	Rules        IntSet
	MUSs         []IntSet
	MCSs         []IntSet
	MSSs         []IntSet
	MaxLoop      int
	LoopCounter  int
	SatFunc      func([]int) bool
	Solver       Solver
	singletonMUS IntSet
}

func NewMarco(rules []int, satFunc func([]int) bool) *Marco {
	marco := Marco{
		Rules:        mapset.NewSet[int](rules...),
		MUSs:         []IntSet{},
		MCSs:         []IntSet{},
		MSSs:         []IntSet{},
		MaxLoop:      1000,
		LoopCounter:  0,
		SatFunc:      satFunc,
		Solver:       NewMaxsatSolver(NewIntSet(rules...)),
		singletonMUS: NewIntSet(),
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
		if m.singletonMUS.Contains(elem) {
			continue
		}
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
			mss := seed
			m.MSSs = append(m.MSSs, mss)
			//fmt.Printf("Found MSS: %s \n", mss)

			mcs := m.Rules.Difference(mss)
			mcsSlice := mcs.ToSlice()
			if len(mcsSlice) == 1 {
				m.singletonMUS.Add(mcsSlice[0])
			}
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

func combinations(input []int) [][]int {
	var results [][]int
	for i := 0; i < len(input); i++ {
		for j := i + 1; j < len(input); j++ {
			results = append(results, []int{input[i], input[j]})
		}
	}
	return results
}

func (m *Marco) Analysis() []Error {
	// Populate MCS List
	for _, mss := range m.MSSs {
		m.MCSs = append(m.MCSs, m.Rules.Difference(mss))
	}

	musIndexList := make([]int, len(m.MUSs))
	for i := range musIndexList {
		musIndexList[i] = i
	}
	//fmt.Printf("%v\n", m.MUSs)
	musGraph := graph.NewGraph(len(musIndexList))
	for _, combination := range combinations(musIndexList) {
		index1 := combination[0]
		mus1 := m.MUSs[index1]

		index2 := combination[1]
		mus2 := m.MUSs[index2]

		if !mus1.Intersect(mus2).IsEmpty() {
			musGraph.AddEdge(index1, index2)
		}
	}

	_, components := musGraph.CountAndGetConnectedComponents()
	//fmt.Printf("Components: \n %v\n", components)

	errors := make([]Error, 0)
	for _, component := range components {
		musList := make([]IntSet, 0)
		mcsList := make([]IntSet, 0)
		for _, musId := range component {
			musList = append(musList, m.MUSs[musId])
		}
		criticalNodes := NewIntSet()
		for _, mus := range musList {
			criticalNodes = criticalNodes.Union(mus)
		}
		for _, mcs := range m.MCSs {
			reduced := mcs.Intersect(criticalNodes)
			if reduced.IsEmpty() {
				continue
			}
			exist := false
			for _, included := range mcsList {
				if reduced.Equal(included) {
					exist = true
					break
				}
			}
			if !exist {
				mcsList = append(mcsList, reduced)
			}
		}
		causes := make([]Cause, len(mcsList))

		for i, mcs := range mcsList {
			causes[i] = Cause{mcs, criticalNodes.Difference(mcs), nil}
		}

		errors = append(errors, Error{
			Causes:        causes,
			CriticalNodes: criticalNodes.ToSlice(),
		})
	}
	return errors
}

func TestMarco() {
	satFunc := func(rules []int) bool {
		solver := NewMaxsatSolver(NewIntSet(1, 2))
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
