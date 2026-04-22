package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/milosavljevicstefan/euprava-projekat/open-data-service/internal/api"
	"github.com/milosavljevicstefan/euprava-projekat/open-data-service/internal/client"
	"github.com/milosavljevicstefan/euprava-projekat/open-data-service/internal/service"
)

func main() {
	// -------------------------------------------------------
	// Konfiguracija iz env varijabli (sa podrazumevanim vrednostima)
	// -------------------------------------------------------
	port := getEnv("PORT", "8084")
	vrticiAPIURL := getEnv("VRTICI_API_URL", "http://localhost:8081")

	log.Printf("[BOOT] Open Data servis se pokreće na portu %s", port)
	log.Printf("[BOOT] Vrtici API URL: %s", vrticiAPIURL)

	// -------------------------------------------------------
	// Inicijalizacija slojeva (Dependency Injection ručno)
	// -------------------------------------------------------

	// 1. HTTP klijent za komunikaciju sa eksternim servisom
	vrticiClient := client.NewVrticiClient(vrticiAPIURL)

	// 2. Servisni sloj sa poslovnom logikom
	openDataSvc := service.NewOpenDataService(vrticiClient)

	// 3. HTTP handler koji registruje rute
	handler := api.NewHandler(openDataSvc)

	// -------------------------------------------------------
	// Registracija ruta na DefaultServeMux
	// -------------------------------------------------------
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Primena logging middleware-a na sve rute
	loggedMux := api.LoggingMiddleware(mux)

	// -------------------------------------------------------
	// Pokretanje HTTP servera
	// -------------------------------------------------------
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      loggedMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Pokretanje servera u pozadini (goroutine)
	go func() {
		log.Printf("[BOOT] Server sluša na http://localhost:%s", port)
		log.Println("[BOOT] Dostupni endpointi:")
		log.Println("  GET /open-data/vrtici/csv")
		log.Println("  GET /open-data/zahtevi/csv")
		log.Println("  GET /open-data/konkursi/csv")
		log.Println("  GET /open-data/ocene/csv")
		log.Println("  GET /open-data/vrtici/json")
		log.Println("  GET /open-data/zahtevi/json")
		log.Println("  GET /open-data/download?dataset=<ime>&format=<csv|json>")
		log.Println("  GET /health")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Server greška: %v", err)
		}
	}()

	// -------------------------------------------------------
	// Graceful shutdown — čekamo SIGINT ili SIGTERM signal
	// -------------------------------------------------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[SHUTDOWN] Primljen signal za gašenje, zatvaranje servera...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("[SHUTDOWN] Greška pri gašenju: %v", err)
	}

	log.Println("[SHUTDOWN] Server uspešno ugašen.")
}

// getEnv čita env varijablu ili vraća podrazumevanu vrednost.
func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
