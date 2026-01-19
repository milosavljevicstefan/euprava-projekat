package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Vrtic struct {
	Naziv           string `json:"naziv"`
	Tip             string `json:"tip"` // "drzavni" ili "privatni"
	Grad            string `json:"grad"`
	Opstina         string `json:"opstina"`
	MaxKapacitet    int    `json:"max_kapacitet"`
	TrenutnoUpisano int    `json:"trenutno_upisano"`
}

func main() {
	// Osnovni pozdrav
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Preschool servis (8081) je online.")
	})

	// Test podaci za kolegu
	http.HandleFunc("/vrtici", func(w http.ResponseWriter, r *http.Request) {
		podaci := []Vrtic{
			{
				Naziv:           "Plavi Cuperak",
				Tip:             "drzavni",
				Grad:            "Beograd",
				Opstina:         "Zvezdara",
				MaxKapacitet:    120,
				TrenutnoUpisano: 95,
			},
			{
				Naziv:           "Sumica",
				Tip:             "privatni",
				Grad:            "Beograd",
				Opstina:         "Vozdovac",
				MaxKapacitet:    60,
				TrenutnoUpisano: 58,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(podaci)
	})

	fmt.Println("Preschool servis na 8081...")
	http.ListenAndServe(":8081", nil)
}
