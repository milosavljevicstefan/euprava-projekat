package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

type UserListItem struct {
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

var usersCollection *mongo.Collection

func main() {
	initMongo()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		fmt.Fprint(w, "Auth servis (8083) je online.")
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

		if err := registerUser(r.Context(), req.Email, req.Password, "roditelj"); err != nil {
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

	http.HandleFunc("/auth/users", func(w http.ResponseWriter, r *http.Request) {
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
		role := strings.ToLower(strings.TrimSpace(claimString(claims, "role")))
		if role != "admin" {
			http.Error(w, "Samo admin moze da pregleda korisnike", http.StatusForbidden)
			return
		}

		roleFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("role")))
		filter := bson.M{}
		if roleFilter != "" {
			filter["role"] = roleFilter
		}
		items, err := listUsers(r.Context(), filter)
		if err != nil {
			http.Error(w, "Greska pri citanju korisnika", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	fmt.Println("Auth servis na 8083...")
	http.ListenAndServe(":8083", nil)
}

func normalizeRole(role string) (string, error) {
	r := strings.ToLower(strings.TrimSpace(role))
	switch r {
	case "", "roditelj", "korisnik":
		return "roditelj", nil
	case "admin":
		return "admin", nil
	case "vaspitac":
		return "vaspitac", nil
	default:
		return "", errors.New("Neispravna rola (roditelj, admin, vaspitac)")
	}
}

func registerUser(ctx context.Context, email, password, role string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return errors.New("Email i lozinka su obavezni")
	}
	if len(password) < 6 {
		return errors.New("Lozinka mora imati najmanje 6 karaktera")
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

func listUsers(ctx context.Context, filter bson.M) ([]UserListItem, error) {
	cursor, err := usersCollection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "email", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]UserListItem, 0)
	for cursor.Next(ctx) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		items = append(items, UserListItem{
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		})
	}
	return items, cursor.Err()
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
		"iss":  "auth-service",
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

func initMongo() {
	uri := getenvDefault("MONGO_URI", "mongodb://mongo:27017")
	dbName := getenvDefault("MONGO_DB", "euprava")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Mongo connect error: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Mongo ping error: %v", err)
	}

	usersCollection = client.Database(dbName).Collection("users")

	_, err = usersCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Printf("Users index warning: %v", err)
	}

	ensureRoleMigration(ctx)
	ensureSeedUser(ctx)
}

func ensureRoleMigration(ctx context.Context) {
	_, err := usersCollection.UpdateMany(ctx, bson.M{"role": "sluzbenik"}, bson.M{"$set": bson.M{"role": "admin"}})
	if err != nil {
		log.Printf("Users role migration warning: %v", err)
	}
	_, err = usersCollection.UpdateMany(ctx, bson.M{"role": "korisnik"}, bson.M{"$set": bson.M{"role": "roditelj"}})
	if err != nil {
		log.Printf("Users role migration warning: %v", err)
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
	_ = registerUser(ctx, "roditelj@euprava.local", "demo123", "roditelj")
	_ = registerUser(ctx, "vaspitac@euprava.local", "demo123", "vaspitac")
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
