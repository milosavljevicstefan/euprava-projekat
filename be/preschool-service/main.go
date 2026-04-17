package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Vrtic struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Naziv           string             `json:"naziv" bson:"naziv"`
	Tip             string             `json:"tip" bson:"tip"`
	Grad            string             `json:"grad" bson:"grad"`
	Opstina         string             `json:"opstina" bson:"opstina"`
	MaxKapacitet    int                `json:"max_kapacitet" bson:"max_kapacitet"`
	TrenutnoUpisano int                `json:"trenutno_upisano" bson:"trenutno_upisano"`
	CreatedBy       string             `json:"created_by" bson:"created_by"`
}

type VrticView struct {
	ID              primitive.ObjectID `json:"id"`
	Naziv           string             `json:"naziv"`
	Tip             string             `json:"tip"`
	Grad            string             `json:"grad"`
	Opstina         string             `json:"opstina"`
	MaxKapacitet    int                `json:"max_kapacitet"`
	TrenutnoUpisano int                `json:"trenutno_upisano"`
	Popunjenost     float64            `json:"popunjenost"`
	SlobodnaMesta   int                `json:"slobodna_mesta"`
	Kriticno        bool               `json:"kriticno"`
	CreatedBy       string             `json:"created_by"`
}

type OpstinaIzvestaj struct {
	Opstina         string  `json:"opstina"`
	BrojVrtica      int     `json:"broj_vrtica"`
	UkupanKapacitet int     `json:"ukupan_kapacitet"`
	UkupnoUpisano   int     `json:"ukupno_upisano"`
	Popunjenost     float64 `json:"popunjenost"`
}

type UpisZahtev struct {
	ID            primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	VrticID       primitive.ObjectID  `json:"vrtic_id" bson:"vrtic_id"`
	VrticNaziv    string              `json:"vrtic_naziv" bson:"vrtic_naziv"`
	VrticOwner    string              `json:"vrtic_owner" bson:"vrtic_owner"`
	ImeRoditelja  string              `json:"ime_roditelja" bson:"ime_roditelja"`
	ImeDeteta     string              `json:"ime_deteta" bson:"ime_deteta"`
	BrojGodina    int                 `json:"broj_godina" bson:"broj_godina"`
	KorisnikEmail string              `json:"korisnik_email" bson:"korisnik_email"`
	Status        string              `json:"status" bson:"status"`
	CreatedAt     time.Time           `json:"created_at" bson:"created_at"`
	ProcessedAt   *time.Time          `json:"processed_at,omitempty" bson:"processed_at,omitempty"`
	ProcessedBy   string              `json:"processed_by,omitempty" bson:"processed_by,omitempty"`
}

type UpisRequest struct {
	VrticID      string `json:"vrtic_id"`
	ImeRoditelja string `json:"ime_roditelja"`
	ImeDeteta    string `json:"ime_deteta"`
	BrojGodina   int    `json:"broj_godina"`
}

var vrticiCollection *mongo.Collection
var zahteviCollection *mongo.Collection

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

			nov.CreatedBy = strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
			if nov.CreatedBy == "" {
				http.Error(w, "Neispravan token", http.StatusUnauthorized)
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

		id, action, err := parseRequestAction(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := processEnrollmentRequest(r.Context(), claims, id, action); err != nil {
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

			existing, err := getVrticByID(r.Context(), id)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					http.Error(w, "Vrtic nije pronadjen", http.StatusNotFound)
					return
				}
				http.Error(w, "Greska pri citanju iz baze", http.StatusInternalServerError)
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
			up.CreatedBy = existing.CreatedBy

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

func parseVrticID(path string) (primitive.ObjectID, error) {
	idPart := strings.TrimPrefix(path, "/vrtici/")
	idPart = strings.Trim(idPart, "/")
	if idPart == "" || strings.Contains(idPart, "/") {
		return primitive.NilObjectID, errors.New("Neispravan ID")
	}
	id, err := primitive.ObjectIDFromHex(idPart)
	if err != nil {
		return primitive.NilObjectID, errors.New("Neispravan ID")
	}
	return id, nil
}

func parseRequestAction(path string) (primitive.ObjectID, string, error) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, "/zahtevi-upisa/"), "/"), "/")
	if len(parts) != 2 {
		return primitive.NilObjectID, "", errors.New("Neispravan URL zahteva")
	}
	id, err := primitive.ObjectIDFromHex(parts[0])
	if err != nil {
		return primitive.NilObjectID, "", errors.New("Neispravan ID zahteva")
	}
	action := parts[1]
	if action != "odobri" && action != "odbij" {
		return primitive.NilObjectID, "", errors.New("Nepoznata akcija")
	}
	return id, action, nil
}

func validateVrticInput(v Vrtic) error {
	if strings.TrimSpace(v.Naziv) == "" {
		return errors.New("Naziv je obavezan")
	}
	if strings.TrimSpace(v.Tip) == "" {
		return errors.New("Tip je obavezan")
	}
	if v.MaxKapacitet <= 0 {
		return errors.New("Max kapacitet mora biti > 0")
	}
	if v.TrenutnoUpisano < 0 {
		return errors.New("Trenutno upisano mora biti >= 0")
	}
	if v.TrenutnoUpisano > v.MaxKapacitet {
		return errors.New("Trenutno upisano ne moze biti vece od kapaciteta")
	}
	return nil
}

func validateEnrollmentInput(req UpisRequest) error {
	if strings.TrimSpace(req.VrticID) == "" {
		return errors.New("Vrtic je obavezan")
	}
	if strings.TrimSpace(req.ImeRoditelja) == "" {
		return errors.New("Ime roditelja je obavezno")
	}
	if strings.TrimSpace(req.ImeDeteta) == "" {
		return errors.New("Ime deteta je obavezno")
	}
	if req.BrojGodina <= 0 || req.BrojGodina > 7 {
		return errors.New("Broj godina mora biti izmedju 1 i 7")
	}
	return nil
}

func requireAdminRole(claims jwt.MapClaims) error {
	if strings.ToLower(strings.TrimSpace(claimString(claims, "role"))) == "admin" {
		return nil
	}
	return errors.New("Nemate dozvolu za admin operacije")
}

func requireUserRole(claims jwt.MapClaims) error {
	if strings.ToLower(strings.TrimSpace(claimString(claims, "role"))) == "korisnik" {
		return nil
	}
	return errors.New("Samo korisnik moze slati zahtev za upis")
}

func ensureOwnership(claims jwt.MapClaims, createdBy string) error {
	actor := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
	owner := strings.ToLower(strings.TrimSpace(createdBy))
	if actor == "" || owner == "" || actor != owner {
		return errors.New("Mozete menjati samo svoje vrtice")
	}
	return nil
}

func handleVrticiList(r *http.Request) ([]VrticView, error) {
	all, err := getAllVrtici(r.Context())
	if err != nil {
		return nil, err
	}

	tip := strings.TrimSpace(r.URL.Query().Get("tip"))
	grad := strings.TrimSpace(r.URL.Query().Get("grad"))
	opstina := strings.TrimSpace(r.URL.Query().Get("opstina"))
	sortBy := r.URL.Query().Get("sort")

	var filtered []Vrtic
	for _, v := range all {
		if tip != "" && v.Tip != tip {
			continue
		}
		if grad != "" && !strings.EqualFold(v.Grad, grad) {
			continue
		}
		if opstina != "" && !strings.EqualFold(v.Opstina, opstina) {
			continue
		}
		filtered = append(filtered, v)
	}

	views := toViews(filtered)
	if sortBy == "slobodna_mesta" {
		sort.Slice(views, func(i, j int) bool {
			return views[i].SlobodnaMesta > views[j].SlobodnaMesta
		})
	} else {
		sort.Slice(views, func(i, j int) bool {
			return strings.ToLower(views[i].Naziv) < strings.ToLower(views[j].Naziv)
		})
	}

	return views, nil
}

func getKriticni(ctx context.Context) ([]VrticView, error) {
	all, err := getAllVrtici(ctx)
	if err != nil {
		return nil, err
	}
	var kriticni []Vrtic
	for _, v := range all {
		if popunjenost(v) >= 0.9 {
			kriticni = append(kriticni, v)
		}
	}
	return toViews(kriticni), nil
}

func izvestajPoOpstini(ctx context.Context) ([]OpstinaIzvestaj, error) {
	all, err := getAllVrtici(ctx)
	if err != nil {
		return nil, err
	}

	byOpstina := map[string]*OpstinaIzvestaj{}
	for _, v := range all {
		key := v.Opstina
		if key == "" {
			key = "Nepoznata"
		}
		entry, ok := byOpstina[key]
		if !ok {
			entry = &OpstinaIzvestaj{Opstina: key}
			byOpstina[key] = entry
		}
		entry.BrojVrtica++
		entry.UkupanKapacitet += v.MaxKapacitet
		entry.UkupnoUpisano += v.TrenutnoUpisano
	}

	var report []OpstinaIzvestaj
	for _, v := range byOpstina {
		if v.UkupanKapacitet > 0 {
			v.Popunjenost = float64(v.UkupnoUpisano) / float64(v.UkupanKapacitet)
		}
		report = append(report, *v)
	}

	sort.Slice(report, func(i, j int) bool {
		return report[i].Opstina < report[j].Opstina
	})

	return report, nil
}

func createEnrollmentRequest(ctx context.Context, claims jwt.MapClaims, req UpisRequest) (*UpisZahtev, error) {
	if err := validateEnrollmentInput(req); err != nil {
		return nil, err
	}

	vrticID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.VrticID))
	if err != nil {
		return nil, errors.New("Neispravan ID vrtica")
	}

	vrtic, err := getVrticByID(ctx, vrticID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("Vrtic nije pronadjen")
		}
		return nil, err
	}

	if slobodnaMesta(vrtic) <= 0 {
		return nil, errors.New("Vrtic nema slobodnih mesta")
	}

	korisnikEmail := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
	if korisnikEmail == "" {
		return nil, errors.New("Neispravan token")
	}

	exists, err := zahteviCollection.CountDocuments(ctx, bson.M{
		"vrtic_id":       vrticID,
		"korisnik_email": korisnikEmail,
		"ime_deteta":     strings.TrimSpace(req.ImeDeteta),
		"status":         bson.M{"$in": []string{"na_cekanju", "odobren"}},
	})
	if err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, errors.New("Vec postoji aktivan zahtev za ovo dete u izabranom vrticu")
	}

	item := UpisZahtev{
		VrticID:       vrticID,
		VrticNaziv:    vrtic.Naziv,
		VrticOwner:    vrtic.CreatedBy,
		ImeRoditelja:  strings.TrimSpace(req.ImeRoditelja),
		ImeDeteta:     strings.TrimSpace(req.ImeDeteta),
		BrojGodina:    req.BrojGodina,
		KorisnikEmail: korisnikEmail,
		Status:        "na_cekanju",
		CreatedAt:     time.Now(),
	}

	res, err := zahteviCollection.InsertOne(ctx, item)
	if err != nil {
		return nil, err
	}
	if id, ok := res.InsertedID.(primitive.ObjectID); ok {
		item.ID = id
	}
	return &item, nil
}

func processEnrollmentRequest(ctx context.Context, claims jwt.MapClaims, id primitive.ObjectID, action string) error {
	item, err := getRequestByID(ctx, id)
	if err != nil {
		return err
	}
	if item.Status != "na_cekanju" {
		return errors.New("Zahtev je vec obradjen")
	}

	status := "odbijen"
		if action == "odobri" {
			vrtic, err := getVrticByID(ctx, item.VrticID)
			if err != nil {
				return err
			}
			if slobodnaMesta(vrtic) <= 0 {
				return errors.New("Vrtic vise nema slobodnih mesta")
			}
			vrtic.TrenutnoUpisano++
			if err := updateVrtic(ctx, vrtic.ID, vrtic); err != nil {
				return err
			}
			status = "odobren"
		}

	now := time.Now()
	_, err = zahteviCollection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"status":       status,
		"processed_at": now,
		"processed_by": strings.ToLower(strings.TrimSpace(claimString(claims, "sub"))),
	}})
	return err
}

func getAllRequests(ctx context.Context) ([]UpisZahtev, error) {
	cursor, err := zahteviCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []UpisZahtev
	for cursor.Next(ctx) {
		var item UpisZahtev
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, cursor.Err()
}

func getRequestsByUser(ctx context.Context, email string) ([]UpisZahtev, error) {
	cursor, err := zahteviCollection.Find(ctx, bson.M{"korisnik_email": email}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []UpisZahtev
	for cursor.Next(ctx) {
		var item UpisZahtev
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, cursor.Err()
}

func getRequestByID(ctx context.Context, id primitive.ObjectID) (UpisZahtev, error) {
	var item UpisZahtev
	err := zahteviCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&item)
	return item, err
}

func buildOpstinaPDFReport(report []OpstinaIzvestaj) ([]byte, error) {
	lines := []string{
		"Izvestaj o kapacitetima po opstini",
		fmt.Sprintf("Datum: %s", time.Now().Format("2006-01-02 15:04")),
		"",
	}

	if len(report) == 0 {
		lines = append(lines, "Nema podataka za izvestaj.")
	} else {
		for _, row := range report {
			lines = append(lines, fmt.Sprintf(
				"%s | vrtici:%d | kapacitet:%d | upisano:%d | popunjenost:%.2f%%",
				row.Opstina,
				row.BrojVrtica,
				row.UkupanKapacitet,
				row.UkupnoUpisano,
				row.Popunjenost*100,
			))
		}
	}

	return buildSimplePDF(lines), nil
}

func buildSimplePDF(lines []string) []byte {
	var stream bytes.Buffer
	stream.WriteString("BT\n/F1 12 Tf\n50 760 Td\n")
	for i, line := range lines {
		if i > 0 {
			stream.WriteString("0 -16 Td\n")
		}
		stream.WriteString("(")
		stream.WriteString(escapePDFText(line))
		stream.WriteString(") Tj\n")
	}
	stream.WriteString("ET")

	content := stream.String()

	var pdf bytes.Buffer
	offsets := []int{0}
	writeObj := func(objNum int, objContent string) {
		offsets = append(offsets, pdf.Len())
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", objNum, objContent)
	}

	pdf.WriteString("%PDF-1.4\n")
	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>")
	writeObj(4, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	writeObj(5, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content))

	xrefPos := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n", len(offsets))
	pdf.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\n", len(offsets))
	fmt.Fprintf(&pdf, "startxref\n%d\n%%%%EOF", xrefPos)

	return pdf.Bytes()
}

func escapePDFText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

func toViews(vrtici []Vrtic) []VrticView {
	views := make([]VrticView, 0, len(vrtici))
	for _, v := range vrtici {
		views = append(views, VrticView{
			ID:              v.ID,
			Naziv:           v.Naziv,
			Tip:             v.Tip,
			Grad:            v.Grad,
			Opstina:         v.Opstina,
			MaxKapacitet:    v.MaxKapacitet,
			TrenutnoUpisano: v.TrenutnoUpisano,
			Popunjenost:     popunjenost(v),
			SlobodnaMesta:   slobodnaMesta(v),
			Kriticno:        popunjenost(v) >= 0.9,
			CreatedBy:       v.CreatedBy,
		})
	}
	return views
}

func popunjenost(v Vrtic) float64 {
	if v.MaxKapacitet <= 0 {
		return 0
	}
	return float64(v.TrenutnoUpisano) / float64(v.MaxKapacitet)
}

func slobodnaMesta(v Vrtic) int {
	if v.MaxKapacitet <= 0 {
		return 0
	}
	return v.MaxKapacitet - v.TrenutnoUpisano
}

func initMongo() {
	uri := getenvDefault("MONGO_URI", "mongodb://mongo:27017")
	dbName := getenvDefault("MONGO_DB", "euprava")
	collectionName := getenvDefault("MONGO_COLLECTION", "vrtici")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Mongo connect error: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Mongo ping error: %v", err)
	}

	db := client.Database(dbName)
	vrticiCollection = db.Collection(collectionName)
	zahteviCollection = db.Collection("zahtevi_upisa")

	ensureSeedData(ctx)
	ensureLegacyOwners(ctx)
	ensureRequestsIndexes(ctx)
}

func ensureSeedData(ctx context.Context) {
	count, err := vrticiCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Printf("Mongo count error: %v", err)
		return
	}
	if count > 0 {
		return
	}

	seed := []interface{}{
		Vrtic{Naziv: "Plavi Cuperak", Tip: "drzavni", Grad: "Beograd", Opstina: "Zvezdara", MaxKapacitet: 120, TrenutnoUpisano: 95, CreatedBy: "student@euprava.local"},
		Vrtic{Naziv: "Sumica", Tip: "privatni", Grad: "Beograd", Opstina: "Vozdovac", MaxKapacitet: 60, TrenutnoUpisano: 58, CreatedBy: "student@euprava.local"},
	}

	if _, err := vrticiCollection.InsertMany(ctx, seed); err != nil {
		log.Printf("Mongo seed insert error: %v", err)
	}
}

func ensureLegacyOwners(ctx context.Context) {
	_, err := vrticiCollection.UpdateMany(ctx, bson.M{
		"$or": []bson.M{{"created_by": bson.M{"$exists": false}}, {"created_by": ""}},
	}, bson.M{"$set": bson.M{"created_by": "student@euprava.local"}})
	if err != nil {
		log.Printf("Mongo owner backfill error: %v", err)
	}
}

func ensureRequestsIndexes(ctx context.Context) {
	_, err := zahteviCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "korisnik_email", Value: 1}}},
		{Keys: bson.D{{Key: "vrtic_owner", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	})
	if err != nil {
		log.Printf("Requests index warning: %v", err)
	}
}

func getAllVrtici(ctx context.Context) ([]Vrtic, error) {
	cursor, err := vrticiCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []Vrtic
	for cursor.Next(ctx) {
		var v Vrtic
		if err := cursor.Decode(&v); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, cursor.Err()
}

func getVrticByID(ctx context.Context, id primitive.ObjectID) (Vrtic, error) {
	var v Vrtic
	err := vrticiCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&v)
	return v, err
}

func insertVrtic(ctx context.Context, v Vrtic) error {
	_, err := vrticiCollection.InsertOne(ctx, v)
	return err
}

func updateVrtic(ctx context.Context, id primitive.ObjectID, v Vrtic) error {
	res, err := vrticiCollection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"naziv":            v.Naziv,
		"tip":              v.Tip,
		"grad":             v.Grad,
		"opstina":          v.Opstina,
		"max_kapacitet":    v.MaxKapacitet,
		"trenutno_upisano": v.TrenutnoUpisano,
		"created_by":       v.CreatedBy,
	}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func deleteVrtic(ctx context.Context, id primitive.ObjectID) error {
	_, _ = zahteviCollection.DeleteMany(ctx, bson.M{"vrtic_id": id, "status": bson.M{"$in": []string{"na_cekanju", "odbijen"}}})
	res, err := vrticiCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func requireAuth(r *http.Request) (jwt.MapClaims, error) {
	secret := getenvDefault("JWT_SECRET", "dev-secret")
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("Nedostaje Authorization header")
	}

	tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
	if tokenString == "" {
		return nil, errors.New("Neispravan token")
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Neispravan algoritam")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("Neispravan ili istekao token")
	}

	return claims, nil
}

func claimString(claims jwt.MapClaims, key string) string {
	value, ok := claims[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func getenvDefault(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}




