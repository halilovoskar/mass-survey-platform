package main

// Test представляет опрос или тест
type Test struct {
	ID      int    `json:"test_id"`
	Title   string `json:"test_title"`
	OwnerID int    `json:"creator_id"`
}

// Глобальные переменные — тоже можно здесь (или в main.go, но тогда ПОСЛЕ type)

var tests = map[int]Test{
	1: {ID: 1, Title: "Первый опрос", OwnerID: 100},
	2: {ID: 2, Title: "Тест по Go", OwnerID: 100},
	3: {ID: 3, Title: "Тест по C++", OwnerID: 100},
	4: {ID: 4, Title: "Тест на натурала", OwnerID: 100},
}

var nextTestID = 5
