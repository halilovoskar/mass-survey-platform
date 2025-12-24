// db.go
package main

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func init() {
	// ← Используем переменные окружения для безопасности
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=survey_db port=5432 sslmode=disable"
		log.Println("⚠️  DATABASE_URL не задан, использую локальное подключение")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // или logger.Info для отладки
	})
	if err != nil {
		log.Fatal("❌ Не удалось подключиться к PostgreSQL:", err)
	}

	// Автоматическая миграция
	err = DB.AutoMigrate(&Test{}, &Question{}, &Answer{})
	if err != nil {
		log.Fatal("❌ Ошибка миграции:", err)
	}

	log.Println("✅ Подключение к PostgreSQL установлено. Таблицы синхронизированы.")
}
