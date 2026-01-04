package authorization

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("974e5e96d87be201bf6300c67cb4326814f98ade2c868cf67561305534b240f2")

func ParseJWT(tokenStr string) (userID string, permissions []string, err error) {
	// Убираем "Bearer " если есть
	tokenStr = strings.TrimSpace(strings.TrimPrefix(tokenStr, "Bearer"))

	// Парсим токен
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неподдерживаемый метод подписи: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", nil, fmt.Errorf("невалидный токен: %w", err)
	}

	// Извлекаем claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", nil, fmt.Errorf("claims не MapClaims")
	}

	// Извлекаем user_id (в Python-модуле это строка!)
	userID, _ = claims["sub"].(string)
	if userID == "" {
		return "", nil, fmt.Errorf("поле 'id' отсутствует в токене")
	}

	permissions = []string{
		"course:test:add",
		"test:quest:add",
		"test:answer:read",
		"test:list:read",
		"course:test:write",
		"user:list:read",
	}

	return userID, permissions, nil
}
