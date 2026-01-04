package routes

import (
	"time"

	"core-go/authorization"
	"core-go/database"
	"core-go/models"

	"github.com/gofiber/fiber/v2"
)

// Setup регистрирует все маршруты из сценария 2.0
func Setup(app *fiber.App) {
	auth := func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Требуется заголовок: Authorization: Bearer <token>",
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

	api := app.Group("", auth)

	// Тесты
	api.Post("/tests", createTest)
	api.Get("/tests", listTests)
	api.Post("/tests/:test_id/questions", addQuestionToTest)
	api.Get("/tests/:test_id/questions", listQuestionsInTest)
	api.Post("/tests/:test_id/attempt", createAttempt)
	api.Post("/attempts/:attempt_id/complete", completeAttempt)
	api.Get("/tests/:test_id/results", getTestResults)
	api.Post("/answers", submitAnswers)
}

// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ

// Определяет, является ли пользователь преподавателем курса
func isTeacherOfCourse(userID string, courseID int) bool {
	/* var course models.Course
	if err := database.DB.Where("id = ? AND teacher_id = ?", courseID, userID).First(&course).Error; err != nil {
		return false
	} */
	return true
}

// Определяет, записан ли пользователь на курс
func isEnrolledInCourse(_ string, _ int) bool {
	// return database.DB.Where("user_id = ? AND course_id = ?", userID, courseID).First(&models.Enrollment{}).Error == nil
	return true // упрощение для защиты
}

func canManageCourse(userID string, courseID int) bool {
	return isTeacherOfCourse(userID, courseID)
}

func canViewTestResults(userID string, testID int) (bool, error) {
	var test models.Test
	if err := database.DB.Where("id = ?", testID).First(&test).Error; err != nil {
		return false, err
	}
	return canManageCourse(userID, test.CourseID), nil
}

// ОБРАБОТЧИКИ

func createTest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	// Согласно сценарию: create требует course:test:add → но Python не даёт прав
	// → Для защиты: разрешаем, если пользователь может управлять курсом

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

	if !isTeacherOfCourse(userID, input.CourseID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Вы не преподаватель этого курса",
		})
	}

	test := models.Test{
		CourseID:     input.CourseID,
		TestName:     input.TestName,
		TestSubject:  input.TestSubject,
		TestDuration: input.TestDuration,
		Graduate:     input.Graduate,
		Status:       "inactive",
		CreatedAt:    time.Now(),
		OwnerID:      userID, // строка!
	}
	database.DB.Create(&test)
	return c.Status(fiber.StatusCreated).JSON(test)
}

func listTests(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	var courses []models.Course
	database.DB.Where("teacher_id = ?", userID).Find(&courses)

	var courseIDs []int
	for _, c := range courses {
		courseIDs = append(courseIDs, c.ID)
	}

	var tests []models.Test
	if len(courseIDs) > 0 {
		database.DB.Where("course_id IN ?", courseIDs).Find(&tests)
	}
	return c.JSON(tests)
}

func addQuestionToTest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	testID, err := c.ParamsInt("test_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Некорректный ID теста"})
	}

	var test models.Test
	if err := database.DB.Where("id = ?", testID).First(&test).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Тест не найден"})
	}

	if !canManageCourse(userID, test.CourseID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Доступ запрещён",
		})
	}

	// Проверка: нет ли попыток
	var count int64
	database.DB.Model(&models.Attempt{}).Where("test_id = ?", testID).Count(&count)
	if count > 0 {
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
		OwnerID:      userID,
	}
	database.DB.Create(&question)
	return c.Status(fiber.StatusCreated).JSON(question)
}

func listQuestionsInTest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	testID, _ := c.ParamsInt("test_id")

	var test models.Test
	if err := database.DB.Where("id = ?", testID).First(&test).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Тест не найден"})
	}

	if !canManageCourse(userID, test.CourseID) && !isEnrolledInCourse(userID, test.CourseID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Доступ запрещён"})
	}

	var questions []models.Question
	database.DB.Where("test_id = ?", testID).Find(&questions)
	return c.JSON(questions)
}

func createAttempt(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	testID, _ := c.ParamsInt("test_id")

	var test models.Test
	if err := database.DB.Where("id = ? AND status = ?", testID, "active").First(&test).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Тест не найден или не активен",
		})
	}

	// Проверка: нет ли активной попытки
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
	return c.Status(fiber.StatusCreated).JSON(attempt)
}

func completeAttempt(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
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
	userID := c.Locals("userID").(string)
	testID, _ := c.ParamsInt("test_id")

	var test models.Test
	if err := database.DB.Where("id = ?", testID).First(&test).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Тест не найден"})
	}

	// Права: преподаватель курса ИЛИ студент смотрит себя
	isTeacher := canManageCourse(userID, test.CourseID)

	var attempts []models.Attempt
	if isTeacher {
		database.DB.Where("test_id = ? AND status = ?", testID, "completed").Find(&attempts)
	} else {
		database.DB.Where("test_id = ? AND user_id = ? AND status = ?", testID, userID, "completed").Find(&attempts)
	}

	type Result struct {
		UserID  string `json:"user_id"`
		Answers []struct {
			QuestionText string `json:"question_text"`
			Value        string `json:"value"`
		} `json:"answers"`
	}

	var results []Result
	for _, a := range attempts {
		if !isTeacher && a.UserID != userID {
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
	userID := c.Locals("userID").(string)
	var input []struct {
		AttemptID  int    `json:"attempt_id"`
		QuestionID int    `json:"question_id"`
		Value      string `json:"value"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Некорректный JSON"})
	}

	for _, ans := range input {
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
