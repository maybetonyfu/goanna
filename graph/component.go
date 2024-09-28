package graph

import (
	"fmt"
)

type Graph struct {
	adj [][]int
}

func NewGraph(n int) *Graph {
	return &Graph{make([][]int, n)}
}

func (g *Graph) AddEdge(u, v int) {
	g.adj[u] = append(g.adj[u], v)
	g.adj[v] = append(g.adj[v], u)
}

func (g *Graph) CountAndGetConnectedComponents() (int, map[int][]int) {
	n := len(g.adj)
	visited := make([]bool, n)
	componentMap := make(map[int][]int)

	var dfs func(int, int)
	dfs = func(v, component int) {
		visited[v] = true
		componentMap[component] = append(componentMap[component], v)
		for _, w := range g.adj[v] {
			if !visited[w] {
				dfs(w, component)
			}
		}
	}

	count := 0
	for i := 0; i < n; i++ {
		if !visited[i] {
			count++
			dfs(i, count)
		}
	}

	return count, componentMap
}

func TestGraph() {
	g := NewGraph(5)
	g.AddEdge(0, 1)
	g.AddEdge(1, 2)
	g.AddEdge(4, 3)
	g.AddEdge(3, 4)

	count, components := g.CountAndGetConnectedComponents()

	fmt.Printf("Number of disconnected components: %d\n", count)
	fmt.Println("Nodes in each component:")
	for componentID, nodes := range components {
		fmt.Printf("Component %d: %v\n", componentID, nodes)
	}
}
