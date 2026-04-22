package main

import (
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"os"
	"strings"
)

func requireAuth(r *http.Request) (jwt.MapClaims, error) {
	secret := getenvDefault("JWT_SECRET", "dev-secret")
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("Nedostaje Authorization header")
	}

	tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
	if tokenString == "" {
		return nil, errors.New("Neispravan token")
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Neispravan algoritam")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("Neispravan ili istekao token")
	}

	return claims, nil
}

func claimString(claims jwt.MapClaims, key string) string {
	value, ok := claims[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func getenvDefault(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}
