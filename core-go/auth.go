package main

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const UserCtxKey contextKey = "userID"

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Неверный формат токена", http.StatusUnauthorized)
			return
		}

		userID := 100

		ctx := context.WithValue(r.Context(), UserCtxKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func getUserIDFromContext(r *http.Request) (int, bool) {
	userID, ok := r.Context().Value(UserCtxKey).(int)
	return userID, ok

}

// Подготовка к permissions (из JWT)
// Пока я не получил настоящего JWT - пусть будет заглушка
func hasPermission(r *http.Request, required string) bool {
	// Сейчас: разрешаем всё (для разработки)
	return true
}
