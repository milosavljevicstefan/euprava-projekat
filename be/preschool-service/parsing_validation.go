package main

import (
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strings"
	"time"
)

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
	action := strings.ToLower(strings.TrimSpace(parts[1]))
	if action == "provera" {
		action = "obrada"
	}
	switch action {
	case "obrada", "dopuna", "odobri", "odbij", "dokument", "dokumenta", "izmeni":
		return id, action, nil
	default:
		return primitive.NilObjectID, "", errors.New("Nepoznata akcija")
	}
}

func parseSimpleObjectID(path, prefix string) (primitive.ObjectID, error) {
	idPart := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if idPart == "" || strings.Contains(idPart, "/") {
		return primitive.NilObjectID, errors.New("Neispravan ID")
	}
	id, err := primitive.ObjectIDFromHex(idPart)
	if err != nil {
		return primitive.NilObjectID, errors.New("Neispravan ID")
	}
	return id, nil
}

func parseKonkursAction(path string) (primitive.ObjectID, string, error) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, "/konkursi/"), "/"), "/")
	if len(parts) != 2 {
		return primitive.NilObjectID, "", errors.New("Neispravan URL konkursa")
	}
	id, err := primitive.ObjectIDFromHex(parts[0])
	if err != nil {
		return primitive.NilObjectID, "", errors.New("Neispravan ID konkursa")
	}
	return id, parts[1], nil
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
	if !req.PotvrdaVakcinacije {
		return errors.New("Potvrda o vakcinaciji je obavezna pri upisu")
	}
	if !req.IzvodIzMaticneKnjige {
		return errors.New("Izvod iz maticne knjige rodjenih je obavezan pri upisu")
	}
	return nil
}

func validateAssignmentInput(req VaspitacRasporedRequest) error {
	if strings.TrimSpace(req.VrticID) == "" {
		return errors.New("Vrtic je obavezan")
	}
	if strings.TrimSpace(req.VaspitacEmail) == "" {
		return errors.New("Email vaspitaca je obavezan")
	}
	return nil
}

func validateMeetingInput(req SastanakRequest) error {
	if strings.TrimSpace(req.ZahtevID) == "" {
		return errors.New("Izaberi dete za sastanak")
	}
	if strings.TrimSpace(req.VaspitacEmail) == "" {
		return errors.New("Izaberi vaspitaca")
	}
	if strings.TrimSpace(req.Termin) == "" {
		return errors.New("Termin sastanka je obavezan")
	}
	return nil
}

func validateSymptomsInput(req SimptomObavestenjeRequest) error {
	if strings.TrimSpace(req.ZahtevID) == "" {
		return errors.New("Dete je obavezno")
	}
	if strings.TrimSpace(req.Poruka) == "" {
		return errors.New("Poruka je obavezna")
	}
	return nil
}

func validateKonkursInput(req KonkursRequest) error {
	if strings.TrimSpace(req.VrticID) == "" {
		return errors.New("Vrtic je obavezan")
	}
	if strings.TrimSpace(req.DatumPocetka) == "" || strings.TrimSpace(req.DatumZavrsetka) == "" {
		return errors.New("Pocetak i kraj konkursa su obavezni")
	}
	if req.MaxMesta <= 0 {
		return errors.New("Max mesta mora biti vece od nule")
	}
	return nil
}

func parseDateValue(raw string, endOfDay bool) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("Datum je obavezan")
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, errors.New("Neispravan format datuma")
	}
	if endOfDay {
		return t.Add(23*time.Hour + 59*time.Minute + 59*time.Second), nil
	}
	return t, nil
}

func konkursStatusLabel(item Konkurs, now time.Time) string {
	if !item.Aktivan {
		return "zatvoren"
	}
	if now.Before(item.DatumPocetka) {
		return "zakazan"
	}
	if now.After(item.DatumZavrsetka) {
		return "istekao"
	}
	return "aktivan"
}

func requireAdminRole(claims jwt.MapClaims) error {
	if strings.ToLower(strings.TrimSpace(claimString(claims, "role"))) == "admin" {
		return nil
	}
	return errors.New("Nemate dozvolu za admin operacije")
}

func requireUserRole(claims jwt.MapClaims) error {
	role := strings.ToLower(strings.TrimSpace(claimString(claims, "role")))
	if role == "roditelj" || role == "korisnik" {
		return nil
	}
	return errors.New("Samo roditelj moze slati zahtev za upis")
}

func requireEducatorRole(claims jwt.MapClaims) error {
	if strings.ToLower(strings.TrimSpace(claimString(claims, "role"))) == "vaspitac" {
		return nil
	}
	return errors.New("Samo vaspitac moze koristiti ovu funkciju")
}
