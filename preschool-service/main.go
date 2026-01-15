package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	// Osnovni pozdrav
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Preschool servis (8081) je online.")
	})

	// Test podaci za kolegu
	http.HandleFunc("/vrtici", func(w http.ResponseWriter, r *http.Request) {
		podaci := []map[string]interface{}{
			{"naziv": "Plavi Cuperak", "mesta": 20},
			{"naziv": "Sumica", "mesta": 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(podaci)
	})

	fmt.Println("Preschool servis na 8081...")
	http.ListenAndServe(":8081", nil)
}