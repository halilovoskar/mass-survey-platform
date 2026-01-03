package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID        int    `gorm:"primaryKey" json:"user_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	RoleID    int    `json:"role_id"`
}

type Role struct {
	gorm.Model
	ID       int    `gorm:"primaryKey" json:"role_id"`
	RoleName string `json:"role_name"`
}

type Course struct {
	gorm.Model
	ID          int    `gorm:"primaryKey" json:"course_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	TeacherID   int    `json:"teacher_id"`
}

type Test struct {
	gorm.Model
	ID           int       `gorm:"primaryKey" json:"test_id"`
	TestName     string    `json:"test_name"`
	TestSubject  string    `json:"test_subject"`
	TestDuration int       `json:"test_duration"`
	Graduate     bool      `json:"graduate"`
	Status       string    `json:"status" gorm:"default:'inactive'"`
	CourseID     int       `json:"course_id"`
	CreatedAt    time.Time `json:"created_id,omitempty"`
	UpdatedAt    time.Time `json:"-"`
	DeletedAt    time.Time `json:"-"`
}

type Question struct {
	gorm.Model
	ID           int    `gorm:"primaryKey" json:"question_id"`
	TestID       int    `json:"test_id"`
	QuestionName string `json:"question_name"`
	QuestionType string `json:"question_type"`
	IsMultiple   bool   `json:"is_multiple"`
	QuestionText string `json:"question_text"`
}

type Answer struct {
	gorm.Model
	ID         int    `gorm:"primaryKey" json:"answer_id"`
	AnswerText string `json:"answer_text"`
	QuestionID int    `json:"question_id"`
	AnswerType string `json:"answer_type"`
}

type Attempt struct {
	gorm.Model
	ID          int       `gorm:"primaryKey" json:"attempt_id"`
	UserID      int       `json:"user_id"`
	TestID      int       `json:"test_id"`
	Status      string    `json:"status"` // "active", "completed"
	Score       float64   `json:"score"`  // 0.0 to 100.0
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

type UserAnswers struct {
	gorm.Model
	ID         int    `gorm:"primaryKey" json:"id"`
	UserID     int    `json:"user_id"`
	AttemptID  int    `json:"attempt_id"`
	QuestionID int    `json:"question_id"`
	AnswerID   int    `json:"answer_id"`
	Value      string `json:"value"` // для текстовых ответов
}
