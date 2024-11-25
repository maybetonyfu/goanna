package main

import (
	"encoding/json"
	"fmt"
	"goanna/haskell"
	"goanna/inventory"
	"goanna/marco"
	"io"
	"log"
	"net/http"
	"strings"
)

type Response struct {
	Stage         string
	ParsingErrors []inventory.Range
	ImportErrors  []inventory.Identifier
	TypeErrors    []haskell.TypeError
	LoadingErrors []string
	NodeRange     map[int]inventory.Range
	InferredTypes map[string]string
	TopLevels     []string
	Declarations  []string
}

const (
	ParsingStage      = "parse"
	TypeCheckingStage = "type-check"
	ImportErrorStage  = "import"
	WellTypedStage    = "well-typed"
)

func handleParsingError(w http.ResponseWriter, inv *inventory.Inventory) {
	response := Response{
		ParsingErrors: inv.ParsingErrors,
		TypeErrors:    []haskell.TypeError{},
		ImportErrors:  []inventory.Identifier{},
		Stage:         ParsingStage,
		NodeRange:     inv.NodeRange,
		InferredTypes: make(map[string]string),
		Declarations:  inv.Declarations,
		TopLevels:     inv.TopLevels,
	}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		panic(err)
	}
}

func handleImportError(w http.ResponseWriter, inv *inventory.Inventory) {
	response := Response{
		ParsingErrors: []inventory.Range{},
		TypeErrors:    []haskell.TypeError{},
		ImportErrors:  inv.ImportErrors,
		Stage:         ImportErrorStage,
		NodeRange:     inv.NodeRange,
		InferredTypes: make(map[string]string),
		Declarations:  inv.Declarations,
		TopLevels:     inv.TopLevels,
	}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		panic(err)
	}
}

func typeCheck(w http.ResponseWriter, r *http.Request) {
	// Allow all origins
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept")

	haskellFile, err := getHaskellFile(r)
	if err != nil {
		fmt.Println("Error extracting Haskell file")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	input, err := parseHaskellFile(haskellFile)
	if err != nil {
		fmt.Println("Error parsing Haskell file")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	inv := inventory.NewInventory(input)

	if len(inv.ParsingErrors) != 0 {
		handleParsingError(w, inv)
		return
	}
	if len(inv.ImportErrors) != 0 {
		handleImportError(w, inv)
		return
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
		report := haskell.MakeReport(errors, *inv, haskellFile)
		response := Response{
			Stage:         TypeCheckingStage,
			TypeErrors:    report.TypeErrors,
			ParsingErrors: []inventory.Range{},
			ImportErrors:  []inventory.Identifier{},
			NodeRange:     report.NodeRange,
			InferredTypes: make(map[string]string),
			Declarations:  inv.Declarations,
			TopLevels:     inv.TopLevels,
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			panic(err)
		}
	} else {
		// Well typed Program
		response := Response{
			Stage:         WellTypedStage,
			TypeErrors:    []haskell.TypeError{},
			ParsingErrors: []inventory.Range{},
			ImportErrors:  []inventory.Identifier{},
			NodeRange:     inv.NodeRange,
			InferredTypes: haskell.InferTypes(*inv),
			Declarations:  inv.Declarations,
			TopLevels:     inv.TopLevels,
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			panic(err)
		}
	}

}

func renderProlog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept")
	haskellFile, err := getHaskellFile(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	input, err := parseHaskellFile(haskellFile)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	inv := inventory.NewInventory(input)
	level := input.MaxLevel
	inv.Generalize(level)
	prologText := inv.RenderProlog()
	_, err = fmt.Fprintf(w, prologText)
	if err != nil {
		panic(err)
	}
}

func getHaskellFile(r *http.Request) (string, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		return "", err
	}
	// Close the request body
	defer func() {
		err := r.Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
	}()
	// Convert the byte slice to a string
	requestBody := string(bodyBytes)
	return requestBody, nil
}

func parseHaskellFile(text string) (inventory.Input, error) {
	resp, err := http.Post(
		"http://fly-local-6pn:8090/translate",
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

func main() {
	http.HandleFunc("/prolog", renderProlog)
	http.HandleFunc("/typecheck", typeCheck)
	_ = http.ListenAndServe(":8080", nil)
}
