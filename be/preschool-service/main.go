package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
}

type OpstinaIzvestaj struct {
	Opstina         string  `json:"opstina"`
	BrojVrtica      int     `json:"broj_vrtica"`
	UkupanKapacitet int     `json:"ukupan_kapacitet"`
	UkupnoUpisano   int     `json:"ukupno_upisano"`
	Popunjenost     float64 `json:"popunjenost"`
}

type User struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email        string             `json:"email" bson:"email"`
	Role         string             `json:"role" bson:"role"`
	PasswordHash string             `json:"-" bson:"password_hash"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}

type ProfileResponse struct {
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

var vrticiCollection *mongo.Collection
var usersCollection *mongo.Collection

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

	http.HandleFunc("/auth/register", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Neispravan JSON", http.StatusBadRequest)
			return
		}

		role, err := normalizeRole(req.Role)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := registerUser(r.Context(), req.Email, req.Password, role); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	http.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Neispravan JSON", http.StatusBadRequest)
			return
		}

		user, err := authenticate(r.Context(), req.Email, req.Password)
		if err != nil {
			http.Error(w, "Neispravni kredencijali", http.StatusUnauthorized)
			return
		}

		token, exp, err := issueToken(user.Email, user.Role)
		if err != nil {
			http.Error(w, "Greska pri generisanju tokena", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			AccessToken: token,
			TokenType:   "Bearer",
			ExpiresIn:   exp,
			Email:       user.Email,
			Role:        user.Role,
		})
	})

	http.HandleFunc("/auth/profile", func(w http.ResponseWriter, r *http.Request) {
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

		email := claimString(claims, "sub")
		if email == "" {
			http.Error(w, "Neispravan token", http.StatusUnauthorized)
			return
		}

		user, err := getUserByEmail(r.Context(), email)
		if err != nil {
			http.Error(w, "Korisnik nije pronadjen", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ProfileResponse{
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		})
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
			if _, err := requireAuth(r); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
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
			if _, err := requireAuth(r); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
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
			if _, err := requireAuth(r); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
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
	return nil
}

func normalizeRole(role string) (string, error) {
	r := strings.ToLower(strings.TrimSpace(role))
	switch r {
	case "", "korisnik":
		return "korisnik", nil
	case "sluzbenik":
		return "sluzbenik", nil
	case "admin":
		return "admin", nil
	default:
		return "", errors.New("Neispravna rola (korisnik, sluzbenik, admin)")
	}
}

func handleVrticiList(r *http.Request) ([]VrticView, error) {
	all, err := getAllVrtici(r.Context())
	if err != nil {
		return nil, err
	}

	tip := r.URL.Query().Get("tip")
	sortBy := r.URL.Query().Get("sort")

	var filtered []Vrtic
	for _, v := range all {
		if tip == "" || v.Tip == tip {
			filtered = append(filtered, v)
		}
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

	vrticiCollection = client.Database(dbName).Collection(collectionName)
	usersCollection = client.Database(dbName).Collection("users")

	if _, err := usersCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		log.Printf("Users index warning: %v", err)
	}

	ensureSeedData(ctx)
	ensureSeedUser(ctx)
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
		Vrtic{Naziv: "Plavi Cuperak", Tip: "drzavni", Grad: "Beograd", Opstina: "Zvezdara", MaxKapacitet: 120, TrenutnoUpisano: 95},
		Vrtic{Naziv: "Sumica", Tip: "privatni", Grad: "Beograd", Opstina: "Vozdovac", MaxKapacitet: 60, TrenutnoUpisano: 58},
	}

	if _, err := vrticiCollection.InsertMany(ctx, seed); err != nil {
		log.Printf("Mongo seed insert error: %v", err)
	}
}

func ensureSeedUser(ctx context.Context) {
	count, err := usersCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Printf("Mongo users count error: %v", err)
		return
	}
	if count > 0 {
		return
	}
	_ = registerUser(ctx, "student@euprava.local", "demo123", "admin")
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
	res, err := vrticiCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func registerUser(ctx context.Context, email, password, role string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return errors.New("Email i lozinka su obavezni")
	}

	exists, err := usersCollection.CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		return err
	}
	if exists > 0 {
		return errors.New("Korisnik vec postoji")
	}

	_, err = usersCollection.InsertOne(ctx, User{
		Email:        email,
		Role:         role,
		PasswordHash: hashPassword(email, password),
		CreatedAt:    time.Now(),
	})
	return err
}

func authenticate(ctx context.Context, email, password string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	var user User
	if err := usersCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user); err != nil {
		return nil, err
	}
	if user.PasswordHash != hashPassword(email, password) {
		return nil, errors.New("pogresna lozinka")
	}
	return &user, nil
}

func getUserByEmail(ctx context.Context, email string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	var user User
	if err := usersCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func hashPassword(email, password string) string {
	salt := getenvDefault("AUTH_SALT", "dev-salt")
	val := sha256.Sum256([]byte(email + ":" + password + ":" + salt))
	return hex.EncodeToString(val[:])
}

func issueToken(email, role string) (string, int64, error) {
	secret := getenvDefault("JWT_SECRET", "dev-secret")
	exp := time.Now().Add(2 * time.Hour).Unix()

	claims := jwt.MapClaims{
		"sub":  email,
		"role": role,
		"exp":  exp,
		"iss":  "preschool-service",
		"aud":  "frontend",
		"jti":  tokenID(email),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", 0, err
	}
	return signed, exp, nil
}

func tokenID(email string) string {
	s := sha256.Sum256([]byte(email + time.Now().String()))
	return hex.EncodeToString(s[:])
}

func claimString(claims jwt.MapClaims, key string) string {
	v, ok := claims[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
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
