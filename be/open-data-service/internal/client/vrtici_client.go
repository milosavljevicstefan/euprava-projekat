// Package client sadrži HTTP klijent za komunikaciju sa eksternim servisom.
package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/milosavljevicstefan/euprava-projekat/open-data-service/internal/model"
)

// VrticiClient je HTTP klijent koji poziva eksterni Vrtici servis.
type VrticiClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewVrticiClient kreira novi instancu klijenta sa zadatim base URL-om.
// Primer: NewVrticiClient("http://vrtici-service:8080")
func NewVrticiClient(baseURL string) *VrticiClient {
	return &VrticiClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Timeout da ne blokiramo server zauvek
		},
	}
}

// FetchAllData poziva GET /api/export/all-data na eksternom servisu
// i vraća deserijalizovane podatke ili grešku ako servis nije dostupan.
func (c *VrticiClient) FetchAllData() (*model.ExportData, error) {
	url := fmt.Sprintf("%s/analytics/all-data", c.baseURL)
	log.Printf("[CLIENT] Preuzimanje podataka sa: %s", url)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("greška pri pozivu eksternog API-ja (%s): %w", url, err)
	}
	defer resp.Body.Close()

	// Proveravamo da li je HTTP status OK (200)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eksterni API vratio status %d umesto 200", resp.StatusCode)
	}

	var data model.ExportData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("greška pri parsiranju odgovora: %w", err)
	}

	log.Printf("[CLIENT] Uspešno preuzeti podaci — vrtici:%d zahtevi:%d konkursi:%d ocene:%d",
		len(data.Vrtici), len(data.Zahtevi), len(data.Konkursi), len(data.Ocene))

	return &data, nil
}
