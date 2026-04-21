package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ExternalZahtev struct {
	ID           string `json:"id"`
	ImeRoditelja string `json:"ime_roditelja"`
	Status       string `json:"status"`
}

type ExternalKonkurs struct {
	ID             string    `json:"id"`
	VrticID        string    `json:"vrtic_id"`
	VrticNaziv     string    `json:"vrtic_naziv"`
	DatumPocetka   time.Time `json:"datum_pocetka"`
	DatumZavrsetka time.Time `json:"datum_zavrsetka"`
	MaxMesta       int       `json:"max_mesta"`
	Aktivan        bool      `json:"aktivan"`
	Status         string    `json:"status"`
	Popunjeno      int       `json:"popunjeno"`
	SlobodnaMesta  int       `json:"slobodna_mesta"`
}

type AllDataResponse struct {
	GeneratedAt time.Time         `json:"generated_at"`
	Vrtici      []ExternalVrtic   `json:"vrtici"`
	Zahtevi     []ExternalZahtev  `json:"zahtevi"`
	Konkursi    []ExternalKonkurs `json:"konkursi"`
}

func fetchOpenDataRequests() ([]ExternalZahtev, error) {
	resp, err := http.Get(preschoolBaseURL + "/otvoreni-podaci/zahtevi")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("preschool-service zahtevi status: %d", resp.StatusCode)
	}

	var items []ExternalZahtev
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func fetchAllKonkursi() ([]ExternalKonkurs, error) {
	resp, err := http.Get(preschoolBaseURL + "/konkursi")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("preschool-service konkursi status: %d", resp.StatusCode)
	}

	var items []ExternalKonkurs
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func allDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vrtici, err := fetchAllVrtici()
	if err != nil {
		http.Error(w, "Greska pri citanju vrtica", http.StatusInternalServerError)
		return
	}

	zahtevi, err := fetchOpenDataRequests()
	if err != nil {
		http.Error(w, "Greska pri citanju zahteva", http.StatusInternalServerError)
		return
	}

	konkursi, err := fetchAllKonkursi()
	if err != nil {
		http.Error(w, "Greska pri citanju konkursa", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AllDataResponse{
		GeneratedAt: time.Now(),
		Vrtici:      vrtici,
		Zahtevi:     zahtevi,
		Konkursi:    konkursi,
	})
}

func init() {
	http.HandleFunc("/analytics/all-data", allDataHandler)
}
