// main.go
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
			if !hasPermission(r, "test:list:read") {
				http.Error(w, "Недостаточно прав", http.StatusForbidden)
				return
			}
			var userTests []Test
			// ← Показываем только не удалённые тесты
			DB.Where("owner_id = ? AND deleted_at IS NULL", userID).Find(&userTests)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userTests)

		case http.MethodPost:
			var input struct {
				Title    string `json:"title"`
				CourseID int    `json:"course_id"` // ← ОБЯЗАТЕЛЬНО из сценария
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, "Некорректный JSON", http.StatusBadRequest)
				return
			}
			if input.Title == "" {
				http.Error(w, "Поле title обязательно", http.StatusBadRequest)
				return
			}
			if input.CourseID <= 0 {
				http.Error(w, "Поле course_id обязательно и должно быть > 0", http.StatusBadRequest)
				return
			}

			// ← Проверяем право на добавление теста в курс
			if !hasPermission(r, "course:test:add") {
				http.Error(w, "Недостаточно прав для добавления теста в курс", http.StatusForbidden)
				return
			}

			var existingTest Test
			if err := DB.Where("title = ? AND owner_id = ? AND course_id = ?", input.Title, userID, input.CourseID).First(&existingTest).Error; err == nil {
				http.Error(w, "Тест с таким названием уже существует в этом курсе", http.StatusBadRequest)
				return
			}

			test := Test{
				Title:    input.Title,
				OwnerID:  userID,
				CourseID: input.CourseID,
				Status:   "inactive", // ← По сценарию: по умолчанию НЕактивен
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

	// 3. Эндпоинт /tests/{test_id}/questions (POST) — ДОБАВЛЕНИЕ ВОПРОСА
	r.HandleFunc("/tests/{test_id}/questions", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromContext(r)
		if !ok {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		testID, err := strconv.Atoi(mux.Vars(r)["test_id"])
		if err != nil {
			http.Error(w, "Некорректный ID теста", http.StatusBadRequest)
			return
		}

		// ← Проверяем, что тест существует, принадлежит пользователю и НЕ был пройден
		var test Test
		if err := DB.Where("id = ? AND owner_id = ? AND deleted_at IS NULL", testID, userID).First(&test).Error; err != nil {
			http.Error(w, "Тест не найден или доступ запрещён", http.StatusForbidden)
			return
		}

		// ← Запрещаем изменять тест, если уже есть попытки
		var attemptCount int64
		DB.Model(&Answer{}).
			Joins("JOIN questions ON answers.question_id = questions.id").
			Where("questions.test_id = ?", testID).
			Count(&attemptCount)
		if attemptCount > 0 {
			http.Error(w, "Нельзя изменять тест после начала прохождения", http.StatusConflict)
			return
		}

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

	// 4. Эндпоинт /tests/{test_id}/questions (GET) — ПОЛУЧЕНИЕ ВОПРОСОВ
	r.HandleFunc("/tests/{test_id}/questions", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromContext(r)
		if !ok {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		testID, err := strconv.Atoi(mux.Vars(r)["test_id"])
		if err != nil {
			http.Error(w, "Некорректный ID теста", http.StatusBadRequest)
			return
		}

		// ← Проверяем доступ: либо владелец, либо студент в курсе (упрощённо — только владелец пока)
		var test Test
		if err := DB.Where("id = ? AND deleted_at IS NULL", testID).First(&test).Error; err != nil {
			http.Error(w, "Тест не найден", http.StatusNotFound)
			return
		}

		isOwner := (userID == test.OwnerID)
		if !isOwner && !hasPermission(r, "course:test:read") {
			http.Error(w, "Доступ запрещён", http.StatusForbidden)
			return
		}

		var questions []Question
		DB.Where("test_id = ?", testID).Find(&questions)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(questions)
	})).Methods("GET")

	// 5. Эндпоинт /answers (POST) — ОТПРАВКА ОТВЕТОВ
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

		firstQID := answers[0].QuestionID
		var question Question
		if err := DB.Where("id = ?", firstQID).First(&question).Error; err != nil {
			http.Error(w, "Вопрос не найден", http.StatusNotFound)
			return
		}

		// ← Проверка: тест активен и не удалён?
		var test Test
		if err := DB.Where("id = ? AND status = ? AND deleted_at IS NULL", question.TestID, "active").First(&test).Error; err != nil {
			http.Error(w, "Тест неактивен или не существует", http.StatusForbidden)
			return
		}

		// ← Проверка: уже есть попытка?
		var existingCount int64
		DB.Model(&Answer{}).
			Where("user_id = ? AND question_id IN (?)",
				userID,
				DB.Select("id").Where("test_id = ?", test.ID).Table("questions")).
			Count(&existingCount)
		if existingCount > 0 {
			http.Error(w, "Вы уже проходили этот тест", http.StatusConflict)
			return
		}

		// ← ТРАНЗАКЦИЯ
		tx := DB.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()
		if tx.Error != nil {
			http.Error(w, "Ошибка транзакции", http.StatusInternalServerError)
			return
		}

		for _, ans := range answers {
			// ← Валидация: вопрос принадлежит тесту
			var q Question
			if err := tx.Where("id = ? AND test_id = ?", ans.QuestionID, test.ID).First(&q).Error; err != nil {
				tx.Rollback()
				http.Error(w, "Вопрос не принадлежит тесту", http.StatusBadRequest)
				return
			}

			answer := Answer{
				UserID:     userID,
				QuestionID: ans.QuestionID,
				Value:      ans.Value,
			}
			if err := tx.Create(&answer).Error; err != nil {
				tx.Rollback()
				log.Printf("Ошибка создания ответа: %v", err)
				http.Error(w, "Ошибка сохранения", http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit().Error; err != nil {
			http.Error(w, "Ошибка фиксации", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"test_id": test.ID,
			"user_id": userID,
		})
	})).Methods("POST")

	// 6. Эндпоинт /tests/{test_id}/results (GET) — ПРОСМОТР РЕЗУЛЬТАТОВ
	r.HandleFunc("/tests/{test_id}/results", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromContext(r)
		if !ok {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		testID, err := strconv.Atoi(mux.Vars(r)["test_id"])
		if err != nil {
			http.Error(w, "Некорректный ID теста", http.StatusBadRequest)
			return
		}

		var test Test
		if err := DB.Select("id, owner_id, course_id").Where("id = ? AND deleted_at IS NULL", testID).First(&test).Error; err != nil {
			http.Error(w, "Тест не найден", http.StatusNotFound)
			return
		}

		isOwner := (userID == test.OwnerID)
		canViewAll := isOwner && hasPermission(r, "test:answer:read")
		canViewOwn := !isOwner // студент

		var targetUserIDs []int
		if canViewAll {
			// Все, кто проходил тест
			DB.Model(&Answer{}).
				Where("question_id IN (?)",
					DB.Select("id").Where("test_id = ?", testID).Table("questions")).
				Pluck("DISTINCT user_id", &targetUserIDs)
		} else if canViewOwn {
			targetUserIDs = []int{userID}
		} else {
			http.Error(w, "Недостаточно прав", http.StatusForbidden)
			return
		}

		var allAnswers []Answer
		DB.Where("user_id IN ? AND question_id IN ?",
			targetUserIDs,
			DB.Select("id").Where("test_id = ?", testID).Table("questions")).
			Find(&allAnswers)

		userAnswers := make(map[int]map[int]string)
		for _, a := range allAnswers {
			if _, ok := userAnswers[a.UserID]; !ok {
				userAnswers[a.UserID] = make(map[int]string)
			}
			userAnswers[a.UserID][a.QuestionID] = a.Value
		}

		var questions []Question
		DB.Where("test_id = ?", testID).Find(&questions)
		qText := make(map[int]string)
		for _, q := range questions {
			qText[q.ID] = q.Text
		}

		type ResultItem struct {
			UserID  int `json:"user_id"`
			Answers []struct {
				QuestionText string `json:"question_text"`
				Value        string `json:"value"`
			} `json:"answers"`
		}

		var results []ResultItem
		for _, uid := range targetUserIDs {
			if ansMap, exists := userAnswers[uid]; exists {
				var list []struct {
					QuestionText string `json:"question_text"`
					Value        string `json:"value"`
				}
				for _, q := range questions {
					if val, ok := ansMap[q.ID]; ok {
						list = append(list, struct {
							QuestionText string `json:"question_text"`
							Value        string `json:"value"`
						}{
							QuestionText: qText[q.ID],
							Value:        val,
						})
					}
				}
				results = append(results, ResultItem{UserID: uid, Answers: list})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})).Methods("GET")

	// Запуск сервера
	log.Println("🚀 Server started on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
