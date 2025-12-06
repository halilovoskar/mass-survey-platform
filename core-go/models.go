package main

import (
	"gorm.io/gorm"
)

type Test struct {
	ID      int    `json:"test_id" gorm:"primaryKey"`
	Title   string `json:"test_title"`
	OwnerID int    `json:"creator_id"`
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
