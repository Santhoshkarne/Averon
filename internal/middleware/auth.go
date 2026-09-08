package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"github.com/golang-jwt/jwt/v5"
)


func AuthMiddleware(authType, secret string, skipPaths []string, next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		
		
		if authType == "" {
		
			next.ServeHTTP(w, r)
			return
		}
		for _, skipPath := range skipPaths {
			if strings.HasPrefix(r.URL.Path, skipPath) {
				next.ServeHTTP(w, r)
				return
			}
		}
		if authType == "api_key" {
			key := r.Header.Get("X-API-Key")
			if key != secret {
				http.Error(w, "Unauthorized: Invalid API Key", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		if authType == "jwt" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized: Missing Authorization Header", http.StatusUnauthorized)
				return
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Unauthorized: Invalid Authorization Header format", http.StatusUnauthorized)
				return
			}
			tokenString := parts[1]
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Unauthorized: Invalid or expired JWT", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Internal Server Error: Unknown Auth Type", http.StatusInternalServerError)
	})
}
