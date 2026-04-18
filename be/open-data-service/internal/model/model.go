// Package model sadrži definicije struktura podataka koje se koriste u sistemu.
package model

import "time"

// Vrtic predstavlja podatke o jednom vrtiću u sistemu.
type Vrtic struct {
	ID        int    `json:"id"`
	Naziv     string `json:"naziv"`
	Adresa    string `json:"adresa"`
	Opstina   string `json:"opstina"`
	Telefon   string `json:"telefon"`
	Email     string `json:"email"`
	Kapacitet int    `json:"kapacitet"`
	BrojDece  int    `json:"broj_dece"`
	Direktor  string `json:"direktor"`
	Aktivan   bool   `json:"aktivan"`
}

// CSVHeader vraća zaglavlje CSV fajla za Vrtic.
func (v Vrtic) CSVHeader() []string {
	return []string{"id", "naziv", "adresa", "opstina", "telefon", "email", "kapacitet", "broj_dece", "direktor", "aktivan"}
}

// CSVRow vraća red podataka za CSV fajl.
func (v Vrtic) CSVRow() []string {
	aktivan := "ne"
	if v.Aktivan {
		aktivan = "da"
	}
	return []string{
		itoa(v.ID),
		v.Naziv,
		v.Adresa,
		v.Opstina,
		v.Telefon,
		v.Email,
		itoa(v.Kapacitet),
		itoa(v.BrojDece),
		v.Direktor,
		aktivan,
	}
}

// ZahtevZaUpis predstavlja zahtev roditelja za upisivanje deteta u vrtić.
type ZahtevZaUpis struct {
	ID          int       `json:"id"`
	VrticID     int       `json:"vrtic_id"`
	ImeDeteta   string    `json:"ime_deteta"`
	GodinaRodj  int       `json:"godina_rodjenja"`
	ImeRoditelja string   `json:"ime_roditelja"`
	KontaktTel  string    `json:"kontakt_telefon"`
	DatumZahteva time.Time `json:"datum_zahteva"`
	Status      string    `json:"status"` // ceka, odobren, odbijen
	Napomena    string    `json:"napomena"`
}

// CSVHeader vraća zaglavlje CSV fajla za ZahtevZaUpis.
func (z ZahtevZaUpis) CSVHeader() []string {
	return []string{"id", "vrtic_id", "ime_deteta", "godina_rodjenja", "ime_roditelja", "kontakt_telefon", "datum_zahteva", "status", "napomena"}
}

// CSVRow vraća red podataka za CSV fajl.
func (z ZahtevZaUpis) CSVRow() []string {
	return []string{
		itoa(z.ID),
		itoa(z.VrticID),
		z.ImeDeteta,
		itoa(z.GodinaRodj),
		z.ImeRoditelja,
		z.KontaktTel,
		z.DatumZahteva.Format("2006-01-02"),
		z.Status,
		z.Napomena,
	}
}

// Konkurs predstavlja oglas/konkurs za upis dece u vrtiće.
type Konkurs struct {
	ID          int       `json:"id"`
	VrticID     int       `json:"vrtic_id"`
	NazivVrtica string    `json:"naziv_vrtića"`
	BrojMesta   int       `json:"broj_mesta"`
	DatumOd     time.Time `json:"datum_od"`
	DatumDo     time.Time `json:"datum_do"`
	Opis        string    `json:"opis"`
	Aktivan     bool      `json:"aktivan"`
	GodinaUpisa int       `json:"godina_upisa"`
}

// CSVHeader vraća zaglavlje CSV fajla za Konkurs.
func (k Konkurs) CSVHeader() []string {
	return []string{"id", "vrtic_id", "naziv_vrtića", "broj_mesta", "datum_od", "datum_do", "opis", "aktivan", "godina_upisa"}
}

// CSVRow vraća red podataka za CSV fajl.
func (k Konkurs) CSVRow() []string {
	aktivan := "ne"
	if k.Aktivan {
		aktivan = "da"
	}
	return []string{
		itoa(k.ID),
		itoa(k.VrticID),
		k.NazivVrtica,
		itoa(k.BrojMesta),
		k.DatumOd.Format("2006-01-02"),
		k.DatumDo.Format("2006-01-02"),
		k.Opis,
		aktivan,
		itoa(k.GodinaUpisa),
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

// ExportData je struktura koja dolazi iz eksternog Vrtici API-ja.
type ExportData struct {
	Vrtici   []Vrtic        `json:"vrtici"`
	Zahtevi  []ZahtevZaUpis `json:"zahtevi"`
	Konkursi []Konkurs      `json:"konkursi"`
	Ocene    []Ocena        `json:"ocene"`
}
