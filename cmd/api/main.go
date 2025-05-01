package main

import (
	"log"
	"os"
	"time"

	"github.com/chachabrian/mooveit-backend/internal/database"
	"github.com/chachabrian/mooveit-backend/internal/handlers"
	"github.com/chachabrian/mooveit-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize database with better error handling
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Get underlying SQL DB instance
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Initialize router
	r := gin.Default()

	// Serve static files
	r.Static("/uploads", "/app/uploads")

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Routes
	api := r.Group("/api")
	{
		// Public routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register(db))
			auth.POST("/login", handlers.Login(db))
		}

		// Protected routes
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			// User routes
			users := protected.Group("/users")
			{
				users.GET("/profile", handlers.GetProfile(db))
				users.PUT("/profile", handlers.UpdateProfile(db))
			}

			// Rides routes
			rides := protected.Group("/rides")
			{
				rides.GET("", handlers.GetAvailableRides(db))
				rides.POST("", handlers.CreateRide(db))
				rides.GET("/driver", handlers.GetDriverRides(db))
				rides.GET("/all", handlers.GetAllRides(db))
				rides.DELETE("/:id", handlers.DeleteRide(db))
			}

			// Bookings routes
			bookings := protected.Group("/bookings")
			{
				bookings.POST("", handlers.CreateBooking(db))
				bookings.GET("/:id/status", handlers.GetBookingStatus(db))
				bookings.GET("/client", handlers.GetClientBookings(db))
				bookings.GET("/driver", handlers.GetDriverBookings(db))
				bookings.PATCH("/:id/status", handlers.UpdateBookingStatus(db))
				bookings.GET("/:id/parcel-details", handlers.GetParcelDetails(db)) // Updated route
			}

			parcels := protected.Group("/parcels")
			{
				parcels.POST("", handlers.CreateParcel(db))
			}
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
