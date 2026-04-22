package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

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
	_, _ = zahteviCollection.DeleteMany(ctx, bson.M{"vrtic_id": id, "status": bson.M{"$in": []string{statusSubmitted, statusInReview, statusNeedDocs, statusWaitingList, statusRejected, "na_cekanju", "u_proveri"}}})
	_, _ = konkursiCollection.DeleteMany(ctx, bson.M{"vrtic_id": id})
	_, _ = rasporediCollection.DeleteMany(ctx, bson.M{"vrtic_id": id})
	_, _ = sastanciCollection.DeleteMany(ctx, bson.M{"vrtic_id": id})
	_, _ = obavestenjaCollection.DeleteMany(ctx, bson.M{"vrtic_id": id})
	res, err := vrticiCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
