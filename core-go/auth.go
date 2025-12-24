// auth.go
package main

import (
	"context"
	"net/http"
	"strings"
)

// Ключи контекста
type contextKey string

const (
	UserCtxKey        contextKey = "userID"
	PermissionsCtxKey contextKey = "permissions"
)

// 🔑 Временный секрет (заменить на os.Getenv("JWT_SECRET"))
var jwtSecret = []byte("survey-dev-secret")

// parseJWT — извлекает user_id и permissions из токена
// 🔜 ЗАМЕНИТЬ НА НАСТОЯЩИЙ JWT, КОГДА БУДЕТ ГОТОВ АВТОРИЗАЦИОННЫЙ МОДУЛЬ
func parseJWT(tokenStr string) (userID int, permissions []string, err error) {
	// ───────────────────────────────────────────────
	// ✅ ЗАГЛУШКА: работает ТОЛЬКО для разработки
	// В продакшене — УДАЛИТЬ этот блок
	// ───────────────────────────────────────────────
	return 100, []string{
		"user:list:read",
		"course:add",
		"course:test:add",
		"course:test:write",
		"course:test:read",
		"test:list:read",
		"test:answer:read",
		"quest:create",
	}, nil
	// ───────────────────────────────────────────────

	// 🛑 РАСКОММЕНТИРУЙ ЭТОТ БЛОК ПРИ ПОДКЛЮЧЕНИИ НАСТОЯЩЕГО JWT
	/*
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("неподдерживаемый алгоритм")
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			return 0, nil, fmt.Errorf("недействительный токен")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return 0, nil, fmt.Errorf("неверный формат claims")
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			return 0, nil, fmt.Errorf("user_id отсутствует")
		}
		userID = int(userIDFloat)

		perms := []string{}
		if rawPerms, ok := claims["permissions"].([]interface{}); ok {
			for _, p := range rawPerms {
				if s, ok := p.(string); ok {
					perms = append(perms, s)
				}
			}
		}
		return userID, perms, nil
	*/
}

// hasPermission — проверяет, есть ли у пользователя право
func hasPermission(r *http.Request, required string) bool {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	_, permissions, err := parseJWT(tokenStr)
	if err != nil {
		return false
	}

	for _, p := range permissions {
		if p == required {
			return true
		}
	}
	return false
}

// AuthMiddleware — проверяет токен и кладёт данные в контекст
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		userID, permissions, err := parseJWT(tokenStr)
		if err != nil {
			http.Error(w, "Невалидный токен", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserCtxKey, userID)
		ctx = context.WithValue(ctx, PermissionsCtxKey, permissions)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// getUserIDFromContext — безопасное извлечение user_id
func getUserIDFromContext(r *http.Request) (int, bool) {
	userID, ok := r.Context().Value(UserCtxKey).(int)
	return userID, ok
}
