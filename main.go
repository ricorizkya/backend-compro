package main

import (
	"backend-go/internal/database"
	"backend-go/internal/handlers"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	err := database.InitDB("postgres://postgres:root@localhost:5432/dashboardlj?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	app := fiber.New()
	
	app.Use(logger.New())
	
	userHandler := handlers.NewUserHandler(database.DB)

	app.Post("/users", userHandler.CreateUser)
	app.Put("/users/:id", userHandler.UpdateUser)

	log.Fatal(app.Listen(":3000"))
}