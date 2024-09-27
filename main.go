package main

import (
	"encoding/json"
	"fmt"
	"mil/inventory"
)

func main() {
	jsonString := `{"declarations":["m0_x"],"rules":[{"head":{"type":"type","name":"m0_x","module":"Main","id":0},"id":null,"axiom":true,"body":"eq(T, _14)"},{"head":{"type":"type","name":"m0_x","module":"Main","id":0},"id":12,"axiom":false,"body":"eq(_12, _m0_x_a)"},{"head":{"type":"type","name":"m0_x","module":"Main","id":0},"id":13,"axiom":false,"body":"eq(_13, _m0_x_a)"},{"head":{"type":"type","name":"m0_x","module":"Main","id":0},"id":14,"axiom":false,"body":"eq(_14, pair(function(_12), _13))"},{"head":{"type":"type","name":"m0_x","module":"Main","id":0},"id":null,"axiom":true,"body":"eq(T, _17)"},{"head":{"type":"type","name":"m0_x","module":"Main","id":0},"id":17,"axiom":false,"body":"eq(_17, _16)"},{"head":{"type":"type","name":"m0_x","module":"Main","id":0},"id":16,"axiom":false,"body":"eq(_16, builtin_Float)"}],"arguments":{},"classes":{"m0_A":[],"m0_B":["m0_A"]},"type_vars":{"m0_x":{"a":[]}},"node_depth":{"19":0,"3":1,"2":2,"1":3,"10":1,"7":2,"6":3,"4":4,"5":4,"9":2,"8":3,"11":1,"14":2,"12":3,"13":3,"18":1,"15":2,"17":2,"16":3},"node_graph":[{"parent":19,"child":3},{"parent":3,"child":2},{"parent":2,"child":1},{"parent":19,"child":10},{"parent":10,"child":7},{"parent":7,"child":6},{"parent":6,"child":4},{"parent":6,"child":5},{"parent":10,"child":9},{"parent":9,"child":8},{"parent":19,"child":11},{"parent":11,"child":14},{"parent":14,"child":12},{"parent":14,"child":13},{"parent":19,"child":18},{"parent":18,"child":15},{"parent":18,"child":17},{"parent":17,"child":16}],"max_depth":4}
`
	var input inventory.Input
	err := json.Unmarshal([]byte(jsonString), &input)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}
	fmt.Printf("Decoded Pair: %+v\n", input)
	inv := inventory.NewInventory(input)
	inv.Generalize(input.MaxLevel)
	p := inv.RenderProlog()
	fmt.Println(p)
}
