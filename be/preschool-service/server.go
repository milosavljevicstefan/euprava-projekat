package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"net/http"
	"strings"
)
type AllDataResponse struct {
	Vrtici            any `json:"vrtici"`
	Kriticni          any `json:"kriticni"`
	OpstinaReport     any `json:"opstina_report"`
	Konkursi          any `json:"konkursi"`
	Rasporedi         any `json:"rasporedi_vaspitaca"`
	Zahtevi           any `json:"zahtevi_upisa"`
	Ocene             any `json:"ocene_vrtica"`
}
func allDataHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vrtici, err := handleVrticiList(r)
	if err != nil {
		http.Error(w, "Greska vrtici", http.StatusInternalServerError)
		return
	}

	kriticni, err := getKriticni(r.Context())
	if err != nil {
		http.Error(w, "Greska kriticni", http.StatusInternalServerError)
		return
	}

	opstina, err := izvestajPoOpstini(r.Context())
	if err != nil {
		http.Error(w, "Greska opstina report", http.StatusInternalServerError)
		return
	}

	konkursi, err := getAllKonkursViews(r.Context(), "", "")
	if err != nil {
		http.Error(w, "Greska konkursi", http.StatusInternalServerError)
		return
	}

	rasporedi, err := listAssignments(r.Context())
	if err != nil {
		http.Error(w, "Greska rasporedi", http.StatusInternalServerError)
		return
	}

	zahtevi, err := getAllRequests(r.Context())
	if err != nil {
		http.Error(w, "Greska zahtevi", http.StatusInternalServerError)
		return
	}

	resp := AllDataResponse{
		Vrtici:        vrtici,
		Kriticni:      kriticni,
		OpstinaReport: opstina,
		Konkursi:      konkursi,
		Rasporedi:     rasporedi,
		Zahtevi:       zahtevi,
		Ocene:         nil, // nema u kodu → preskočeno
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
func main() {
	initMongo()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		fmt.Fprint(w, "Preschool servis (8081) je online.")
	})
	http.HandleFunc("/analytics/all-data", allDataHandler)
	http.HandleFunc("/vrtici/kriticni", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		views, err := getKriticni(r.Context())
		if err != nil {
			http.Error(w, "Greska pri citanju iz baze", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(views)
	})

	http.HandleFunc("/vrtici/izvestaj/opstina", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		report, err := izvestajPoOpstini(r.Context())
		if err != nil {
			http.Error(w, "Greska pri citanju iz baze", http.StatusInternalServerError)
			return
		}

		if r.URL.Query().Get("format") == "pdf" || strings.Contains(r.Header.Get("Accept"), "application/pdf") {
			pdfBytes, err := buildOpstinaPDFReport(report)
			if err != nil {
				http.Error(w, "Greska pri generisanju PDF izvestaja", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", "attachment; filename=\"izvestaj-opstina.pdf\"")
			w.Write(pdfBytes)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	})

	http.HandleFunc("/vrtici", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		switch r.Method {
		case http.MethodGet:
			views, err := handleVrticiList(r)
			if err != nil {
				http.Error(w, "Greska pri citanju iz baze", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(views)
		case http.MethodPost:
			claims, err := requireAuth(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			if err := requireAdminRole(claims); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}

			var nov Vrtic
			if err := json.NewDecoder(r.Body).Decode(&nov); err != nil {
				http.Error(w, "Neispravan JSON", http.StatusBadRequest)
				return
			}
			if err := validateVrticInput(nov); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := insertVrtic(r.Context(), nov); err != nil {
				http.Error(w, "Greska pri upisu u bazu", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/konkursi", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		switch r.Method {
		case http.MethodGet:
			items, err := getAllKonkursViews(r.Context(), strings.TrimSpace(r.URL.Query().Get("status")), strings.TrimSpace(r.URL.Query().Get("vrtic_id")))
			if err != nil {
				http.Error(w, "Greska pri citanju konkursa", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(items)
		case http.MethodPost:
			claims, err := requireAuth(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			if err := requireAdminRole(claims); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}

			var req KonkursRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Neispravan JSON", http.StatusBadRequest)
				return
			}

			item, err := createKonkurs(r.Context(), req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(item)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/konkursi/", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := requireAdminRole(claims); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		id, action, err := parseKonkursAction(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if action != "zatvori" {
			http.Error(w, "Nepoznata akcija", http.StatusBadRequest)
			return
		}
		if err := closeKonkurs(r.Context(), id); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				http.Error(w, "Konkurs nije pronadjen", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/rasporedi-vaspitaca", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		switch r.Method {
		case http.MethodGet:
			if err := requireAdminRole(claims); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
			items, err := listAssignments(r.Context())
			if err != nil {
				http.Error(w, "Greska pri citanju rasporeda", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(items)
		case http.MethodPost:
			if err := requireAdminRole(claims); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
			var req VaspitacRasporedRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Neispravan JSON", http.StatusBadRequest)
				return
			}
			item, err := createAssignment(r.Context(), req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(item)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/rasporedi-vaspitaca/", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := requireAdminRole(claims); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		id, err := parseSimpleObjectID(r.URL.Path, "/rasporedi-vaspitaca/")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := deleteAssignment(r.Context(), id); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				http.Error(w, "Raspored nije pronadjen", http.StatusNotFound)
				return
			}
			http.Error(w, "Greska pri brisanju rasporeda", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/roditelj/vaspitaci", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := requireUserRole(claims); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		email := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
		items, err := getRoditeljVaspitaci(r.Context(), email)
		if err != nil {
			http.Error(w, "Greska pri citanju vaspitaca", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	http.HandleFunc("/sastanci/moji", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := requireUserRole(claims); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		email := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
		items, err := getMeetingsByParent(r.Context(), email)
		if err != nil {
			http.Error(w, "Greska pri citanju sastanaka", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	http.HandleFunc("/sastanci", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := requireUserRole(claims); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		var req SastanakRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Neispravan JSON", http.StatusBadRequest)
			return
		}
		item, err := createMeeting(r.Context(), claims, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)
	})

	http.HandleFunc("/vaspitac/deca", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := requireEducatorRole(claims); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		email := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
		items, err := getEducatorChildren(r.Context(), email)
		if err != nil {
			http.Error(w, "Greska pri citanju dece", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	http.HandleFunc("/vaspitac/sastanci", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := requireEducatorRole(claims); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		email := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
		items, err := getMeetingsByEducator(r.Context(), email)
		if err != nil {
			http.Error(w, "Greska pri citanju sastanaka", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	http.HandleFunc("/vaspitac/sastanci/", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := requireEducatorRole(claims); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		id, action, err := parseMeetingAction(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var payload SastanakActionPayload
		if r.Body != nil {
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
				http.Error(w, "Neispravan JSON", http.StatusBadRequest)
				return
			}
		}
		if err := processMeetingDecision(r.Context(), claims, id, action, payload.Reason); err != nil {
			status := http.StatusBadRequest
			switch {
			case errors.Is(err, mongo.ErrNoDocuments):
				status = http.StatusNotFound
			case strings.Contains(err.Error(), "Nemate dozvolu"):
				status = http.StatusForbidden
			}
			http.Error(w, err.Error(), status)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/vaspitac/obavestenja", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := requireEducatorRole(claims); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		var req SimptomObavestenjeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Neispravan JSON", http.StatusBadRequest)
			return
		}
		item, err := createSymptomsNotification(r.Context(), claims, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)
	})

	http.HandleFunc("/obavestenja/moja", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := requireUserRole(claims); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		email := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
		items, err := getNotificationsByParent(r.Context(), email)
		if err != nil {
			http.Error(w, "Greska pri citanju obavestenja", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	http.HandleFunc("/zahtevi-upisa/moji", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := requireUserRole(claims); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		email := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
		items, err := getRequestsByUser(r.Context(), email)
		if err != nil {
			http.Error(w, "Greska pri citanju zahteva", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	http.HandleFunc("/zahtevi-upisa", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		switch r.Method {
		case http.MethodPost:
			claims, err := requireAuth(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			if err := requireUserRole(claims); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}

			var req UpisRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Neispravan JSON", http.StatusBadRequest)
				return
			}

			newReq, err := createEnrollmentRequest(r.Context(), claims, req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(newReq)
		case http.MethodGet:
			claims, err := requireAuth(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			if err := requireAdminRole(claims); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}

			items, err := getAllRequests(r.Context())
			if err != nil {
				http.Error(w, "Greska pri citanju zahteva", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(items)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/zahtevi-upisa/", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		claims, err := requireAuth(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		switch r.Method {
		case http.MethodGet:
			id, action, err := parseRequestAction(r.URL.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if action != "dokument" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			item, err := getRequestByID(r.Context(), id)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					http.Error(w, "Zahtev nije pronadjen", http.StatusNotFound)
					return
				}
				http.Error(w, "Greska pri citanju zahteva", http.StatusInternalServerError)
				return
			}
			if !canAccessRequestDocument(item, claims) {
				http.Error(w, "Nemate dozvolu za ovaj dokument", http.StatusForbidden)
				return
			}

			pdf, fileName, err := buildRequestDecisionPDF(item)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
			w.Write(pdf)
			return

		case http.MethodPut:
			id, action, err := parseRequestAction(r.URL.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if action == "dokument" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			if action == "izmeni" {
				if err := requireUserRole(claims); err != nil {
					http.Error(w, err.Error(), http.StatusForbidden)
					return
				}
				var payload UpisRequest
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					http.Error(w, "Neispravan JSON", http.StatusBadRequest)
					return
				}
				item, err := updateEnrollmentRequest(r.Context(), claims, id, payload)
				if err != nil {
					status := http.StatusBadRequest
					switch {
					case errors.Is(err, mongo.ErrNoDocuments):
						status = http.StatusNotFound
					case strings.Contains(err.Error(), "Nemate dozvolu"):
						status = http.StatusForbidden
					}
					http.Error(w, err.Error(), status)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(item)
				return
			}

			if action == "dokumenta" {
				if err := requireUserRole(claims); err != nil {
					http.Error(w, err.Error(), http.StatusForbidden)
					return
				}
				var payload DokumentaUpdateRequest
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					http.Error(w, "Neispravan JSON", http.StatusBadRequest)
					return
				}
				if err := updateRequestDocuments(r.Context(), claims, id, payload); err != nil {
					status := http.StatusBadRequest
					switch {
					case errors.Is(err, mongo.ErrNoDocuments):
						status = http.StatusNotFound
					case strings.Contains(err.Error(), "Nemate dozvolu"):
						status = http.StatusForbidden
					}
					http.Error(w, err.Error(), status)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			if err := requireAdminRole(claims); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}

			var payload RequestActionPayload
			if r.Body != nil {
				defer r.Body.Close()
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
					http.Error(w, "Neispravan JSON", http.StatusBadRequest)
					return
				}
			}

			if err := processEnrollmentRequest(r.Context(), claims, id, action, payload.Reason); err != nil {
				status := http.StatusBadRequest
				switch {
				case errors.Is(err, mongo.ErrNoDocuments):
					status = http.StatusNotFound
				case strings.Contains(err.Error(), "Nemate dozvolu"):
					status = http.StatusForbidden
				}
				http.Error(w, err.Error(), status)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/vrtici/", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		id, err := parseVrticID(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			vrtic, err := getVrticByID(r.Context(), id)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					http.Error(w, "Vrtic nije pronadjen", http.StatusNotFound)
					return
				}
				http.Error(w, "Greska pri citanju iz baze", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(toViews([]Vrtic{vrtic})[0])
		case http.MethodPut:
			claims, err := requireAuth(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			if err := requireAdminRole(claims); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}

			var up Vrtic
			if err := json.NewDecoder(r.Body).Decode(&up); err != nil {
				http.Error(w, "Neispravan JSON", http.StatusBadRequest)
				return
			}
			if err := validateVrticInput(up); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if err := updateVrtic(r.Context(), id, up); err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					http.Error(w, "Vrtic nije pronadjen", http.StatusNotFound)
					return
				}
				http.Error(w, "Greska pri azuriranju", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			claims, err := requireAuth(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			if err := requireAdminRole(claims); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}

			if err := deleteVrtic(r.Context(), id); err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					http.Error(w, "Vrtic nije pronadjen", http.StatusNotFound)
					return
				}
				http.Error(w, "Greska pri brisanju", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Preschool servis na 8081...")
	http.ListenAndServe(":8081", nil)
}
