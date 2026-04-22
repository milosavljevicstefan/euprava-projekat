package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"
)

func buildOpstinaPDFReport(report []OpstinaIzvestaj) ([]byte, error) {
	lines := []string{
		"Izvestaj o kapacitetima po opstini",
		fmt.Sprintf("Datum: %s", time.Now().Format("2006-01-02 15:04")),
		"",
	}

	if len(report) == 0 {
		lines = append(lines, "Nema podataka za izvestaj.")
	} else {
		for _, row := range report {
			lines = append(lines, fmt.Sprintf(
				"%s | vrtici:%d | kapacitet:%d | upisano:%d | popunjenost:%.2f%%",
				row.Opstina,
				row.BrojVrtica,
				row.UkupanKapacitet,
				row.UkupnoUpisano,
				row.Popunjenost*100,
			))
		}
	}

	return buildSimplePDF(lines), nil
}

func buildRequestDecisionPDF(item UpisZahtev) ([]byte, string, error) {
	status := canonicalRequestStatus(item.Status)
	if status != statusApproved && status != statusRejected {
		return nil, "", errors.New("PDF je dostupan samo za odobren ili odbijen zahtev")
	}

	title := "Potvrda o upisu"
	fileName := fmt.Sprintf("potvrda-upis-%s.pdf", item.ID.Hex())
	if status == statusRejected {
		title = "Odbijenica"
		fileName = fmt.Sprintf("odbijenica-%s.pdf", item.ID.Hex())
	}

	lines := []string{
		title,
		"E-Uprava - Vrtici",
		fmt.Sprintf("Vrtic: %s", item.VrticNaziv),
		fmt.Sprintf("Roditelj: %s", item.ImeRoditelja),
		fmt.Sprintf("Dete: %s", item.ImeDeteta),
		fmt.Sprintf("Broj godina: %d", item.BrojGodina),
		fmt.Sprintf("Potvrda o vakcinaciji: %s", map[bool]string{true: "prilozena", false: "nije prilozena"}[item.PotvrdaVakcinacije]),
		fmt.Sprintf("Izvod iz maticne knjige rodjenih: %s", map[bool]string{true: "prilozen", false: "nije prilozen"}[item.IzvodIzMaticneKnjige]),
		fmt.Sprintf("Datum podnosenja: %s", item.CreatedAt.Format("02.01.2006 15:04")),
		fmt.Sprintf("Status: %s", status),
	}
	if item.ProcessedBy != "" {
		lines = append(lines, fmt.Sprintf("Obradio: %s", item.ProcessedBy))
	}
	if item.ProcessedAt != nil {
		lines = append(lines, fmt.Sprintf("Datum obrade: %s", item.ProcessedAt.Format("02.01.2006 15:04")))
	}
	if strings.TrimSpace(item.Reason) != "" {
		lines = append(lines, fmt.Sprintf("Napomena: %s", item.Reason))
	}

	return buildSimplePDF(lines), fileName, nil
}

func buildSimplePDF(lines []string) []byte {
	var stream bytes.Buffer
	stream.WriteString("BT\n/F1 12 Tf\n50 760 Td\n")
	for i, line := range lines {
		if i > 0 {
			stream.WriteString("0 -16 Td\n")
		}
		stream.WriteString("(")
		stream.WriteString(escapePDFText(line))
		stream.WriteString(") Tj\n")
	}
	stream.WriteString("ET")

	content := stream.String()

	var pdf bytes.Buffer
	offsets := []int{0}
	writeObj := func(objNum int, objContent string) {
		offsets = append(offsets, pdf.Len())
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", objNum, objContent)
	}

	pdf.WriteString("%PDF-1.4\n")
	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>")
	writeObj(4, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	writeObj(5, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content))

	xrefPos := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n", len(offsets))
	pdf.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\n", len(offsets))
	fmt.Fprintf(&pdf, "startxref\n%d\n%%%%EOF", xrefPos)

	return pdf.Bytes()
}

func escapePDFText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

func toViews(vrtici []Vrtic) []VrticView {
	views := make([]VrticView, 0, len(vrtici))
	for _, v := range vrtici {
		views = append(views, VrticView{
			ID:              v.ID,
			Naziv:           v.Naziv,
			Tip:             v.Tip,
			Grad:            v.Grad,
			Opstina:         v.Opstina,
			MaxKapacitet:    v.MaxKapacitet,
			TrenutnoUpisano: v.TrenutnoUpisano,
			Popunjenost:     popunjenost(v),
			SlobodnaMesta:   slobodnaMesta(v),
			Kriticno:        popunjenost(v) >= 0.9,
		})
	}
	return views
}

func popunjenost(v Vrtic) float64 {
	if v.MaxKapacitet <= 0 {
		return 0
	}
	return float64(v.TrenutnoUpisano) / float64(v.MaxKapacitet)
}

func slobodnaMesta(v Vrtic) int {
	if v.MaxKapacitet <= 0 {
		return 0
	}
	return v.MaxKapacitet - v.TrenutnoUpisano
}
