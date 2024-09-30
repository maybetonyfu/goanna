package main

import (
	"encoding/json"
	"fmt"
	"mil/inventory"
	"mil/marco"
	"net/http"
)

func renderProlog(w http.ResponseWriter, r *http.Request) {
	var maxBytes int64 = 1048576 // 1MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	var input inventory.Input
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		switch e := err.(type) {
		case *json.SyntaxError:
			http.Error(w, fmt.Sprintf("JSON syntax error at position %d", e.Offset), http.StatusBadRequest)
		case *json.UnmarshalTypeError:
			http.Error(w, fmt.Sprintf("Invalid type for %s at position %d", e.Field, e.Offset), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	//err := json.Unmarshal([]byte(jsonString), &input)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}
	//fmt.Printf("Decoded JSON: %+v\n", input)
	inv := inventory.NewInventory(input)
	level := input.MaxLevel
	errors := make([]marco.Error, 0)
	for {
		if level == 0 {
			panic("No more level to generalize ")
		}
		fmt.Printf("Current level: %d\n", level)
		inv.Generalize(level)
		if !inv.AxiomCheck() {
			level = level - 1
			continue
		}

		if !inv.TypeCheck() {
			ruleIds := make([]int, len(inv.Rules))
			for i, rule := range inv.Rules {
				ruleIds[i] = rule.Id
			}
			mc := marco.NewMarco(ruleIds, inv.Satisfiable)
			mc.Run()
			errors = mc.Analysis()
			break

		} else {
			break
		}
	}

	json.NewEncoder(w).Encode(errors)
}

func main() {
	http.HandleFunc("/prolog", renderProlog)
	http.ListenAndServe(":8090", nil)
}
