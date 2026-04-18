package main

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"sort"
	"strings"
	"time"
)

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

	konkurs, err := getActiveKonkursByVrticID(ctx, vrticID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("Za izabrani vrtic trenutno nema aktivnog konkursa")
		}
		return nil, err
	}

	korisnikEmail := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
	if korisnikEmail == "" {
		return nil, errors.New("Neispravan token")
	}

	blockingStatuses := append([]string{}, activeRequestStatuses...)
	blockingStatuses = append(blockingStatuses, "na_cekanju", "u_proveri")
	exists, err := zahteviCollection.CountDocuments(ctx, bson.M{
		"vrtic_id":       vrticID,
		"korisnik_email": korisnikEmail,
		"ime_deteta":     strings.TrimSpace(req.ImeDeteta),
		"status":         bson.M{"$in": blockingStatuses},
	})
	if err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, errors.New("Vec postoji aktivan zahtev za ovo dete u izabranom vrticu")
	}

	status := statusSubmitted
	reason := ""
	if slobodnaMesta(vrtic) <= 0 {
		status = statusWaitingList
		reason = "Trenutno nema slobodnih mesta. Zahtev je dodat na listu cekanja."
	}

	approvedForKonkurs, err := countApprovedRequestsForKonkurs(ctx, konkurs.ID)
	if err != nil {
		return nil, err
	}
	if approvedForKonkurs >= konkurs.MaxMesta {
		status = statusWaitingList
		reason = "Konkurs je trenutno popunjen. Zahtev je dodat na listu cekanja."
	}

	item := UpisZahtev{
		VrticID:    vrticID,
		KonkursID:  konkurs.ID,
		VrticNaziv: vrtic.Naziv,

		ImeRoditelja:         strings.TrimSpace(req.ImeRoditelja),
		ImeDeteta:            strings.TrimSpace(req.ImeDeteta),
		BrojGodina:           req.BrojGodina,
		KorisnikEmail:        korisnikEmail,
		PotvrdaVakcinacije:   req.PotvrdaVakcinacije,
		IzvodIzMaticneKnjige: req.IzvodIzMaticneKnjige,
		Status:               status,
		CreatedAt:            time.Now(),
		Reason:               reason,
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

func processEnrollmentRequest(ctx context.Context, claims jwt.MapClaims, id primitive.ObjectID, action string, reason string) error {
	item, err := getRequestByID(ctx, id)
	if err != nil {
		return err
	}

	current := canonicalRequestStatus(item.Status)
	reason = strings.TrimSpace(reason)

	switch action {
	case "obrada":
		if current != statusSubmitted && current != statusNeedDocs && current != statusWaitingList {
			return errors.New("Samo podnet, vracen ili cekajuci zahtev moze da predje u obradu")
		}
		return updateRequestStatus(ctx, id, claims, statusInReview, "")
	case "dopuna":
		if current == statusApproved || current == statusRejected {
			return errors.New("Zahtev je vec zavrsen")
		}
		if reason == "" {
			return errors.New("Unesite sta nedostaje u dokumentaciji")
		}
		return updateRequestStatus(ctx, id, claims, statusNeedDocs, reason)
	case "odbij":
		if current == statusApproved || current == statusRejected {
			return errors.New("Zahtev je vec zavrsen")
		}
		if reason == "" {
			return errors.New("Unesite razlog odbijanja")
		}
		return updateRequestStatus(ctx, id, claims, statusRejected, reason)
	case "odobri":
		if current == statusApproved || current == statusRejected {
			return errors.New("Zahtev je vec zavrsen")
		}

		vrtic, err := getVrticByID(ctx, item.VrticID)
		if err != nil {
			return err
		}
		if slobodnaMesta(vrtic) <= 0 {
			return updateRequestStatus(ctx, id, claims, statusWaitingList, "Trenutno nema slobodnih mesta. Zahtev ostaje na listi cekanja.")
		}
		if !item.KonkursID.IsZero() {
			konkurs, err := getKonkursByID(ctx, item.KonkursID)
			if err != nil {
				return err
			}
			approvedForKonkurs, err := countApprovedRequestsForKonkurs(ctx, konkurs.ID)
			if err != nil {
				return err
			}
			if approvedForKonkurs >= konkurs.MaxMesta {
				return updateRequestStatus(ctx, id, claims, statusWaitingList, "Konkurs je trenutno popunjen. Zahtev ostaje na listi cekanja.")
			}
		}

		vrtic.TrenutnoUpisano++
		if err := updateVrtic(ctx, vrtic.ID, vrtic); err != nil {
			return err
		}
		return updateRequestStatus(ctx, id, claims, statusApproved, "")
	default:
		return errors.New("Nepoznata akcija")
	}
}
func getAllRequests(ctx context.Context) ([]UpisZahtev, error) {
	cursor, err := zahteviCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make([]UpisZahtev, 0)
	for cursor.Next(ctx) {
		var item UpisZahtev
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		item.Status = canonicalRequestStatus(item.Status)
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

	result := make([]UpisZahtev, 0)
	for cursor.Next(ctx) {
		var item UpisZahtev
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		item.Status = canonicalRequestStatus(item.Status)
		result = append(result, item)
	}
	return result, cursor.Err()
}

func getRequestByID(ctx context.Context, id primitive.ObjectID) (UpisZahtev, error) {
	var item UpisZahtev
	err := zahteviCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&item)
	if err == nil {
		item.Status = canonicalRequestStatus(item.Status)
	}
	return item, err
}

func updateRequestStatus(ctx context.Context, id primitive.ObjectID, claims jwt.MapClaims, status string, reason string) error {
	now := time.Now()
	payload := bson.M{
		"status":       canonicalRequestStatus(status),
		"processed_at": now,
		"processed_by": strings.ToLower(strings.TrimSpace(claimString(claims, "sub"))),
	}
	reason = strings.TrimSpace(reason)
	update := bson.M{"$set": payload}
	if reason != "" {
		payload["reason"] = reason
	} else {
		update["$unset"] = bson.M{"reason": ""}
	}
	_, err := zahteviCollection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func updateRequestDocuments(ctx context.Context, claims jwt.MapClaims, id primitive.ObjectID, payload DokumentaUpdateRequest) error {
	item, err := getRequestByID(ctx, id)
	if err != nil {
		return err
	}
	email := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
	if email == "" || email != strings.ToLower(strings.TrimSpace(item.KorisnikEmail)) {
		return errors.New("Nemate dozvolu da dopunite ovu dokumentaciju")
	}
	if canonicalRequestStatus(item.Status) != statusNeedDocs {
		return errors.New("Dokumentacija se moze dopuniti samo kada je zahtev vracen na dopunu")
	}
	if !payload.PotvrdaVakcinacije || !payload.IzvodIzMaticneKnjige {
		return errors.New("Obe stavke dokumentacije moraju biti prilozene")
	}
	_, err = zahteviCollection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"potvrda_vakcinacije":     true,
			"izvod_iz_maticne_knjige": true,
			"status":                  statusSubmitted,
		},
		"$unset": bson.M{
			"processed_at": "",
			"processed_by": "",
			"reason":       "",
		},
	})
	return err
}

func updateEnrollmentRequest(ctx context.Context, claims jwt.MapClaims, id primitive.ObjectID, req UpisRequest) (*UpisZahtev, error) {
	if err := validateEnrollmentInput(req); err != nil {
		return nil, err
	}

	item, err := getRequestByID(ctx, id)
	if err != nil {
		return nil, err
	}

	email := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
	if email == "" || email != strings.ToLower(strings.TrimSpace(item.KorisnikEmail)) {
		return nil, errors.New("Nemate dozvolu da izmenite ovaj zahtev")
	}
	if canonicalRequestStatus(item.Status) != statusNeedDocs {
		return nil, errors.New("Zahtev moze da se izmeni samo kada je vracen na dopunu")
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

	konkurs, err := getActiveKonkursByVrticID(ctx, vrticID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("Za izabrani vrtic trenutno nema aktivnog konkursa")
		}
		return nil, err
	}

	blockingStatuses := append([]string{}, activeRequestStatuses...)
	blockingStatuses = append(blockingStatuses, statusSubmitted, statusWaitingList, "u_proveri")
	exists, err := zahteviCollection.CountDocuments(ctx, bson.M{
		"_id":            bson.M{"$ne": id},
		"vrtic_id":       vrticID,
		"korisnik_email": email,
		"ime_deteta":     strings.TrimSpace(req.ImeDeteta),
		"status":         bson.M{"$in": blockingStatuses},
	})
	if err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, errors.New("Vec postoji aktivan zahtev za ovo dete u izabranom vrticu")
	}

	status := statusSubmitted
	reason := ""
	if slobodnaMesta(vrtic) <= 0 {
		status = statusWaitingList
		reason = "Trenutno nema slobodnih mesta. Zahtev je dodat na listu cekanja."
	}

	approvedForKonkurs, err := countApprovedRequestsForKonkurs(ctx, konkurs.ID)
	if err != nil {
		return nil, err
	}
	if approvedForKonkurs >= konkurs.MaxMesta {
		status = statusWaitingList
		reason = "Konkurs je trenutno popunjen. Zahtev je dodat na listu cekanja."
	}

	update := bson.M{
		"$set": bson.M{
			"vrtic_id":                  vrticID,
			"konkurs_id":                konkurs.ID,
			"vrtic_naziv":               vrtic.Naziv,
			"ime_roditelja":             strings.TrimSpace(req.ImeRoditelja),
			"ime_deteta":                strings.TrimSpace(req.ImeDeteta),
			"broj_godina":               req.BrojGodina,
			"potvrda_vakcinacije":       req.PotvrdaVakcinacije,
			"izvod_iz_maticne_knjige":   req.IzvodIzMaticneKnjige,
			"status":                    status,
			"reason":                    reason,
		},
		"$unset": bson.M{
			"processed_at": "",
			"processed_by": "",
		},
	}

	if reason == "" {
		update["$unset"].(bson.M)["reason"] = ""
		delete(update["$set"].(bson.M), "reason")
	}

	if _, err := zahteviCollection.UpdateOne(ctx, bson.M{"_id": id}, update); err != nil {
		return nil, err
	}

	updated, err := getRequestByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func createAssignment(ctx context.Context, req VaspitacRasporedRequest) (*VaspitacRaspored, error) {
	if err := validateAssignmentInput(req); err != nil {
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
	email := strings.ToLower(strings.TrimSpace(req.VaspitacEmail))
	exists, err := rasporediCollection.CountDocuments(ctx, bson.M{"vrtic_id": vrticID, "vaspitac_email": email})
	if err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, errors.New("Vaspitac je vec rasporedjen u izabrani vrtic")
	}
	item := VaspitacRaspored{
		VrticID:       vrticID,
		VrticNaziv:    vrtic.Naziv,
		VaspitacEmail: email,
		CreatedAt:     time.Now(),
	}
	res, err := rasporediCollection.InsertOne(ctx, item)
	if err != nil {
		return nil, err
	}
	if id, ok := res.InsertedID.(primitive.ObjectID); ok {
		item.ID = id
	}
	return &item, nil
}

func listAssignments(ctx context.Context) ([]VaspitacRaspored, error) {
	cursor, err := rasporediCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := make([]VaspitacRaspored, 0)
	for cursor.Next(ctx) {
		var item VaspitacRaspored
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, cursor.Err()
}

func deleteAssignment(ctx context.Context, id primitive.ObjectID) error {
	res, err := rasporediCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func getAssignmentsByVrtic(ctx context.Context, vrticID primitive.ObjectID) ([]VaspitacRaspored, error) {
	cursor, err := rasporediCollection.Find(ctx, bson.M{"vrtic_id": vrticID}, options.Find().SetSort(bson.D{{Key: "vaspitac_email", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := make([]VaspitacRaspored, 0)
	for cursor.Next(ctx) {
		var item VaspitacRaspored
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, cursor.Err()
}

func getAssignmentsByEducator(ctx context.Context, email string) ([]VaspitacRaspored, error) {
	cursor, err := rasporediCollection.Find(ctx, bson.M{"vaspitac_email": strings.ToLower(strings.TrimSpace(email))}, options.Find().SetSort(bson.D{{Key: "vrtic_naziv", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := make([]VaspitacRaspored, 0)
	for cursor.Next(ctx) {
		var item VaspitacRaspored
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, cursor.Err()
}

func getRoditeljVaspitaci(ctx context.Context, email string) ([]RoditeljVaspitaciView, error) {
	requests, err := getRequestsByUser(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, err
	}
	result := make([]RoditeljVaspitaciView, 0)
	for _, item := range requests {
		if canonicalRequestStatus(item.Status) != statusApproved {
			continue
		}
		assignments, err := getAssignmentsByVrtic(ctx, item.VrticID)
		if err != nil {
			return nil, err
		}
		if len(assignments) == 0 {
			continue
		}
		emails := make([]string, 0, len(assignments))
		for _, assignment := range assignments {
			emails = append(emails, assignment.VaspitacEmail)
		}
		result = append(result, RoditeljVaspitaciView{
			ZahtevID:   item.ID,
			VrticID:    item.VrticID,
			VrticNaziv: item.VrticNaziv,
			ImeDeteta:  item.ImeDeteta,
			Vaspitaci:  emails,
		})
	}
	return result, nil
}

func parseMeetingTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("Termin sastanka je obavezan")
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02T15:04", raw); err == nil {
		return t, nil
	}
	return time.Time{}, errors.New("Neispravan format termina")
}

func educatorAssignedToVrtic(ctx context.Context, email string, vrticID primitive.ObjectID) (bool, error) {
	count, err := rasporediCollection.CountDocuments(ctx, bson.M{
		"vaspitac_email": strings.ToLower(strings.TrimSpace(email)),
		"vrtic_id":       vrticID,
	})
	return count > 0, err
}

func createMeeting(ctx context.Context, claims jwt.MapClaims, req SastanakRequest) (*Sastanak, error) {
	if err := validateMeetingInput(req); err != nil {
		return nil, err
	}
	zahtevID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.ZahtevID))
	if err != nil {
		return nil, errors.New("Neispravan zahtev")
	}
	item, err := getRequestByID(ctx, zahtevID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("Zahtev nije pronadjen")
		}
		return nil, err
	}
	parentEmail := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
	if parentEmail == "" || parentEmail != strings.ToLower(strings.TrimSpace(item.KorisnikEmail)) {
		return nil, errors.New("Nemate dozvolu za zakazivanje ovog sastanka")
	}
	if canonicalRequestStatus(item.Status) != statusApproved {
		return nil, errors.New("Sastanak se moze zakazati samo za odobren upis")
	}
	educatorEmail := strings.ToLower(strings.TrimSpace(req.VaspitacEmail))
	allowed, err := educatorAssignedToVrtic(ctx, educatorEmail, item.VrticID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("Izabrani vaspitac nije rasporedjen u vrtic deteta")
	}
	termin, err := parseMeetingTime(req.Termin)
	if err != nil {
		return nil, err
	}
	if !termin.After(time.Now()) {
		return nil, errors.New("Termin mora biti u buducnosti")
	}
	meeting := Sastanak{
		ZahtevID:      item.ID,
		VrticID:       item.VrticID,
		VrticNaziv:    item.VrticNaziv,
		ImeDeteta:     item.ImeDeteta,
		RoditeljEmail: parentEmail,
		VaspitacEmail: educatorEmail,
		Termin:        termin,
		Napomena:      strings.TrimSpace(req.Napomena),
		Status:        "zakazan",
		CreatedAt:     time.Now(),
	}
	res, err := sastanciCollection.InsertOne(ctx, meeting)
	if err != nil {
		return nil, err
	}
	if id, ok := res.InsertedID.(primitive.ObjectID); ok {
		meeting.ID = id
	}
	return &meeting, nil
}

func getMeetingsByParent(ctx context.Context, email string) ([]Sastanak, error) {
	cursor, err := sastanciCollection.Find(ctx, bson.M{"roditelj_email": strings.ToLower(strings.TrimSpace(email))}, options.Find().SetSort(bson.D{{Key: "termin", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := make([]Sastanak, 0)
	for cursor.Next(ctx) {
		var item Sastanak
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, cursor.Err()
}

func getMeetingsByEducator(ctx context.Context, email string) ([]Sastanak, error) {
	cursor, err := sastanciCollection.Find(ctx, bson.M{"vaspitac_email": strings.ToLower(strings.TrimSpace(email))}, options.Find().SetSort(bson.D{{Key: "termin", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := make([]Sastanak, 0)
	for cursor.Next(ctx) {
		var item Sastanak
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, cursor.Err()
}

func getEducatorChildren(ctx context.Context, email string) ([]UpisZahtev, error) {
	assignments, err := getAssignmentsByEducator(ctx, email)
	if err != nil {
		return nil, err
	}
	if len(assignments) == 0 {
		return []UpisZahtev{}, nil
	}
	ids := make([]primitive.ObjectID, 0, len(assignments))
	for _, item := range assignments {
		ids = append(ids, item.VrticID)
	}
	cursor, err := zahteviCollection.Find(ctx, bson.M{
		"vrtic_id": bson.M{"$in": ids},
		"status":   statusApproved,
	}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := make([]UpisZahtev, 0)
	for cursor.Next(ctx) {
		var item UpisZahtev
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		item.Status = canonicalRequestStatus(item.Status)
		items = append(items, item)
	}
	return items, cursor.Err()
}

func createSymptomsNotification(ctx context.Context, claims jwt.MapClaims, req SimptomObavestenjeRequest) (*SimptomObavestenje, error) {
	if err := validateSymptomsInput(req); err != nil {
		return nil, err
	}
	zahtevID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.ZahtevID))
	if err != nil {
		return nil, errors.New("Neispravan zahtev")
	}
	item, err := getRequestByID(ctx, zahtevID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("Zahtev nije pronadjen")
		}
		return nil, err
	}
	if canonicalRequestStatus(item.Status) != statusApproved {
		return nil, errors.New("Obavestenje se moze poslati samo za odobren upis")
	}
	educatorEmail := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
	allowed, err := educatorAssignedToVrtic(ctx, educatorEmail, item.VrticID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("Nemate dozvolu da saljete obavestenja za ovo dete")
	}
	notice := SimptomObavestenje{
		ZahtevID:      item.ID,
		VrticID:       item.VrticID,
		VrticNaziv:    item.VrticNaziv,
		ImeDeteta:     item.ImeDeteta,
		RoditeljEmail: item.KorisnikEmail,
		VaspitacEmail: educatorEmail,
		Poruka:        strings.TrimSpace(req.Poruka),
		CreatedAt:     time.Now(),
	}
	res, err := obavestenjaCollection.InsertOne(ctx, notice)
	if err != nil {
		return nil, err
	}
	if id, ok := res.InsertedID.(primitive.ObjectID); ok {
		notice.ID = id
	}
	return &notice, nil
}

func getNotificationsByParent(ctx context.Context, email string) ([]SimptomObavestenje, error) {
	cursor, err := obavestenjaCollection.Find(ctx, bson.M{"roditelj_email": strings.ToLower(strings.TrimSpace(email))}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := make([]SimptomObavestenje, 0)
	for cursor.Next(ctx) {
		var item SimptomObavestenje
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, cursor.Err()
}
