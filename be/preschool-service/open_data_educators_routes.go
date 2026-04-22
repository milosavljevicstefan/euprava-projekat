package main

import (
	"encoding/json"
	"net/http"
)

func init() {
	http.HandleFunc("/otvoreni-podaci/vaspitaci", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		items, err := getOpenDataEducators(r.Context())
		if err != nil {
			http.Error(w, "Greska pri citanju vaspitaca", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})
}
