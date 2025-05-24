package main

import (
	"encoding/json"
	"fmt"
	"goanna/inventory"
	"goanna/marco"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type datum struct {
	lineOfCode          int
	numberOfSyntaxNodes int
	numberOfCauses      int
	numberOfLocations   int
	duration            int
}

func parseHaskellFile(text string) (inventory.Input, error) {
	resp, err := http.Post(
		"http://localhost:8090/translate",
		"text/plain",
		strings.NewReader(text))
	if err != nil {
		fmt.Println("Error making HTTP request")
		return inventory.Input{}, err

	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error read respond body")
		return inventory.Input{}, err
	}
	var input inventory.Input
	err = json.Unmarshal(body, &input)
	if err != nil {
		fmt.Printf("Error unmarshalling JSON response: %v", err)
		return inventory.Input{}, err
	}
	return input, nil
}

func typecheck(source string) datum {
	start := time.Now()

	input, err := parseHaskellFile(source)
	if err != nil {
		fmt.Println("Error parsing Haskell file")
		panic("Error parsing Haskell file")
	}

	inv := inventory.NewInventory(input)

	if len(inv.ParsingErrors) != 0 {
		fmt.Println(inv.ParsingErrors)
		panic("Error parsing Haskell file")
	}
	if len(inv.ImportErrors) != 0 {
		fmt.Println(inv.ImportErrors)
		panic("Error importing names")
	}
	level := input.MaxLevel
	errors := make([]marco.Error, 0)

	for {
		if level == 0 {
			panic("No more level to generalize ")
		}
		inv.Generalize(level)
		if !inv.AxiomCheck() {
			level = level - 1
			continue
		}
		if !inv.TypeCheck() {
			ruleIds := inv.EffectiveRules
			inv.ConsultAxioms()
			mc := marco.NewMarco(ruleIds, inv.Satisfiable)
			mc.Run()

			errors = mc.Analysis()
			if len(errors) == 1 && len(errors[0].CriticalNodes) == 0 {
				fmt.Printf("No solutions: %v\n", errors)
				level = level - 1
				continue
			}
			break
		} else {
			fmt.Println("No type error")
			break
		}
	}
	if len(errors) != 0 { // Type error found
		duration := time.Since(start)
		fmt.Println(duration)
		return datum{
			lineOfCode:          len(strings.Split(source, "\n")),
			duration:            int(duration / time.Millisecond),
			numberOfCauses:      len(errors[0].Causes),
			numberOfLocations:   len(errors[0].CriticalNodes),
			numberOfSyntaxNodes: len(input.NodeRange),
		}
	} else {
		// Well typed Program
		panic("No type error")
	}
}

func main() {
	fileName := "data.csv"
	writeHeader(fileName)
	for i := 7; i <= 300; i += 5 {
		for j := 4; j <= 30; j += 2 {
			if i < j {
				break
			}
			haskellFile := makeMultiParty(i, j)
			fmt.Printf("Error spec: (%d, %d)\n", i, j)
			d := typecheck(haskellFile)
			fmt.Printf("---------------------------\n")
			writeCsvRow(fileName, d)
		}
	}
	//haskellFile := makeMultiParty(10, 4)
	//fmt.Println(haskellFile)
	//typecheck(haskellFile)

}
