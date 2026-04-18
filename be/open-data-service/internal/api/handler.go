package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/milosavljevicstefan/euprava-projekat/open-data-service/internal/service"
)

// Handler drži referencu na servis i registruje sve rute.
type Handler struct {
	svc *service.OpenDataService
}

// NewHandler kreira novi Handler sa zadatim servisom.
func NewHandler(svc *service.OpenDataService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registruje sve HTTP rute na zadatom mux-u.
// Sve rute su pod prefiksom /open-data/
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// CSV endpointi
	mux.HandleFunc("/open-data/vrtici/csv", h.GetVrticiCSV)
	mux.HandleFunc("/open-data/zahtevi/csv", h.GetZahteviCSV)
	mux.HandleFunc("/open-data/konkursi/csv", h.GetKonkursiCSV)
	mux.HandleFunc("/open-data/ocene/csv", h.GetOceneCSV)

	// JSON endpointi
	mux.HandleFunc("/open-data/vrtici/json", h.GetVrticiJSON)
	mux.HandleFunc("/open-data/zahtevi/json", h.GetZahteviJSON)

	// Generički download endpoint
	mux.HandleFunc("/open-data/download", h.Download)

	// Health check endpoint (korisno za Docker/k8s probe)
	mux.HandleFunc("/health", h.HealthCheck)
}

// =========================================================
// CSV HANDLERI
// =========================================================

// GetVrticiCSV vraća CSV fajl sa podacima o vrtićima.
// GET /open-data/vrtici/csv
func (h *Handler) GetVrticiCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "dozvoljeno samo GET")
		return
	}

	csvBytes, filename, err := h.svc.GetVrticiCSV()
	if err != nil {
		log.Printf("[ERROR] GetVrticiCSV: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("greška pri preuzimanju podataka: %v", err))
		return
	}

	writeCSV(w, csvBytes, filename)
}

// GetZahteviCSV vraća CSV fajl sa zahtevima za upis.
// GET /open-data/zahtevi/csv
func (h *Handler) GetZahteviCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "dozvoljeno samo GET")
		return
	}

	csvBytes, filename, err := h.svc.GetZahteviCSV()
	if err != nil {
		log.Printf("[ERROR] GetZahteviCSV: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("greška pri preuzimanju podataka: %v", err))
		return
	}

	writeCSV(w, csvBytes, filename)
}

// GetKonkursiCSV vraća CSV fajl sa konkursima.
// GET /open-data/konkursi/csv
func (h *Handler) GetKonkursiCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "dozvoljeno samo GET")
		return
	}

	csvBytes, filename, err := h.svc.GetKonkursiCSV()
	if err != nil {
		log.Printf("[ERROR] GetKonkursiCSV: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("greška pri preuzimanju podataka: %v", err))
		return
	}

	writeCSV(w, csvBytes, filename)
}

// GetOceneCSV vraća CSV fajl sa ocenama vrtića.
// GET /open-data/ocene/csv
func (h *Handler) GetOceneCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "dozvoljeno samo GET")
		return
	}

	csvBytes, filename, err := h.svc.GetOceneCSV()
	if err != nil {
		log.Printf("[ERROR] GetOceneCSV: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("greška pri preuzimanju podataka: %v", err))
		return
	}

	writeCSV(w, csvBytes, filename)
}

// =========================================================
// JSON HANDLERI
// =========================================================

// GetVrticiJSON vraća JSON sa vrtićima i metapodacima (timestamp, count).
// GET /open-data/vrtici/json
func (h *Handler) GetVrticiJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "dozvoljeno samo GET")
		return
	}

	jsonBytes, err := h.svc.GetVrticiJSON()
	if err != nil {
		log.Printf("[ERROR] GetVrticiJSON: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("greška pri preuzimanju podataka: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, jsonBytes)
}

// GetZahteviJSON vraća JSON sa zahtevima i metapodacima.
// GET /open-data/zahtevi/json
func (h *Handler) GetZahteviJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "dozvoljeno samo GET")
		return
	}

	jsonBytes, err := h.svc.GetZahteviJSON()
	if err != nil {
		log.Printf("[ERROR] GetZahteviJSON: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("greška pri preuzimanju podataka: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, jsonBytes)
}

// =========================================================
// DOWNLOAD HANDLER
// =========================================================

// Download je generički endpoint za preuzimanje dataseta u zadatom formatu.
// GET /open-data/download?dataset=vrtici&format=csv
//
// Query parametri:
//   - dataset: vrtici | zahtevi | konkursi | ocene  (obavezno)
//   - format:  csv | json                           (obavezno)
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "dozvoljeno samo GET")
		return
	}

	// Validacija query parametara
	dataset := r.URL.Query().Get("dataset")
	format := r.URL.Query().Get("format")

	if dataset == "" {
		writeError(w, http.StatusBadRequest, "nedostaje query parametar 'dataset' (dozvoljeno: vrtici, zahtevi, konkursi, ocene)")
		return
	}
	if format == "" {
		writeError(w, http.StatusBadRequest, "nedostaje query parametar 'format' (dozvoljeno: csv, json)")
		return
	}

	result, err := h.svc.GetDownload(dataset, format)
	if err != nil {
		log.Printf("[ERROR] Download dataset=%s format=%s: %v", dataset, format, err)
		// Razlikujemo grešku validacije (400) od greške servisa (500)
		status := http.StatusInternalServerError
		if isValidationError(err) {
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, result.Filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result.Content)
}

// =========================================================
// HEALTH CHECK
// =========================================================

// HealthCheck vraća status servisa (uvek 200 OK ako je servis gore).
// GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"status":  "ok",
		"service": "open-data-service",
	}
	b, _ := json.Marshal(resp)
	writeJSON(w, http.StatusOK, b)
}

// =========================================================
// HELPER FUNKCIJE ZA PISANJE ODGOVORA
// =========================================================

// writeCSV postavlja odgovarajuće headere i šalje CSV sadržaj.
func writeCSV(w http.ResponseWriter, data []byte, filename string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// writeJSON postavlja Content-Type i šalje sirovi JSON.
func writeJSON(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

// ErrorResponse je struktura za JSON greške.
type ErrorResponse struct {
	Error   string `json:"error"`
	Status  int    `json:"status"`
}

// writeError šalje JSON poruku greške sa zadatim statusom.
func writeError(w http.ResponseWriter, status int, message string) {
	resp := ErrorResponse{Error: message, Status: status}
	b, _ := json.Marshal(resp)
	writeJSON(w, status, b)
}

// isValidationError proverava da li je greška tipa validacije (400) ili serverska (500).
// Oslanjamo se na sadržaj poruke — jednostavno rešenje bez custom error tipova.
func isValidationError(err error) bool {
	msg := err.Error()
	return contains(msg, "nepoznat dataset") || contains(msg, "nepoznat format")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && searchString(s, substr))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
