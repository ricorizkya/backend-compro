package main

import (
	"backend-go/internal/database"
	"backend-go/internal/handlers"
	"backend-go/internal/middleware"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found or error loading, using system environment variables")
	}

	// Setup database connection
	dbConnString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	err = database.InitDB(dbConnString)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.CloseDB()

	app := fiber.New()

	// Middleware
	app.Use(logger.New())

	// Initialize handlers
	userHandler := handlers.NewUserHandler(database.DB)
	authHandler := handlers.NewAuthHandler(database.DB)
	carouselHandler := handlers.NewCarouselHandler(database.DB)
	productHandler := handlers.NewProductHandler(database.DB)

	// Routes
	app.Post("/login", authHandler.Login)

	// Coba implementasi tanpa group
	// app.Get("/users", middleware.AuthMiddleware, userHandler.GetUsers)
	
	// Protected routes
	protected := app.Group("", middleware.AuthMiddleware)
	{
		protected.Post("/logout", authHandler.Logout)
		protected.Get("/users", userHandler.GetUsers)
		protected.Post("/users", userHandler.CreateUser)
		protected.Put("/users/:id", userHandler.UpdateUser)
		protected.Delete("/users/:id", userHandler.DeleteUser)

		protected.Post("/carousel", carouselHandler.CreateCarousel)
		protected.Put("/carousel/:id", carouselHandler.UpdateCarousel)
		protected.Delete("/carousel/:id", carouselHandler.DeleteCarousel)
		protected.Get("/carousel", carouselHandler.GetCarousels)

		protected.Post("/products", productHandler.CreateProduct)
		protected.Put("/products/:id", productHandler.UpdateProduct)
		protected.Delete("/products/:id", productHandler.DeleteProduct)
		protected.Get("/products", productHandler.GetProducts)
		// adminGroup := protected.Group("", middleware.AdminMiddleware)
		// adminGroup.Post("/carousel", carouselHandler.CreateCarousel)
	}

	// Start server
	log.Fatal(app.Listen(":" + os.Getenv("PORT")))
}