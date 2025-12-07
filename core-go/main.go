package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	// 1. Эндпоинт /health
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{
			"status": "ok",
			"module": "survey_core",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	// 2. Эндпоинт /tests (GET и POST)
	r.HandleFunc("/tests", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromContext(r)
		if !ok {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		switch r.Method {
		case http.MethodGet:
			var userTests []Test
			DB.Where("owner_id = ?", userID).Find(&userTests)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userTests)

		case http.MethodPost:
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

			var existingTest Test
			if err := DB.Where("title = ? AND owner_id = ?", input.Title, userID).First(&existingTest).Error; err == nil {
				http.Error(w, "Тест с таким названием уже существует", http.StatusBadRequest)
				return
			}

			test := Test{
				Title:   input.Title,
				OwnerID: userID,
			}
			if err := DB.Create(&test).Error; err != nil {
				http.Error(w, "Ошибка сохранения теста", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(test)

		default:
			http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		}
	})).Methods("GET", "POST")

	// 3. Эндпоинт /tests/{test_id}/questions (POST)
	r.HandleFunc("/tests/{test_id}/questions", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromContext(r)
		if !ok {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		// Извлекаем test_id из URL
		vars := mux.Vars(r)
		testIDStr := vars["test_id"]
		testID, err := strconv.Atoi(testIDStr)
		if err != nil {
			http.Error(w, "Некорректный ID теста", http.StatusBadRequest)
			return
		}

		// Проверяем, что тест принадлежит пользователю
		var test Test
		if err := DB.Where("id = ? AND owner_id = ?", testID, userID).First(&test).Error; err != nil {
			http.Error(w, "Тест не найден или доступ запрещён", http.StatusForbidden)
			return
		}

		// Читаем вопрос из тела запроса
		var input struct {
			Text string `json:"text"`
			Type string `json:"type"` // "single", "text"
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Некорректный JSON", http.StatusBadRequest)
			return
		}
		if input.Text == "" {
			http.Error(w, "Поле text обязательно", http.StatusBadRequest)
			return
		}

		// Создаём вопрос
		question := Question{
			TestID: testID,
			Text:   input.Text,
			Type:   input.Type,
		}
		if err := DB.Create(&question).Error; err != nil {
			http.Error(w, "Ошибка сохранения вопроса", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(question)
	})).Methods("POST")

	// 4. Эндпоинт /tests/{test_id}/questions (GET)
	r.HandleFunc("/tests/{test_id}/questions", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromContext(r)
		if !ok {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		testIDStr := vars["test_id"]
		testID, err := strconv.Atoi(testIDStr)
		if err != nil {
			http.Error(w, "Некорректный ID теста", http.StatusBadRequest)
			return
		}

		// Проверяем доступ к тесту (либо владелец, либо участник — для простоты пока только владелец)
		var test Test
		if err := DB.Where("id = ? AND owner_id = ?", testID, userID).First(&test).Error; err != nil {
			// Можно расширить: проверить, записан ли пользователь на тест
			http.Error(w, "Тест не найден или доступ запрещён", http.StatusForbidden)
			return
		}

		var questions []Question
		DB.Where("test_id = ?", testID).Find(&questions)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(questions)
	})).Methods("GET")

	// 5. Эндпоинт /answers (POST) — отправка ответов пользователя
	r.HandleFunc("/answers", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
			return
		}

		userID, ok := getUserIDFromContext(r)
		if !ok {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		// Ожидаем массив ответов: [{"question_id":1,"value":"A"}, ...]
		var answers []struct {
			QuestionID int    `json:"question_id"`
			Value      string `json:"value"`
		}

		if err := json.NewDecoder(r.Body).Decode(&answers); err != nil {
			http.Error(w, "Некорректный JSON", http.StatusBadRequest)
			return
		}

		if len(answers) == 0 {
			http.Error(w, "Массив ответов пуст", http.StatusBadRequest)
			return
		}

		// Сохраняем каждый ответ
		for _, ans := range answers {
			answer := Answer{
				UserID:     userID,
				QuestionID: ans.QuestionID,
				Value:      ans.Value,
			}
			if err := DB.Create(&answer).Error; err != nil {
				// Можно логировать, но не прерывать весь массив
				log.Printf("Ошибка сохранения ответа: %v", err)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"count":   len(answers),
			"user_id": userID,
		})
	})).Methods("POST")

	// Запуск сервера
	log.Println("Server started on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r)) // ← ВАЖНО: не nil, а r
}
