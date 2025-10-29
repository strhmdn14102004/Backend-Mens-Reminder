package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const UserIDKey ctxKey = "uid"

func IssueJWT(secret string, uid int64) (string, error) {
	claims := jwt.MapClaims{
		"sub": uid,
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

func Auth(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := r.Header.Get("Authorization")
		if !strings.HasPrefix(a, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(a, "Bearer ")
		t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !t.Valid {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mc := t.Claims.(jwt.MapClaims)
		uid := int64(mc["sub"].(float64))
		ctx := context.WithValue(r.Context(), UserIDKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
