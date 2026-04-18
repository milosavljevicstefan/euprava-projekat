// Package service sadrži poslovnu logiku za generisanje open data formata.
package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"github.com/milosavljevicstefan/euprava-projekat/open-data-service/internal/client"
	"github.com/milosavljevicstefan/euprava-projekat/open-data-service/internal/model"
)

// DatasetVersion sadrži podatke i metapodatke o verziji dataseta.
type DatasetVersion struct {
	Timestamp string      `json:"timestamp"` // Vreme preuzimanja podataka
	Dataset   string      `json:"dataset"`   // Naziv dataseta (npr. "vrtici")
	Count     int         `json:"count"`     // Broj zapisa
	Data      interface{} `json:"data"`      // Stvarni podaci
}

// OpenDataService je servis koji koordinira preuzimanje i formatiranje podataka.
type OpenDataService struct {
	client *client.VrticiClient
}

// NewOpenDataService kreira novi servis sa zadatim klijentom.
func NewOpenDataService(c *client.VrticiClient) *OpenDataService {
	return &OpenDataService{client: c}
}

// fetchData je interna helper metoda — poziva API i vraća ExportData.
func (s *OpenDataService) fetchData() (*model.ExportData, error) {
	data, err := s.client.FetchAllData()
	if err != nil {
		return nil, fmt.Errorf("nije moguće preuzeti podatke: %w", err)
	}
	return data, nil
}

// trenutnoVreme vraća ISO timestamp koji se koristi za verzionisanje.
func trenutnoVreme() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// =========================================================
// CSV GENERATORI
// =========================================================

// csvFromRows je generička helper funkcija koja gradi CSV bafer iz zaglavlja i redova.
func csvFromRows(header []string, rows [][]string) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Pisanje UTF-8 BOM za bolju kompatibilnost sa Excel-om
	buf.WriteString("\xEF\xBB\xBF")

	if err := w.Write(header); err != nil {
		return nil, fmt.Errorf("greška pri pisanju CSV zaglavlja: %w", err)
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("greška pri pisanju CSV reda: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("greška CSV writer-a: %w", err)
	}
	return buf.Bytes(), nil
}

// GetVrticiCSV preuzima podatke i generiše CSV za vrtiće.
func (s *OpenDataService) GetVrticiCSV() ([]byte, string, error) {
	data, err := s.fetchData()
	if err != nil {
		return nil, "", err
	}

	var header []string
	var rows [][]string

	if len(data.Vrtici) > 0 {
		header = data.Vrtici[0].CSVHeader()
	} else {
		header = model.Vrtic{}.CSVHeader()
	}

	for _, v := range data.Vrtici {
		rows = append(rows, v.CSVRow())
	}

	csvBytes, err := csvFromRows(header, rows)
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("vrtici_%s.csv", time.Now().Format("20060102_150405"))
	return csvBytes, filename, nil
}

// GetZahteviCSV preuzima podatke i generiše CSV za zahteve za upis.
func (s *OpenDataService) GetZahteviCSV() ([]byte, string, error) {
	data, err := s.fetchData()
	if err != nil {
		return nil, "", err
	}

	var header []string
	var rows [][]string

	if len(data.Zahtevi) > 0 {
		header = data.Zahtevi[0].CSVHeader()
	} else {
		header = model.ZahtevZaUpis{}.CSVHeader()
	}

	for _, z := range data.Zahtevi {
		rows = append(rows, z.CSVRow())
	}

	csvBytes, err := csvFromRows(header, rows)
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("zahtevi_%s.csv", time.Now().Format("20060102_150405"))
	return csvBytes, filename, nil
}

// GetKonkursiCSV preuzima podatke i generiše CSV za konkurse.
func (s *OpenDataService) GetKonkursiCSV() ([]byte, string, error) {
	data, err := s.fetchData()
	if err != nil {
		return nil, "", err
	}

	var header []string
	var rows [][]string

	if len(data.Konkursi) > 0 {
		header = data.Konkursi[0].CSVHeader()
	} else {
		header = model.Konkurs{}.CSVHeader()
	}

	for _, k := range data.Konkursi {
		rows = append(rows, k.CSVRow())
	}

	csvBytes, err := csvFromRows(header, rows)
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("konkursi_%s.csv", time.Now().Format("20060102_150405"))
	return csvBytes, filename, nil
}

// GetOceneCSV preuzima podatke i generiše CSV za ocene.
func (s *OpenDataService) GetOceneCSV() ([]byte, string, error) {
	data, err := s.fetchData()
	if err != nil {
		return nil, "", err
	}

	var header []string
	var rows [][]string

	if len(data.Ocene) > 0 {
		header = data.Ocene[0].CSVHeader()
	} else {
		header = model.Ocena{}.CSVHeader()
	}

	for _, o := range data.Ocene {
		rows = append(rows, o.CSVRow())
	}

	csvBytes, err := csvFromRows(header, rows)
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("ocene_%s.csv", time.Now().Format("20060102_150405"))
	return csvBytes, filename, nil
}

// =========================================================
// JSON GENERATORI
// =========================================================

// GetVrticiJSON vraća JSON odgovor sa verzionisanim datasetom vrtića.
func (s *OpenDataService) GetVrticiJSON() ([]byte, error) {
	data, err := s.fetchData()
	if err != nil {
		return nil, err
	}

	response := DatasetVersion{
		Timestamp: trenutnoVreme(),
		Dataset:   "vrtici",
		Count:     len(data.Vrtici),
		Data:      data.Vrtici,
	}
	return json.Marshal(response)
}

// GetZahteviJSON vraća JSON odgovor sa verzionisanim datasetom zahteva.
func (s *OpenDataService) GetZahteviJSON() ([]byte, error) {
	data, err := s.fetchData()
	if err != nil {
		return nil, err
	}

	response := DatasetVersion{
		Timestamp: trenutnoVreme(),
		Dataset:   "zahtevi",
		Count:     len(data.Zahtevi),
		Data:      data.Zahtevi,
	}
	return json.Marshal(response)
}

// =========================================================
// DOWNLOAD (generički endpoint)
// =========================================================

// DownloadResult sadrži sadržaj fajla, ime fajla i content-type.
type DownloadResult struct {
	Content     []byte
	Filename    string
	ContentType string
}

// GetDownload je generički handler koji na osnovu dataset i format parametara
// vraća odgovarajući fajl za preuzimanje.
func (s *OpenDataService) GetDownload(dataset, format string) (*DownloadResult, error) {
	switch format {
	case "csv":
		return s.downloadCSV(dataset)
	case "json":
		return s.downloadJSON(dataset)
	default:
		return nil, fmt.Errorf("nepoznat format '%s' — dozvoljeno: csv, json", format)
	}
}

func (s *OpenDataService) downloadCSV(dataset string) (*DownloadResult, error) {
	var content []byte
	var filename string
	var err error

	switch dataset {
	case "vrtici":
		content, filename, err = s.GetVrticiCSV()
	case "zahtevi":
		content, filename, err = s.GetZahteviCSV()
	case "konkursi":
		content, filename, err = s.GetKonkursiCSV()
	case "ocene":
		content, filename, err = s.GetOceneCSV()
	default:
		return nil, fmt.Errorf("nepoznat dataset '%s' — dozvoljeno: vrtici, zahtevi, konkursi, ocene", dataset)
	}

	if err != nil {
		return nil, err
	}
	return &DownloadResult{
		Content:     content,
		Filename:    filename,
		ContentType: "text/csv; charset=utf-8",
	}, nil
}

func (s *OpenDataService) downloadJSON(dataset string) (*DownloadResult, error) {
	var content []byte
	var err error

	switch dataset {
	case "vrtici":
		content, err = s.GetVrticiJSON()
	case "zahtevi":
		content, err = s.GetZahteviJSON()
	default:
		// Za konkurse i ocene, direktno vraćamo podatke
		data, ferr := s.fetchData()
		if ferr != nil {
			return nil, ferr
		}
		var resp DatasetVersion
		switch dataset {
		case "konkursi":
			resp = DatasetVersion{Timestamp: trenutnoVreme(), Dataset: "konkursi", Count: len(data.Konkursi), Data: data.Konkursi}
		case "ocene":
			resp = DatasetVersion{Timestamp: trenutnoVreme(), Dataset: "ocene", Count: len(data.Ocene), Data: data.Ocene}
		default:
			return nil, fmt.Errorf("nepoznat dataset '%s'", dataset)
		}
		content, err = json.Marshal(resp)
	}

	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%s_%s.json", dataset, time.Now().Format("20060102_150405"))
	return &DownloadResult{
		Content:     content,
		Filename:    filename,
		ContentType: "application/json; charset=utf-8",
	}, nil
}
