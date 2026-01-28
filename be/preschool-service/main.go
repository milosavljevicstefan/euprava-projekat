package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Vrtic struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Naziv           string             `json:"naziv" bson:"naziv"`
	Tip             string             `json:"tip" bson:"tip"` // "drzavni" ili "privatni"
	Grad            string             `json:"grad" bson:"grad"`
	Opstina         string             `json:"opstina" bson:"opstina"`
	MaxKapacitet    int                `json:"max_kapacitet" bson:"max_kapacitet"`
	TrenutnoUpisano int                `json:"trenutno_upisano" bson:"trenutno_upisano"`
}

var vrticiCollection *mongo.Collection

func main() {
	initMongo()

	// Osnovni pozdrav
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Preschool servis (8081) je online.")
	})

	// Test podaci za kolegu
	http.HandleFunc("/vrtici", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			podaci, err := getAllVrtici(r.Context())
			if err != nil {
				http.Error(w, "Greska pri citanju iz baze", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(podaci)
		case http.MethodPost:
			var nov Vrtic
			if err := json.NewDecoder(r.Body).Decode(&nov); err != nil {
				http.Error(w, "Neispravan JSON", http.StatusBadRequest)
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

	fmt.Println("Preschool servis na 8081...")
	http.ListenAndServe(":8081", nil)
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
	ensureSeedData(ctx)
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
		Vrtic{
			Naziv:           "Plavi Cuperak",
			Tip:             "drzavni",
			Grad:            "Beograd",
			Opstina:         "Zvezdara",
			MaxKapacitet:    120,
			TrenutnoUpisano: 95,
		},
		Vrtic{
			Naziv:           "Sumica",
			Tip:             "privatni",
			Grad:            "Beograd",
			Opstina:         "Vozdovac",
			MaxKapacitet:    60,
			TrenutnoUpisano: 58,
		},
	}

	if _, err := vrticiCollection.InsertMany(ctx, seed); err != nil {
		log.Printf("Mongo seed insert error: %v", err)
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

func insertVrtic(ctx context.Context, v Vrtic) error {
	_, err := vrticiCollection.InsertOne(ctx, v)
	return err
}

func getenvDefault(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
