# EUprava Projekat

Ovo je timski projekat iz oblasti softverskog inženjeringa koji simulira upravljanje vrtićima koristeći mikroservisnu arhitekturu. Projekat sadrži **backend servise** napisane u Go-u i **frontend** sa statičkim HTML/JS fajlovima.

---

## Struktura projekta
```
root/
│
├── be/ # Backend servisi (Go)
│ ├── preschool-service/ # Servis za vrtiće
│ ├── auth-service/ # Servis za autentifikaciju (SSO)
│ ├── open-data-service/ # Servis za open data analitiku i ranking
│ └── docker-compose.yml # Docker Compose konfiguracija svih servisa
│
└── fe/ # Frontend
├── index.html
├── app.js
└── styles.css
```

---

## Backend servisi

### 1. Preschool Service (`be/preschool-service`)

- CRUD operacije za vrtiće (`/vrtici`)
- Dohvatanje kritičnih vrtića (`/vrtici/kriticni`)
- Izveštaj po opštinama (`/vrtici/izvestaj/opstina`) sa opcijom PDF generacije
- Integracija sa MongoDB za skladištenje podataka

### 2. Auth Service (`be/auth-service`)

- JWT autentifikacija i autorizacija
- Kreiranje korisnika i login
- SSO podrška za timsku autentifikaciju

### 3. Open Data Service (`be/open-data-service`)

- Analitika vrtića po opštinama
- Endpoint-i:
  - `/analytics/coverage?opstina={naziv}` – pokrivenost kapaciteta vrtića prema broju dece
  - `/analytics/ranking` – rangiranje opština po popunjenosti vrtića (najpopunjenije prve)
  - `/analytics/projection` – rangiranje opština po popunjenosti vrtića (najmanje popunjene prve)
- CSV ili JSON izlaz za dalju analitiku

---

## Frontend (`fe/`)

- Staticki HTML + JavaScript
- Pokreće se pomoću **Live Server ekstenzije** u VS Code-u
- Komunicira sa backend servisima putem **REST API-ja**

---

## Docker setup

Projekat koristi Docker Compose za lokalno pokretanje svih servisa:

```bash
cd be
docker compose up --build
```
Servisi će biti dostupni na sledećim portovima:

Servis	Port
Preschool Service	8081
Open Data Service	8082
Auth Service	8083
MongoDB	27018

Pokrenuti frontend:

Otvori fe/index.html preko Live Server ekstenzije

Frontend će koristiti API pozive prema backend servisima

Testiranje API-ja:

# Primeri curl komandi
curl http://localhost:8081/vrtici
curl "http://localhost:8082/analytics/coverage?opstina=Zvezdara"
curl http://localhost:8082/analytics/ranking
Tehnologije

Backend: Go (Golang), MongoDB, JWT, Docker

Frontend: HTML, JavaScript

Alati: Docker, Live Server (VS Code), curl za testiranje API-ja

Autori

Stefan G SR/21 2021 – Preschool Service
Stefan M SR/12 2021 – Open Data Service + Frontend

Timski rad: Auth Service i SSO integracija

Napomene

Projekat je izrađen u okviru tima, svaki član ima svoj servis.

Docker Compose omogućava lokalno pokretanje svih servisa zajedno sa bazom.

Frontend koristi statički pristup, dok backend pruža REST API-je za CRUD i analitiku.
