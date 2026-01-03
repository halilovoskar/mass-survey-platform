package routes

import (
	"time"

	"core-go/authorization"
	"core-go/database"
	"core-go/models"

	"github.com/gofiber/fiber/v2"
)

// Setup регистрирует все маршруты из сценария
func Setup(app *fiber.App) {
	// Middleware: проверка токена
	auth := func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Требуется JWT токен в заголовке: Authorization: Bearer <token>",
			})
		}

		tokenStr := authHeader[7:]
		userID, perms, err := authorization.ParseJWT(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Невалидный или просроченный токен",
			})
		}

		c.Locals("userID", userID)
		c.Locals("permissions", perms)
		return c.Next()
	}

	// Все эндпоинты защищены
	api := app.Group("", auth)

	// ============================
	// РЕСУРС: Тесты (Test)
	// ============================

	// POST /tests — Создать тест (требуется course:test:add)
	api.Post("/tests", createTest)

	// GET /tests — Список тестов (только свои — по умолчанию)
	api.Get("/tests", listTests)

	// POST /tests/:test_id/questions — Добавить вопрос (требуется test:quest:add)
	api.Post("/tests/:test_id/questions", addQuestionToTest)

	// GET /tests/:test_id/questions — Список вопросов
	api.Get("/tests/:test_id/questions", listQuestionsInTest)

	// POST /tests/:test_id/attempt — Создать попытку (по умолчанию — только для активного теста)
	api.Post("/tests/:test_id/attempt", createAttempt)

	// POST /attempts/:attempt_id/complete — Завершить попытку
	api.Post("/attempts/:attempt_id/complete", completeAttempt)

	// GET /tests/:test_id/results — Результаты (требуется test:answer:read)
	api.Get("/tests/:test_id/results", getTestResults)

	// POST /answers — Отправить/обновить ответы
	api.Post("/answers", submitAnswers)
}

// --- Реализация обработчиков ---

func createTest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(int)
	perms := c.Locals("permissions").([]string)

	if !authorization.HasPermission(perms, "course:test:add") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Недостаточно прав для создания теста",
		})
	}

	var input struct {
		CourseID     int    `json:"course_id"`
		TestName     string `json:"test_name"`
		TestSubject  string `json:"test_subject"`
		TestDuration int    `json:"test_duration"`
		Graduate     bool   `json:"graduate"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Некорректный JSON"})
	}

	// Проверяем, что пользователь — преподаватель курса
	var course models.Course
	if err := database.DB.Where("id = ? AND teacher_id = ?", input.CourseID, userID).First(&course).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Курс не найден или вы не являетесь его преподавателем",
		})
	}

	test := models.Test{
		CourseID:     input.CourseID,
		TestName:     input.TestName,
		TestSubject:  input.TestSubject,
		TestDuration: input.TestDuration,
		Graduate:     input.Graduate,
		Status:       "inactive", // По умолчанию не активен
		CreatedAt:    time.Now(),
	}
	database.DB.Create(&test)

	return c.Status(201).JSON(test)
}

func listTests(c *fiber.Ctx) error {
	userID := c.Locals("userID").(int)
	var tests []models.Test
	// Пользователь видит только тесты, привязанные к его курсам
	database.DB.Joins("JOIN courses ON courses.id = tests.course_id").
		Where("courses.teacher_id = ?", userID).
		Find(&tests)
	return c.JSON(tests)
}

func addQuestionToTest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(int)
	perms := c.Locals("permissions").([]string)

	if !authorization.HasPermission(perms, "test:quest:add") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Недостаточно прав для добавления вопроса",
		})
	}

	testID, err := c.ParamsInt("test_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Некорректный ID теста"})
	}

	// Проверяем, что тест принадлежит пользователю (через курс)
	var test models.Test
	if err := database.DB.Joins("JOIN courses ON courses.id = tests.course_id").
		Where("tests.id = ? AND courses.teacher_id = ?", testID, userID).
		First(&test).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Тест не найден или доступ запрещён",
		})
	}

	// Проверяем, что ещё нет попыток
	var attemptCount int64
	database.DB.Model(&models.Attempt{}).Where("test_id = ?", testID).Count(&attemptCount)
	if attemptCount > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Нельзя изменять тест после начала попыток",
		})
	}

	var input struct {
		QuestionName string `json:"question_name"`
		QuestionText string `json:"question_text"`
		QuestionType string `json:"question_type"`
		IsMultiple   bool   `json:"is_multiple"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Некорректный JSON"})
	}

	question := models.Question{
		TestID:       testID,
		QuestionName: input.QuestionName,
		QuestionText: input.QuestionText,
		QuestionType: input.QuestionType,
		IsMultiple:   input.IsMultiple,
	}
	database.DB.Create(&question)

	return c.Status(201).JSON(question)
}

func listQuestionsInTest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(int)
	testID, _ := c.ParamsInt("test_id")

	// Проверяем доступ: либо преподаватель курса, либо студент (записан на курс)
	var course models.Course
	if err := database.DB.Joins("JOIN courses ON courses.id = tests.course_id").
		Where("tests.id = ? AND (courses.teacher_id = ? OR courses.id IN (?))",
			testID, userID,
			database.DB.Table("enrollments").Select("course_id").Where("user_id = ?", userID),
		).First(&course).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Доступ к тесту запрещён",
		})
	}

	var questions []models.Question
	database.DB.Where("test_id = ?", testID).Find(&questions)
	return c.JSON(questions)
}

func createAttempt(c *fiber.Ctx) error {
	userID := c.Locals("userID").(int)
	testID, _ := c.ParamsInt("test_id")

	// Проверяем, что тест активен
	var test models.Test
	if err := database.DB.Where("id = ? AND status = ?", testID, "active").First(&test).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Тест не найден или не активен",
		})
	}

	// Проверяем, что у пользователя нет активной попытки
	var existing models.Attempt
	if err := database.DB.Where("user_id = ? AND test_id = ? AND status = ?", userID, testID, "active").First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "У вас уже есть активная попытка",
		})
	}

	attempt := models.Attempt{
		UserID:    userID,
		TestID:    testID,
		Status:    "active",
		StartedAt: time.Now(),
	}
	database.DB.Create(&attempt)

	return c.Status(201).JSON(attempt)
}

func completeAttempt(c *fiber.Ctx) error {
	userID := c.Locals("userID").(int)
	attemptID, _ := c.ParamsInt("attempt_id")

	var attempt models.Attempt
	if err := database.DB.Where("id = ? AND user_id = ? AND status = ?", attemptID, userID, "active").First(&attempt).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Активная попытка не найдена",
		})
	}

	database.DB.Model(&attempt).Updates(models.Attempt{
		Status:      "completed",
		CompletedAt: time.Now(),
	})

	return c.SendStatus(fiber.StatusNoContent)
}

func getTestResults(c *fiber.Ctx) error {
	userID := c.Locals("userID").(int)
	perms := c.Locals("permissions").([]string)
	testID, _ := c.ParamsInt("test_id")

	// Проверяем: либо владелец курса, либо студент смотрит себя
	var course models.Course
	if err := database.DB.Joins("JOIN tests ON tests.course_id = courses.id").
		Where("tests.id = ? AND courses.teacher_id = ?", testID, userID).
		First(&course).Error; err == nil {
		// Владелец курса — разрешено
	} else if authorization.HasPermission(perms, "test:answer:read") {
		// Студент смотрит свои результаты — разрешено
	} else {
		// Нет прав
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Недостаточно прав для просмотра результатов",
		})
	}

	// Получаем завершённые попытки
	var attempts []models.Attempt
	database.DB.Where("test_id = ? AND status = ?", testID, "completed").Find(&attempts)

	// Собираем ответы
	type Result struct {
		UserID  int `json:"user_id"`
		Answers []struct {
			QuestionText string `json:"question_text"`
			Value        string `json:"value"`
		} `json:"answers"`
	}

	var results []Result
	for _, a := range attempts {
		// Позволяем студенту видеть только себя
		if course.TeacherID != userID && a.UserID != userID {
			continue
		}

		var answers []models.UserAnswers
		database.DB.Where("attempt_id = ?", a.ID).Find(&answers)

		var answerList []struct {
			QuestionText string `json:"question_text"`
			Value        string `json:"value"`
		}
		for _, ans := range answers {
			var q models.Question
			database.DB.Where("id = ?", ans.QuestionID).First(&q)
			answerList = append(answerList, struct {
				QuestionText string `json:"question_text"`
				Value        string `json:"value"`
			}{
				QuestionText: q.QuestionText,
				Value:        ans.Value,
			})
		}

		results = append(results, Result{
			UserID:  a.UserID,
			Answers: answerList,
		})
	}

	return c.JSON(results)
}

func submitAnswers(c *fiber.Ctx) error {
	userID := c.Locals("userID").(int)
	var input []struct {
		AttemptID  int    `json:"attempt_id"`
		QuestionID int    `json:"question_id"`
		Value      string `json:"value"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Некорректный JSON"})
	}

	for _, ans := range input {
		// Проверяем, что попытка принадлежит пользователю и активна
		var attempt models.Attempt
		if err := database.DB.Where("id = ? AND user_id = ? AND status = ?", ans.AttemptID, userID, "active").First(&attempt).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Попытка не найдена или не активна",
			})
		}

		userAns := models.UserAnswers{
			UserID:     userID,
			AttemptID:  ans.AttemptID,
			QuestionID: ans.QuestionID,
			Value:      ans.Value,
		}
		database.DB.Create(&userAns)
	}

	return c.JSON(fiber.Map{"status": "ok", "count": len(input)})
}
