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

	http.HandleFunc("/test-veza", func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get("http://preschool-app:8081/vrtici")
		if err != nil {
			fmt.Fprintf(w, "Greska u komunikaciji: %v", err)
			return
		}
		defer resp.Body.Close()

		telo, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(w, "Uspesno povuceni podaci od Preschool servisa: %s", string(telo))
	})

	http.HandleFunc("/test-kriticni", func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get("http://preschool-app:8081/vrtici/kriticni")
		if err != nil {
			fmt.Fprintf(w, "Greska u komunikaciji: %v", err)
			return
		}
		defer resp.Body.Close()

		telo, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(w, "Kriticni vrtici: %s", string(telo))
	})

	http.HandleFunc("/test-izvestaj-opstina", func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get("http://preschool-app:8081/vrtici/izvestaj/opstina")
		if err != nil {
			fmt.Fprintf(w, "Greska u komunikaciji: %v", err)
			return
		}
		defer resp.Body.Close()

		telo, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(w, "Izvestaj po opstini: %s", string(telo))
	})

	fmt.Println("Analiticki servis na 8082...")
	http.ListenAndServe(":8082", nil)
}
