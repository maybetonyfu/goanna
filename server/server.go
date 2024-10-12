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
	report := haskell.MakeReport(errors, *inv, haskellFile)
	json.NewEncoder(w).Encode(report)
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
	resp, err := http.Post("http://localhost:8000/translate", "text/plain", strings.NewReader(text))
	if err != nil {
		return inventory.Input{}, err

	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return inventory.Input{}, err
	}
	var input inventory.Input
	err = json.Unmarshal(body, &input)
	if err != nil {
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
