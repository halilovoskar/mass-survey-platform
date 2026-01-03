// main.go
package main

import (
	"log"

	"core-go/database"
	"core-go/routes"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// 1. Инициализация базы данных
	database.InitDB()

	// 2. Создание Fiber-приложения
	app := fiber.New()

	// 3. Регистрация всех маршрутов из сценария
	routes.Setup(app)

	// 4. Запуск сервера
	log.Println("🚀 Survey Core запущен на http://localhost:8080")
	log.Fatal(app.Listen(":8080"))
}
