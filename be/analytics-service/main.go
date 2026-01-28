package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Analiticki servis (8082) je online.")
	})

	// TEST KOMUNIKACIJE
	http.HandleFunc("/test-veza", func(w http.ResponseWriter, r *http.Request) {
		// Koristimo IME KONTEJNERA iz docker-compose-a (preschool-app)
		resp, err := http.Get("http://preschool-app:8081/vrtici")
		if err != nil {
			fmt.Fprintf(w, "Greska u komunikaciji: %v", err)
			return
		}
		defer resp.Body.Close()

		telo, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(w, "Uspesno povuceni podaci od Preschool servisa: %s", string(telo))
	})

	fmt.Println("Analiticki servis na 8082...")
	http.ListenAndServe(":8082", nil)
}