package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OcenaVrtica struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	VrticID       primitive.ObjectID `json:"vrtic_id" bson:"vrtic_id"`
	KorisnikEmail string             `json:"korisnik_email" bson:"korisnik_email"`
	Ocena         int                `json:"ocena" bson:"ocena"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
}

type RatingRequest struct {
	VrticID string `json:"vrtic_id"`
	Ocena   int    `json:"ocena"`
}

type RatingSummary struct {
	VrticID       primitive.ObjectID `json:"vrtic_id"`
	ProsecnaOcena float64            `json:"prosecna_ocena"`
	BrojOcena     int                `json:"broj_ocena"`
}

var oceneCollection *mongo.Collection
var ratingsIndexesOnce sync.Once

func init() {
	http.HandleFunc("/ocene", handleOcene)
}

func handleOcene(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {
	case http.MethodGet:
		items, err := getRatingsSummary(r.Context())
		if err != nil {
			http.Error(w, "Greska pri citanju ocena", http.StatusInternalServerError)
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
		if err := requireUserRole(claims); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		var req RatingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Neispravan JSON", http.StatusBadRequest)
			return
		}

		if err := upsertRating(r.Context(), claims, req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func ratingsColl() *mongo.Collection {
	if oceneCollection == nil && vrticiCollection != nil {
		oceneCollection = vrticiCollection.Database().Collection("ocene_vrtica")
	}
	return oceneCollection
}

func ensureRatingsIndexes(ctx context.Context) {
	ratingsIndexesOnce.Do(func() {
		coll := ratingsColl()
		if coll == nil {
			return
		}
		_, err := coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
			{Keys: bson.D{{Key: "vrtic_id", Value: 1}}},
			{Keys: bson.D{{Key: "korisnik_email", Value: 1}}, Options: options.Index().SetUnique(true).SetName("unique_user_vrtic_rating")},
		})
		if err != nil {
			log.Printf("Ratings index warning: %v", err)
		}
	})
}

func validateRatingRequest(req RatingRequest) error {
	if strings.TrimSpace(req.VrticID) == "" {
		return errors.New("Vrtic je obavezan")
	}
	if req.Ocena < 1 || req.Ocena > 5 {
		return errors.New("Ocena mora biti izmedju 1 i 5")
	}
	return nil
}

func upsertRating(ctx context.Context, claims jwt.MapClaims, req RatingRequest) error {
	if err := validateRatingRequest(req); err != nil {
		return err
	}
	coll := ratingsColl()
	if coll == nil {
		return errors.New("Kolekcija ocena nije dostupna")
	}
	ensureRatingsIndexes(ctx)

	vrticID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.VrticID))
	if err != nil {
		return errors.New("Neispravan ID vrtica")
	}
	if _, err := getVrticByID(ctx, vrticID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("Vrtic nije pronadjen")
		}
		return err
	}

	email := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
	if email == "" {
		return errors.New("Neispravan token")
	}

	now := time.Now()
	_, err = coll.UpdateOne(ctx,
		bson.M{"vrtic_id": vrticID, "korisnik_email": email},
		bson.M{"$set": bson.M{"ocena": req.Ocena, "updated_at": now}, "$setOnInsert": bson.M{"created_at": now}},
		options.Update().SetUpsert(true),
	)
	return err
}

func getRatingsSummary(ctx context.Context) ([]RatingSummary, error) {
	coll := ratingsColl()
	if coll == nil {
		return []RatingSummary{}, nil
	}
	ensureRatingsIndexes(ctx)

	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	type agg struct {
		total int
		count int
	}
	byVrtic := map[primitive.ObjectID]*agg{}
	for cursor.Next(ctx) {
		var item OcenaVrtica
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		entry := byVrtic[item.VrticID]
		if entry == nil {
			entry = &agg{}
			byVrtic[item.VrticID] = entry
		}
		entry.total += item.Ocena
		entry.count++
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	result := make([]RatingSummary, 0, len(byVrtic))
	for vrticID, entry := range byVrtic {
		result = append(result, RatingSummary{VrticID: vrticID, ProsecnaOcena: float64(entry.total) / float64(entry.count), BrojOcena: entry.count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].VrticID.Hex() < result[j].VrticID.Hex() })
	return result, nil
}
