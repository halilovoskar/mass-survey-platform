// models.go
package main

import "gorm.io/gorm"

// Test — тест, привязанный к дисциплине
type Test struct {
	ID        int            `json:"test_id" gorm:"primaryKey"`
	Title     string         `json:"test_title"`
	OwnerID   int            `json:"creator_id"` // ID преподавателя
	CourseID  int            `json:"course_id"`  // ← ОБЯЗАТЕЛЬНО: привязка к дисциплине
	Status    string         `json:"status"`     // "active" или "inactive"
	DeletedAt gorm.DeletedAt `json:"-"`          // ← Для soft delete
}

// Question — вопрос в тесте
type Question struct {
	ID     int    `json:"question_id" gorm:"primaryKey"`
	TestID int    `json:"test_id" gorm:"index"` // ← индекс для производительности
	Text   string `json:"text"`
	Type   string `json:"type"` // "single", "text" и т.д.
}

// Answer — ответ пользователя на вопрос
type Answer struct {
	ID         int    `json:"answer_id" gorm:"primaryKey"`
	UserID     int    `json:"user_id" gorm:"index"`     // ← индекс для поиска по пользователю
	QuestionID int    `json:"question_id" gorm:"index"` // ← индекс для поиска по вопросу
	Value      string `json:"value"`                    // текст или индекс варианта
}
