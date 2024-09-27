package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func renderProlog(w http.ResponseWriter, r *http.Request) {
	var maxBytes int64 = 1048576 // 1MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	var person Person
	err := json.NewDecoder(r.Body).Decode(&person)
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
	// Send a response
	response := struct {
		Message string `json:"message"`
	}{
		Message: "Person received successfully",
	}
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/prolog", renderProlog)
	http.ListenAndServe(":8090", nil)
}
