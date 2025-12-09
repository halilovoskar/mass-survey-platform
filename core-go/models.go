package main

import (
	"time"

	"gorm.io/gorm"
)

type Attempt struct {
	ID          int       `json:"attemted_id" gorm:"primaryKey"`
	UserID      int       `json:"user_id"`
	TestID      int       `json:"test_id"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}
type Test struct {
	ID      int    `json:"test_id" gorm:"primaryKey"`
	Title   string `json:"test_title"`
	OwnerID int    `json:"creator_id"`
	Status  string `json:"status"` // active or inactive
}

type Question struct {
	ID     int    `json:"question_id" gorm:"primaryKey"`
	TestID int    `json:"test_id"`
	Text   string `json:"text"`
	Type   string `json:"type"` // "single", "text", etc.
}

type Answer struct {
	ID         int    `json:"answer_id" gorm:"primaryKey"`
	UserID     int    `json:"user_id"`
	QuestionID int    `json:"question_id"`
	Value      string `json:"value"`
}

var _ gorm.DeletedAt
