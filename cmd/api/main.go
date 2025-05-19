package main

import (
	"log"
	"os"
	"time"

	"github.com/chachabrian/mooveit-backend/internal/database"
	"github.com/chachabrian/mooveit-backend/internal/handlers"
	"github.com/chachabrian/mooveit-backend/internal/middleware"
	"github.com/gin-contrib/cors"
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

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	// Serve static files
	r.Static("/uploads", "/app/uploads")
	r.Static("/static", "./static")

	// Routes
	api := r.Group("/api")
	{
		// Public routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register(db))
			auth.POST("/login", handlers.Login(db))
			auth.POST("/forgot-password", handlers.RequestPasswordReset(db))
			auth.POST("/verify-otp", handlers.VerifyOTP(db))
			auth.POST("/reset-password", handlers.ResetPassword(db))
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

	// Fix: Explicitly bind to all network interfaces
	if err := r.Run("0.0.0.0:" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}