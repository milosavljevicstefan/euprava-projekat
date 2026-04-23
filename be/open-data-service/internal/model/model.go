// Package model sadrži definicije struktura podataka koje se koriste u sistemu.
package model

import (
	"time"
	"fmt"
)

// Vrtic predstavlja podatke o jednom vrtiću u sistemu.
type Vrtic struct {
	ID            string  `json:"id"`
	Naziv         string  `json:"naziv"`
	Tip           string  `json:"tip"`
	Grad          string  `json:"grad"`
	Opstina       string  `json:"opstina"`
	Kapacitet     int     `json:"max_kapacitet"`
	BrojDece      int     `json:"trenutno_upisano"`
	Popunjenost   float64 `json:"popunjenost"`
	SlobodnaMesta int     `json:"slobodna_mesta"`
	Kriticno      bool    `json:"kriticno"`
}
// CSVHeader vraća zaglavlje CSV fajla za Vrtic.
func (v Vrtic) CSVHeader() []string {
	return []string{"naziv", "tip", "grad", "kapacitet", "opstina", "broj_dece", "popunjenost", "kriticni?"}
}
func boolToString(b bool) string {
	if b {
		return "DA"
	}
	return "NE"
}
func ftoaa(f float64) string {
	return fmt.Sprintf("%.0f%%", f*100)
}
// CSVRow vraća red podataka za CSV fajl.
func (v Vrtic) CSVRow() []string {
	return []string{
		v.Naziv,
		v.Tip,
		v.Grad,
		itoa(v.Kapacitet),
		v.Opstina,
		itoa(v.BrojDece),
		ftoaa(v.Popunjenost),
		boolToString(v.Kriticno),
	}
}

// ZahtevZaUpis predstavlja zahtev roditelja za upisivanje deteta u vrtić.
type ZahtevZaUpis struct {
	ID            string    `json:"id"`
	VrticID       string    `json:"vrtic_id"`
	ImeDeteta     string    `json:"ime_deteta"`
	GodinaRodj    int       `json:"broj_godina"`
	ImeRoditelja  string    `json:"ime_roditelja"`
	DatumZahteva  time.Time `json:"created_at"`
	Status        string    `json:"status"`
	Napomena      string    `json:"napomena"`
}

// CSVHeader vraća zaglavlje CSV fajla za ZahtevZaUpis.
func (z ZahtevZaUpis) CSVHeader() []string {
	return []string{"ime_deteta", "godina_rodjenja", "ime_roditelja", "datum_zahteva", "status"}
}

// CSVRow vraća red podataka za CSV fajl.
func (z ZahtevZaUpis) CSVRow() []string {
	return []string{
		z.ImeDeteta,
		itoa(z.GodinaRodj),
		z.ImeRoditelja,
		z.DatumZahteva.Format("2006-01-02"),
		z.Status,
	}
}

// Konkurs predstavlja oglas/konkurs za upis dece u vrtiće.
type Konkurs struct {
    ID              string    `json:"id"`
    VrticID         string    `json:"vrtic_id"`
    NazivVrtica     string    `json:"vrtic_naziv"`
    DatumOd         time.Time `json:"datum_pocetka"`
    DatumDo         time.Time `json:"datum_zavrsetka"`
    BrojMesta       int       `json:"max_mesta"`
    Aktivan         bool      `json:"aktivan"`
}

// CSVHeader vraća zaglavlje CSV fajla za Konkurs.
func (k Konkurs) CSVHeader() []string {
	return []string{ "naziv_vrtića", "broj_mesta", "datum_od", "datum_do", "aktivan"}
}

// CSVRow vraća red podataka za CSV fajl.
func (k Konkurs) CSVRow() []string {
	aktivan := "ne"
	if k.Aktivan {
		aktivan = "da"
	}
	return []string{
		k.NazivVrtica,
		itoa(k.BrojMesta),
		k.DatumOd.Format("2006-01-02"),
		k.DatumDo.Format("2006-01-02"),
		aktivan,
	}
}

// Ocena predstavlja ocenu/recenziju roditelja za određeni vrtić.
type Ocena struct {
	ID          int       `json:"id"`
	VrticID     int       `json:"vrtic_id"`
	NazivVrtica string    `json:"naziv_vrtića"`
	Ocena       float64   `json:"ocena"` // 1.0 - 5.0
	Komentar    string    `json:"komentar"`
	DatumOcene  time.Time `json:"datum_ocene"`
	AnonimanRoditelj bool `json:"anoniman_roditelj"`
}

// CSVHeader vraća zaglavlje CSV fajla za Ocena.
func (o Ocena) CSVHeader() []string {
	return []string{"id", "vrtic_id", "naziv_vrtića", "ocena", "komentar", "datum_ocene", "anoniman_roditelj"}
}

// CSVRow vraća red podataka za CSV fajl.
func (o Ocena) CSVRow() []string {
	anoniman := "ne"
	if o.AnonimanRoditelj {
		anoniman = "da"
	}
	return []string{
		itoa(o.ID),
		itoa(o.VrticID),
		o.NazivVrtica,
		ftoa(o.Ocena),
		o.Komentar,
		o.DatumOcene.Format("2006-01-02"),
		anoniman,
	}
}

type ExportData struct {
	Vrtici        []Vrtic  		 `json:"vrtici"`
	Zahtevi       []ZahtevZaUpis `json:"zahtevi_upisa"`
	Konkursi      []Konkurs      `json:"konkursi"`
	Ocene         []Ocena        `json:"ocene_vrtica"`
}
