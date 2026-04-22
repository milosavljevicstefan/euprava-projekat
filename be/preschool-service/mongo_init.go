package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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
	konkursiCollection = db.Collection("konkursi")
	rasporediCollection = db.Collection("rasporedi_vaspitaca")
	sastanciCollection = db.Collection("sastanci")
	obavestenjaCollection = db.Collection("obavestenja")

	ensureSeedData(ctx)
	ensureRequestsIndexes(ctx)
	ensureKonkursIndexes(ctx)
	ensureAssignmentsIndexes(ctx)
	ensureMeetingsIndexes(ctx)
	ensureNotificationsIndexes(ctx)
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

func ensureRequestsIndexes(ctx context.Context) {
	_, err := zahteviCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "korisnik_email", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "konkurs_id", Value: 1}}},
	})
	if err != nil {
		log.Printf("Requests index warning: %v", err)
	}
}

func ensureKonkursIndexes(ctx context.Context) {
	_, err := konkursiCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "vrtic_id", Value: 1}}},
		{Keys: bson.D{{Key: "aktivan", Value: 1}}},
		{Keys: bson.D{{Key: "datum_pocetka", Value: 1}}},
		{Keys: bson.D{{Key: "datum_zavrsetka", Value: 1}}},
	})
	if err != nil {
		log.Printf("Konkurs index warning: %v", err)
	}
}

func ensureAssignmentsIndexes(ctx context.Context) {
	_, err := rasporediCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "vaspitac_email", Value: 1}}},
		{Keys: bson.D{{Key: "vrtic_id", Value: 1}}},
		{Keys: bson.D{{Key: "vaspitac_email", Value: 1}, {Key: "vrtic_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
	if err != nil {
		log.Printf("Assignments index warning: %v", err)
	}
}

func ensureMeetingsIndexes(ctx context.Context) {
	_, err := sastanciCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "roditelj_email", Value: 1}}},
		{Keys: bson.D{{Key: "vaspitac_email", Value: 1}}},
		{Keys: bson.D{{Key: "zahtev_id", Value: 1}}},
		{Keys: bson.D{{Key: "termin", Value: 1}}},
	})
	if err != nil {
		log.Printf("Meetings index warning: %v", err)
	}
}

func ensureNotificationsIndexes(ctx context.Context) {
	_, err := obavestenjaCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "roditelj_email", Value: 1}}},
		{Keys: bson.D{{Key: "vaspitac_email", Value: 1}}},
		{Keys: bson.D{{Key: "zahtev_id", Value: 1}}},
	})
	if err != nil {
		log.Printf("Notifications index warning: %v", err)
	}
}
