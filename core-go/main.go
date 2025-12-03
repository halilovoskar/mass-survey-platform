package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{
			"status": "ok",
			"module": "survey_core",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	// GET /tests - возвращает список всех тестов
	http.HandleFunc("/tests", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
			return
		}
		// Преобразуем map в срез
		testList := make([]Test, 0, len(tests))
		for _, test := range tests {
			testList = append(testList, test)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testList)
	})

	log.Println("Server started on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
