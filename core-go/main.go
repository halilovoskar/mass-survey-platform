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
	// Единый обработчик для /tests - поддерживает GET и POST
	http.HandleFunc("/tests", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromContext(r)
		if !ok {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		switch r.Method {
		case http.MethodGet:
			// Возвращаем список всех тестов пользователя
			var userTests []Test
			for _, test := range tests {
				if test.OwnerID == userID {
					userTests = append(userTests, test)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userTests)

		case http.MethodPost:
			// Создаём новый тест
			var input struct {
				Title string `json:"title"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, "Некорректный JSON", http.StatusBadRequest)
				return
			}
			if input.Title == "" {
				http.Error(w, "Поле title обязательно", http.StatusBadRequest)
				return
			}

			// Генерируем новый ID и сохраняем
			test := Test{
				ID:      nextTestID,
				Title:   input.Title,
				OwnerID: userID,
			}

			tests[nextTestID] = test
			nextTestID++

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(test)

		default:
			// Метод не поддерживается
			http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		}
	}))

	log.Println("Server started on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
