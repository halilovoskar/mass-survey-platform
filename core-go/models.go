package main

// Test представляет опрос или тест
type Test struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	OwnerID int    `json:"owner_id"`
}

// Глобальные переменные — тоже можно здесь (или в main.go, но тогда ПОСЛЕ type)

var tests = map[int]Test{
	1: {ID: 1, Title: "Первый опрос", OwnerID: 100},
	2: {ID: 2, Title: "Тест по Go", OwnerID: 100},
}

//var nextTestID = 3
