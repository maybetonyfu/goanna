package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"mil/haskell"
	"mil/inventory"
	"mil/marco"
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
}

const (
	ParsingStage      = "parse"
	TypeCheckingStage = "type-check"
	ImportErrorStage  = "import"
)

func handleParsingError(w http.ResponseWriter, inv *inventory.Inventory) {
	response := Response{
		ParsingErrors: inv.ParsingErrors,
		TypeErrors:    []haskell.TypeError{},
		ImportErrors:  []inventory.Identifier{},
		Stage:         ParsingStage,
		NodeRange:     inv.NodeRange,
	}
	json.NewEncoder(w).Encode(response)
}

func handleImportError(w http.ResponseWriter, inv *inventory.Inventory) {
	response := Response{
		ParsingErrors: []inventory.Range{},
		TypeErrors:    []haskell.TypeError{},
		ImportErrors:  inv.ImportErrors,
		Stage:         ImportErrorStage,
		NodeRange:     inv.NodeRange,
	}
	json.NewEncoder(w).Encode(response)
}

func typeCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
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
			break

		} else {
			break
		}
	}
	report := haskell.MakeReport(errors, *inv, haskellFile)
	response := Response{
		Stage:         TypeCheckingStage,
		TypeErrors:    report.TypeErrors,
		ParsingErrors: []inventory.Range{},
		ImportErrors:  []inventory.Identifier{},
		NodeRange:     report.NodeRange,
	}
	json.NewEncoder(w).Encode(response)
}

func renderProlog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
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
	fmt.Fprintf(w, prologText)
}

func getHaskellFile(r *http.Request) (string, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		return "", err
	}
	// Close the request body
	defer r.Body.Close()
	// Convert the byte slice to a string
	requestBody := string(bodyBytes)
	return requestBody, nil
}

func parseHaskellFile(text string) (inventory.Input, error) {
	resp, err := http.Post(
		"http://localhost:8000/translate",
		"text/plain",
		strings.NewReader(text))
	if err != nil {
		fmt.Println("Error making HTTP request")
		return inventory.Input{}, err

	}
	defer resp.Body.Close()
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

func mainPage(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("template/main.html")
	if err != nil {
		fmt.Println("Error parsing file")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	t.Execute(w, struct{}{})
}

func main() {
	http.HandleFunc("/prolog", renderProlog)
	http.HandleFunc("/typecheck", typeCheck)
	http.HandleFunc("/", mainPage)
	http.ListenAndServe(":8090", nil)
}
