package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"encoding/csv"
	"net/http"
	"strconv"
	"fmt"
	"sort"
)

var openData map[string]int
const preschoolBaseURL = "http://preschool-app:8081"

type ExternalVrtic struct {
	Naziv           string  `json:"naziv"`
	Opstina         string  `json:"opstina"`
	MaxKapacitet    int     `json:"max_kapacitet"`
	TrenutnoUpisano int     `json:"trenutno_upisano"`
	Popunjenost     float64 `json:"popunjenost"`
	SlobodnaMesta   int     `json:"slobodna_mesta"`
	Kriticno        bool    `json:"kriticno"`
}

type OpstinaReport struct {
	Opstina         string  `json:"opstina"`
	BrojVrtica      int     `json:"broj_vrtica"`
	UkupanKapacitet int     `json:"ukupan_kapacitet"`
	UkupnoUpisano   int     `json:"ukupno_upisano"`
	Popunjenost     float64 `json:"popunjenost"`
}

type CoverageResponse struct {
	Opstina      string  `json:"opstina"`
	BrojDece     int     `json:"broj_dece"`
	Kapacitet    int     `json:"kapacitet"`
	Deficit      int     `json:"deficit"`
	Pokrivenost  float64 `json:"pokrivenost"`
}

func loadOpenData() error {
    f, err := os.Open("open_data.csv")
    if err != nil {
        return err
    }
    defer f.Close()

    reader := csv.NewReader(f)
    records, err := reader.ReadAll()
    if err != nil {
        return err
    }

    openData = make(map[string]int)
    for i, rec := range records {
        if i == 0 {
            // skip header
            continue
        }
        broj, err := strconv.Atoi(rec[1])
        if err != nil {
            log.Printf("Neispravan broj dece za %s: %v", rec[0], err)
            continue
        }
        openData[rec[0]] = broj
    }
    return nil
}

func fetchAllVrtici() ([]ExternalVrtic, error) {
	resp, err := http.Get(preschoolBaseURL + "/vrtici")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var vrtici []ExternalVrtic
	err = json.NewDecoder(resp.Body).Decode(&vrtici)
	return vrtici, err
}

func fetchOpstinaReport() ([]OpstinaReport, error) {
	resp, err := http.Get(preschoolBaseURL + "/vrtici/izvestaj/opstina")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var report []OpstinaReport
	err = json.NewDecoder(resp.Body).Decode(&report)
	return report, err
}

func calculateCoverage(opstina string) (*CoverageResponse, error) {
    report, err := fetchOpstinaReport()
    if err != nil {
        return nil, err
    }

    brojDece, ok := openData[opstina]
    if !ok {
        return nil, errors.New("Opstina nije u open data")
    }

    for _, r := range report {
        if r.Opstina == opstina {
            deficit := brojDece - r.UkupanKapacitet
            pokrivenost := float64(r.UkupanKapacitet) / float64(brojDece) * 100

            return &CoverageResponse{
                Opstina:     opstina,
                BrojDece:    brojDece,
                Kapacitet:   r.UkupanKapacitet,
                Deficit:     deficit,
                Pokrivenost: pokrivenost,
            }, nil
        }
    }

    // ako opština postoji u open data ali nema vrtica
    return &CoverageResponse{
        Opstina:     opstina,
        BrojDece:    brojDece,
        Kapacitet:   0,
        Deficit:     brojDece,
        Pokrivenost: 0,
    }, nil
}

func rankingOpstina() ([]OpstinaReport, error) {
	report, err := fetchOpstinaReport()
	if err != nil {
		return nil, err
	}

	sort.Slice(report, func(i, j int) bool {
		return report[i].Popunjenost > report[j].Popunjenost
	})

	return report, nil
}

func projection(years int) ([]OpstinaReport, error) {
	report, err := fetchOpstinaReport()
	if err != nil {
		return nil, err
	}

	growth := 0.05 // 5% godišnje

	for i := range report {
		for y := 0; y < years; y++ {
			report[i].UkupnoUpisano = int(float64(report[i].UkupnoUpisano) * (1 + growth))
		}
		report[i].Popunjenost =
			float64(report[i].UkupnoUpisano) / float64(report[i].UkupanKapacitet)
	}

	return report, nil
}

func coverageHandler(w http.ResponseWriter, r *http.Request) {
	opstina := r.URL.Query().Get("opstina")
	result, err := calculateCoverage(opstina)
	if err != nil {
		http.Error(w, "Greska", http.StatusInternalServerError)
		return
	}
	if result == nil {
		http.Error(w, "Opstina nije pronadjena ili nema podataka", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func rankingHandler(w http.ResponseWriter, r *http.Request) {
	result, err := rankingOpstina()
	if err != nil {
		http.Error(w, "Greska", 500)
		return
	}
	json.NewEncoder(w).Encode(result)
}

func projectionHandler(w http.ResponseWriter, r *http.Request) {
	yearsStr := r.URL.Query().Get("years")
	years, _ := strconv.Atoi(yearsStr)

	result, err := projection(years)
	if err != nil {
		http.Error(w, "Greska", 500)
		return
	}
	json.NewEncoder(w).Encode(result)
}

func main() {
	if err := loadOpenData(); err != nil {
        log.Fatalf("Greska pri ucitavanju open data: %v", err)
    } else {
		log.Println("Open data ucitana uspesno")
	}
	http.HandleFunc("/analytics/coverage", coverageHandler)
	http.HandleFunc("/analytics/ranking", rankingHandler)
	http.HandleFunc("/analytics/projection", projectionHandler)

	fmt.Println("Open Data servis na 8082...")
	http.ListenAndServe(":8082", nil)
}