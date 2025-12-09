package main

import (
	"context"
	"net/http"
	"strings"
)

// Ключи для хранения данных в контексте запроса
type contextKey string

const (
	UserCtxKey        contextKey = "userID"
	PermissionsCtxKey contextKey = "permissions"
)

// 🔑 Временный секрет для JWT (заменить, когда коллега даст настоящий)
// var jwtsecret = []byte("survey-dev-secret")

// parseJWT — извлекает user_id и permissions из токена
// ПОКА ИСПОЛЬЗУЕТ ЗАГЛУШКУ → ЛЕГКО ЗАМЕНИТЬ НА НАСТОЯЩИЙ JWT
func parseJWT(tokenStr string) (userID int, permissions []string, err error) {
	// 🔜 ОТКЛЮЧИ ЭТУ ЧАСТЬ, КОГДА ПОДКЛЮЧИШЬ НАСТОЯЩИЙ JWT
	// -----------------------------------------------
	// ЗАГЛУШКА: любой токен — валиден, user_id = 100
	// Права — временно все разрешены
	return 100, []string{
		"test:list:read",
		"test:create:write",
		"test:answer:read",
		"course:test:add",
		"course:test:write",
	}, nil
	// -----------------------------------------------

	// РАСКОММЕНТИРУЮ ЭТУ ЧАСТЬ, КОГДА БУДЕТ НАСТОЯЩИЙ JWT
	/*
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			// Проверяем алгоритм (для HS256)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("неподдерживаемый алгоритм подписи")
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			return 0, nil, err
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return 0, nil, fmt.Errorf("claims не являются MapClaims")
		}

		// Извлекаем user_id
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			return 0, nil, fmt.Errorf("user_id отсутствует или неверного типа")
		}
		userID = int(userIDFloat)

		// Извлекаем permissions
		var perms []string
		if permsRaw, ok := claims["permissions"].([]interface{}); ok {
			for _, p := range permsRaw {
				if permStr, ok := p.(string); ok {
					perms = append(perms, permStr)
				}
			}
		}

		return userID, perms, nil
	*/
}

// hasPermission — проверяет, есть ли у пользователя требуемое право
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

// AuthMiddleware — проверяет токен и кладёт user_id + permissions в контекст
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

		// Передаём данные в обработчики через контекст
		ctx := context.WithValue(r.Context(), UserCtxKey, userID)
		ctx = context.WithValue(ctx, PermissionsCtxKey, permissions)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// getUserIDFromContext — извлекает user_id из контекста
func getUserIDFromContext(r *http.Request) (int, bool) {
	userID, ok := r.Context().Value(UserCtxKey).(int)
	return userID, ok
}

// getPermissionsFromContext — извлекает permissions из контекста (если нужно)
// func getPermissionsFromContext(r *http.Request) ([]string, bool) {
//	perms, ok := r.Context().Value(PermissionsCtxKey).([]string)
//	return perms, ok
//}
