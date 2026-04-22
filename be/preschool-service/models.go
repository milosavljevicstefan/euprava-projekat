package main

import (
	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"strings"
	"time"
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

type UpisZahtev struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	VrticID    primitive.ObjectID `json:"vrtic_id" bson:"vrtic_id"`
	KonkursID  primitive.ObjectID `json:"konkurs_id,omitempty" bson:"konkurs_id,omitempty"`
	VrticNaziv string             `json:"vrtic_naziv" bson:"vrtic_naziv"`

	ImeRoditelja         string     `json:"ime_roditelja" bson:"ime_roditelja"`
	ImeDeteta            string     `json:"ime_deteta" bson:"ime_deteta"`
	BrojGodina           int        `json:"broj_godina" bson:"broj_godina"`
	KorisnikEmail        string     `json:"korisnik_email" bson:"korisnik_email"`
	PotvrdaVakcinacije   bool       `json:"potvrda_vakcinacije" bson:"potvrda_vakcinacije"`
	IzvodIzMaticneKnjige bool       `json:"izvod_iz_maticne_knjige" bson:"izvod_iz_maticne_knjige"`
	Status               string     `json:"status" bson:"status"`
	CreatedAt            time.Time  `json:"created_at" bson:"created_at"`
	ProcessedAt          *time.Time `json:"processed_at,omitempty" bson:"processed_at,omitempty"`
	ProcessedBy          string     `json:"processed_by,omitempty" bson:"processed_by,omitempty"`
	Reason               string     `json:"reason,omitempty" bson:"reason,omitempty"`
}

type UpisRequest struct {
	VrticID              string `json:"vrtic_id"`
	ImeRoditelja         string `json:"ime_roditelja"`
	ImeDeteta            string `json:"ime_deteta"`
	BrojGodina           int    `json:"broj_godina"`
	PotvrdaVakcinacije   bool   `json:"potvrda_vakcinacije"`
	IzvodIzMaticneKnjige bool   `json:"izvod_iz_maticne_knjige"`
}

type RequestActionPayload struct {
	Reason string `json:"reason"`
}

type DokumentaUpdateRequest struct {
	PotvrdaVakcinacije   bool `json:"potvrda_vakcinacije"`
	IzvodIzMaticneKnjige bool `json:"izvod_iz_maticne_knjige"`
}

type Konkurs struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	VrticID        primitive.ObjectID `json:"vrtic_id" bson:"vrtic_id"`
	DatumPocetka   time.Time          `json:"datum_pocetka" bson:"datum_pocetka"`
	DatumZavrsetka time.Time          `json:"datum_zavrsetka" bson:"datum_zavrsetka"`
	MaxMesta       int                `json:"max_mesta" bson:"max_mesta"`
	Aktivan        bool               `json:"aktivan" bson:"aktivan"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	ClosedAt       *time.Time         `json:"closed_at,omitempty" bson:"closed_at,omitempty"`
}

type KonkursRequest struct {
	VrticID        string `json:"vrtic_id"`
	DatumPocetka   string `json:"datum_pocetka"`
	DatumZavrsetka string `json:"datum_zavrsetka"`
	MaxMesta       int    `json:"max_mesta"`
}

type KonkursView struct {
	ID             primitive.ObjectID `json:"id"`
	VrticID        primitive.ObjectID `json:"vrtic_id"`
	VrticNaziv     string             `json:"vrtic_naziv"`
	DatumPocetka   time.Time          `json:"datum_pocetka"`
	DatumZavrsetka time.Time          `json:"datum_zavrsetka"`
	MaxMesta       int                `json:"max_mesta"`
	Aktivan        bool               `json:"aktivan"`
	Status         string             `json:"status"`
	Popunjeno      int                `json:"popunjeno"`
	SlobodnaMesta  int                `json:"slobodna_mesta"`
}

type VaspitacRaspored struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	VrticID       primitive.ObjectID `json:"vrtic_id" bson:"vrtic_id"`
	VrticNaziv    string             `json:"vrtic_naziv" bson:"vrtic_naziv"`
	VaspitacEmail string             `json:"vaspitac_email" bson:"vaspitac_email"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
}

type VaspitacRasporedRequest struct {
	VrticID       string `json:"vrtic_id"`
	VaspitacEmail string `json:"vaspitac_email"`
}

type RoditeljVaspitaciView struct {
	ZahtevID   primitive.ObjectID `json:"zahtev_id"`
	VrticID    primitive.ObjectID `json:"vrtic_id"`
	VrticNaziv string             `json:"vrtic_naziv"`
	ImeDeteta  string             `json:"ime_deteta"`
	Vaspitaci  []string           `json:"vaspitaci"`
}

type Sastanak struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ZahtevID      primitive.ObjectID `json:"zahtev_id" bson:"zahtev_id"`
	VrticID       primitive.ObjectID `json:"vrtic_id" bson:"vrtic_id"`
	VrticNaziv    string             `json:"vrtic_naziv" bson:"vrtic_naziv"`
	ImeDeteta     string             `json:"ime_deteta" bson:"ime_deteta"`
	RoditeljEmail string             `json:"roditelj_email" bson:"roditelj_email"`
	VaspitacEmail string             `json:"vaspitac_email" bson:"vaspitac_email"`
	Termin        time.Time          `json:"termin" bson:"termin"`
	Napomena      string             `json:"napomena,omitempty" bson:"napomena,omitempty"`
	Status        string             `json:"status" bson:"status"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	ProcessedAt   *time.Time         `json:"processed_at,omitempty" bson:"processed_at,omitempty"`
	ProcessedBy   string             `json:"processed_by,omitempty" bson:"processed_by,omitempty"`
	Reason        string             `json:"reason,omitempty" bson:"reason,omitempty"`
}

type SastanakRequest struct {
	ZahtevID      string `json:"zahtev_id"`
	VaspitacEmail string `json:"vaspitac_email"`
	Termin        string `json:"termin"`
	Napomena      string `json:"napomena"`
}

type SastanakActionPayload struct {
	Reason string `json:"reason"`
}

type SimptomObavestenje struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ZahtevID      primitive.ObjectID `json:"zahtev_id" bson:"zahtev_id"`
	VrticID       primitive.ObjectID `json:"vrtic_id" bson:"vrtic_id"`
	VrticNaziv    string             `json:"vrtic_naziv" bson:"vrtic_naziv"`
	ImeDeteta     string             `json:"ime_deteta" bson:"ime_deteta"`
	RoditeljEmail string             `json:"roditelj_email" bson:"roditelj_email"`
	VaspitacEmail string             `json:"vaspitac_email" bson:"vaspitac_email"`
	Poruka        string             `json:"poruka" bson:"poruka"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
}

type SimptomObavestenjeRequest struct {
	ZahtevID string `json:"zahtev_id"`
	Poruka   string `json:"poruka"`
}

const (
	statusSubmitted   = "podnet"
	statusInReview    = "u_obradi"
	statusNeedDocs    = "dopuna_dokumentacije"
	statusApproved    = "odobren"
	statusRejected    = "odbijen"
	statusWaitingList = "na_listi_cekanja"

	meetingStatusPending  = "na_cekanju"
	meetingStatusAccepted = "prihvacen"
	meetingStatusRejected = "odbijen"
)

var activeRequestStatuses = []string{statusSubmitted, statusInReview, statusNeedDocs, statusWaitingList, statusApproved}

var vrticiCollection *mongo.Collection
var zahteviCollection *mongo.Collection
var konkursiCollection *mongo.Collection
var rasporediCollection *mongo.Collection
var sastanciCollection *mongo.Collection
var obavestenjaCollection *mongo.Collection

func canonicalRequestStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", statusSubmitted, "na_cekanju":
		return statusSubmitted
	case statusInReview, "u_proveri":
		return statusInReview
	case statusNeedDocs:
		return statusNeedDocs
	case statusApproved:
		return statusApproved
	case statusRejected:
		return statusRejected
	case statusWaitingList:
		return statusWaitingList
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func canonicalMeetingStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", meetingStatusPending:
		return meetingStatusPending
	case "zakazan", meetingStatusAccepted:
		return meetingStatusAccepted
	case meetingStatusRejected:
		return meetingStatusRejected
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func isAdminClaim(claims jwt.MapClaims) bool {
	return strings.ToLower(strings.TrimSpace(claimString(claims, "role"))) == "admin"
}

func canAccessRequestDocument(item UpisZahtev, claims jwt.MapClaims) bool {
	if isAdminClaim(claims) {
		return true
	}
	email := strings.ToLower(strings.TrimSpace(claimString(claims, "sub")))
	return email != "" && email == strings.ToLower(strings.TrimSpace(item.KorisnikEmail))
}
