package main

import (
	"encoding/json"
	"fmt"
	"mil/inventory"
	prolog_tool "mil/prolog-tool"
)

func main() {
	jsonString := `{"declarations":["m0_x"],"rules":[{"head":{"type":"type","name":"m0_x","module":"Main","id":0},"id":null,"axiom":true,"body":"eq(T, _2)"},{"head":{"type":"type","name":"m0_x","module":"Main","id":0},"id":2,"axiom":false,"body":"eq(_2, builtin_Float)"},{"head":{"type":"type","name":"m0_x","module":"Main","id":0},"id":null,"axiom":true,"body":"eq(T, _5)"},{"head":{"type":"type","name":"m0_x","module":"Main","id":0},"id":5,"axiom":false,"body":"eq(_5, _4)"},{"head":{"type":"type","name":"m0_x","module":"Main","id":0},"id":4,"axiom":false,"body":"eq(_4, builtin_Float)"}],"arguments":{},"classes":{},"type_vars":{},"node_depth":{"7":0,"1":1,"2":2,"6":1,"3":2,"5":2,"4":3},"node_graph":[{"parent":7,"child":1},{"parent":1,"child":2},{"parent":7,"child":6},{"parent":6,"child":3},{"parent":6,"child":5},{"parent":5,"child":4}],"max_depth":3}`
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
	pl := prolog_tool.NewProlog()
	y := pl.ConsultAndCheck(p, "type_check.")
	fmt.Println(y)

	s, r := pl.ConsultAndQuery1(p, "main(G).")

	fmt.Println(s, r)
}
