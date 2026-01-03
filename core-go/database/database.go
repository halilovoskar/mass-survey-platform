package database

import (
	"core-go/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := "host=localhost user=postgres password=postgres dbname=survey_db port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Не удалось подключиться к БД")
	}

	// Автомиграция
	db.AutoMigrate(&models.User{}, &models.Role{}, &models.Course{}, &models.Test{}, &models.Question{}, &models.Answer{}, &models.Attempt{}, &models.UserAnswers{})

	DB = db
}
