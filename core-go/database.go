package main

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	// Строка подключения к PostgreSQL
	dsn := "host=localhost user=postgres password=postgres dbname=survey_db port=5432 sslmode=disable"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Не удалось подключиться к PostgreSQL:", err)
	}
	log.Println("Успешное подключение к PostgreSQL")

	// Автоматически создаём таблицы, если их нет
	DB.AutoMigrate(&Test{}, &Question{}, &Answer{})
	log.Println("Таблицы созданы (если не существовали)")
}
