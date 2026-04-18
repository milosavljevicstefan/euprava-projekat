package main

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
	"time"
)

func getKonkursByID(ctx context.Context, id primitive.ObjectID) (Konkurs, error) {
	var item Konkurs
	err := konkursiCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&item)
	return item, err
}

func getActiveKonkursByVrticID(ctx context.Context, vrticID primitive.ObjectID) (Konkurs, error) {
	var item Konkurs
	now := time.Now()
	err := konkursiCollection.FindOne(ctx, bson.M{
		"vrtic_id":        vrticID,
		"aktivan":         true,
		"datum_pocetka":   bson.M{"$lte": now},
		"datum_zavrsetka": bson.M{"$gte": now},
	}, options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})).Decode(&item)
	return item, err
}

func countApprovedRequestsForKonkurs(ctx context.Context, konkursID primitive.ObjectID) (int, error) {
	count, err := zahteviCollection.CountDocuments(ctx, bson.M{"konkurs_id": konkursID, "status": "odobren"})
	return int(count), err
}

func createKonkurs(ctx context.Context, req KonkursRequest) (*KonkursView, error) {
	if err := validateKonkursInput(req); err != nil {
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

	pocetak, err := parseDateValue(req.DatumPocetka, false)
	if err != nil {
		return nil, err
	}
	zavrsetak, err := parseDateValue(req.DatumZavrsetka, true)
	if err != nil {
		return nil, err
	}
	if !zavrsetak.After(pocetak) {
		return nil, errors.New("Datum zavrsetka mora biti posle datuma pocetka")
	}
	if req.MaxMesta > slobodnaMesta(vrtic) {
		return nil, errors.New("Max mesta na konkursu ne moze biti vece od trenutno slobodnih mesta u vrticu")
	}

	now := time.Now()
	_, err = konkursiCollection.UpdateMany(ctx, bson.M{"vrtic_id": vrticID, "aktivan": true}, bson.M{"$set": bson.M{"aktivan": false, "closed_at": now}})
	if err != nil {
		return nil, err
	}

	item := Konkurs{
		VrticID:        vrticID,
		DatumPocetka:   pocetak,
		DatumZavrsetka: zavrsetak,
		MaxMesta:       req.MaxMesta,
		Aktivan:        true,
		CreatedAt:      now,
	}
	res, err := konkursiCollection.InsertOne(ctx, item)
	if err != nil {
		return nil, err
	}
	if id, ok := res.InsertedID.(primitive.ObjectID); ok {
		item.ID = id
	}

	view := KonkursView{
		ID:             item.ID,
		VrticID:        item.VrticID,
		VrticNaziv:     vrtic.Naziv,
		DatumPocetka:   item.DatumPocetka,
		DatumZavrsetka: item.DatumZavrsetka,
		MaxMesta:       item.MaxMesta,
		Aktivan:        item.Aktivan,
		Status:         konkursStatusLabel(item, now),
		Popunjeno:      0,
		SlobodnaMesta:  item.MaxMesta,
	}
	return &view, nil
}

func closeKonkurs(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	res, err := konkursiCollection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"aktivan": false, "closed_at": now}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func getAllKonkursViews(ctx context.Context, statusFilter string, vrticIDRaw string) ([]KonkursView, error) {
	filter := bson.M{}
	if strings.TrimSpace(vrticIDRaw) != "" {
		vrticID, err := primitive.ObjectIDFromHex(strings.TrimSpace(vrticIDRaw))
		if err != nil {
			return nil, errors.New("Neispravan ID vrtica")
		}
		filter["vrtic_id"] = vrticID
	}

	cursor, err := konkursiCollection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	vrtici, err := getAllVrtici(ctx)
	if err != nil {
		return nil, err
	}
	vrticMap := map[primitive.ObjectID]Vrtic{}
	for _, v := range vrtici {
		vrticMap[v.ID] = v
	}

	now := time.Now()
	result := make([]KonkursView, 0)
	for cursor.Next(ctx) {
		var item Konkurs
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		status := konkursStatusLabel(item, now)
		if statusFilter != "" && status != statusFilter {
			continue
		}
		approved, err := countApprovedRequestsForKonkurs(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		vrtic := vrticMap[item.VrticID]
		slobodno := item.MaxMesta - approved
		if slobodno < 0 {
			slobodno = 0
		}
		result = append(result, KonkursView{
			ID:             item.ID,
			VrticID:        item.VrticID,
			VrticNaziv:     vrtic.Naziv,
			DatumPocetka:   item.DatumPocetka,
			DatumZavrsetka: item.DatumZavrsetka,
			MaxMesta:       item.MaxMesta,
			Aktivan:        item.Aktivan,
			Status:         status,
			Popunjeno:      approved,
			SlobodnaMesta:  slobodno,
		})
	}
	return result, cursor.Err()
}
