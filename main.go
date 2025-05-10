package main

import (
	"backend-go/internal/database"
	"backend-go/internal/handlers"
	"backend-go/internal/middleware"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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

	// Inisialisasi Fiber
	app := fiber.New()

	// Middleware CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "*",
		AllowCredentials: false,
	}))

	// Middleware Logger
	app.Use(logger.New())

	// Serve static files (Fiber way)
	app.Static("/uploads", "./uploads")

	// Initialize handlers
	userHandler := handlers.NewUserHandler(database.DB)
	authHandler := handlers.NewAuthHandler(database.DB)
	carouselHandler := handlers.NewCarouselHandler(database.DB)
	productHandler := handlers.NewProductHandler(database.DB)
	portfolioImagesHandler := handlers.NewPortfolioHandler(database.DB)
	portfolioReviewsHandler := handlers.NewPortfolioHandler(database.DB)
	messagesHandler := handlers.NewMessagesHandler(database.DB)

	// Routes
	app.Post("/register", userHandler.RegisterUser)
	app.Post("/login", authHandler.Login)

	// Protected routes
	protected := app.Group("", middleware.AuthMiddleware)
	{
		// Users
		protected.Post("/logout", authHandler.Logout)
		protected.Get("/users", userHandler.GetUsers)
		protected.Get("/users/:id", userHandler.GetUserByID)
		protected.Post("/users", userHandler.CreateUser)
		protected.Put("/users/:id", userHandler.UpdateUser)
		protected.Delete("/users/:id", userHandler.DeleteUser)

		// Carousels
		protected.Post("/carousel", carouselHandler.CreateCarousel)
		protected.Put("/carousel/:id", carouselHandler.UpdateCarousel)
		protected.Delete("/carousel/:id", carouselHandler.DeleteCarousel)
		protected.Get("/carousel", carouselHandler.GetCarousels)
		protected.Get("/carousel/:id", carouselHandler.GetCarouselByID)

		// Products
		protected.Post("/products", productHandler.CreateProduct)
		protected.Put("/products/:id", productHandler.UpdateProduct)
		protected.Delete("/products/:id", productHandler.DeleteProduct)
		protected.Get("/products", productHandler.GetProducts)
		protected.Get("/products/:id", productHandler.GetProductByID)

		// Portfolio Images
		protected.Post("/portfolio/images", portfolioImagesHandler.CreatePortfolioImage)
		protected.Put("/portfolio/images/:id", portfolioImagesHandler.UpdatePortfolioImage)
		protected.Delete("/portfolio/images/:id", portfolioImagesHandler.DeletePortfolioImage)
		protected.Get("/portfolio/images", portfolioImagesHandler.GetPortfolioImages)
		protected.Get("/portfolio/images/:id", portfolioImagesHandler.GetPortfolioImageByID)

		// Portfolio Reviews
		protected.Post("/portfolio/reviews", portfolioReviewsHandler.CreatePortfolioReview)
		protected.Put("/portfolio/reviews/:id", portfolioReviewsHandler.UpdatePortfolioReview)
		protected.Delete("/portfolio/reviews/:id", portfolioReviewsHandler.DeletePortfolioReview)
		protected.Get("/portfolio/reviews", portfolioReviewsHandler.GetPortfolioReviews)
		protected.Get("/portfolio/reviews/:id", portfolioReviewsHandler.GetPortfolioReviewByID)

		// Messages
		protected.Post("/messages", messagesHandler.CreateMessage)
		protected.Put("/messages/:id", messagesHandler.UpdateMessage)
		protected.Delete("/messages/:id", messagesHandler.DeleteMessage)
		protected.Get("/messages", messagesHandler.GetMessages)
		protected.Get("/messages/:id", messagesHandler.GetMessageByID)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // Default port
	}

	log.Printf("Server running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
