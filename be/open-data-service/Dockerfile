# ---- Build stage ----
FROM golang:1.25.5-alpine AS builder
WORKDIR /app

# Kopiramo go.mod (go.sum možda ne postoji pa koristimo wildcard)
COPY go.mod go.sum* ./

# Nema eksternih zavisnosti, samo verifikujemo
RUN go mod download || true

# Kopiramo izvorni kod eksplicitno po folderima
COPY cmd/      ./cmd/
COPY internal/ ./internal/

# go mod tidy generiše go.sum ako ne postoji, zatim buildujemo
RUN CGO_ENABLED=0 GOOS=linux go build -o open-data-service ./cmd

# ---- Run stage ----
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/open-data-service .

EXPOSE 8084

ENV PORT=8084
ENV VRTICI_API_URL=http://localhost:8081

CMD ["./open-data-service"]
