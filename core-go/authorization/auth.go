package authorization

import (
	"fmt"
	"slices"
)

func ParseJWT(tokenStr string) (userID int, permissions []string, err error) {
	if tokenStr != "dummy-token" {
		return 0, nil, fmt.Errorf("неверный токен")
	}
	return 100, []string{
		//"test:list:read",
		//"test:create:write",
		"test:quest:add",
		"test:answer:read",
		"course:test:add",
	}, nil
}

func HasPermission(perms any, required string) bool {
	permissions, ok := perms.([]string)
	if !ok {
		return false
	}
	return slices.Contains(permissions, required)
}
