package main

import (
	"log"
	"os"
	"time"

	"github.com/chachabrian/mooveit-backend/internal/database"
	"github.com/chachabrian/mooveit-backend/internal/handlers"
	"github.com/chachabrian/mooveit-backend/internal/middleware"
	"github.com/chachabrian/mooveit-backend/internal/services"
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

	// Initialize Redis
	if err := services.InitRedis(); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}

	// Initialize Firebase (optional - will log warning if not configured)
	if err := services.InitFirebase(); err != nil {
		log.Printf("Firebase initialization warning: %v", err)
	}

	// Initialize Storage (S3 or local fallback)
	if err := services.InitStorage(); err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize WebSocket hub
	hub := services.NewHub()
	go hub.Run()

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
			auth.POST("/verify-email", handlers.VerifyEmail(db))
			auth.POST("/forgot-password", handlers.RequestPasswordReset(db))
			auth.POST("/verify-otp", handlers.VerifyOTP(db))
			auth.POST("/reset-password", handlers.ResetPassword(db))
		}

		// WebSocket connection
		api.GET("/ws", middleware.AuthMiddleware(), handlers.WebSocketHandler(hub))

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

			// Driver location and availability routes
			driver := protected.Group("/driver")
			{
				driver.POST("/location", handlers.UpdateDriverLocation(db, hub))
				driver.POST("/availability", handlers.UpdateDriverAvailability(db))
				driver.GET("/status", handlers.GetDriverStatus(db))
				driver.GET("/assigned-rides", handlers.GetDriverAssignedRides(db))
				driver.POST("/rides/:rideId/accept", handlers.AcceptRide(db, hub))
				driver.POST("/rides/:rideId/reject", handlers.RejectRide(db, hub))
				driver.POST("/rides/:rideId/arrived", handlers.DriverArrived(db, hub))
				driver.POST("/rides/:rideId/start", handlers.StartRide(db, hub))
				driver.GET("/trip-history", handlers.GetDriverTripHistory(db))
			}

			// Rides routes
			rides := protected.Group("/rides")
			{
				rides.GET("", handlers.GetAvailableRides(db))
				rides.POST("", handlers.CreateRide(db))
				rides.GET("/driver", handlers.GetDriverRides(db))
				rides.GET("/all", handlers.GetAllRides(db))
				rides.DELETE("/:id", handlers.DeleteRide(db))
				rides.GET("/nearby-drivers", handlers.GetNearbyDrivers(db))
				rides.POST("/request", handlers.RequestRide(db, hub))
				rides.POST("/:rideId/cancel", handlers.CancelRide(db, hub))
				rides.GET("/:rideId/status", handlers.GetRideStatus(db))
				rides.PATCH("/:rideId/status", handlers.UpdateRideStatus(db, hub))
				rides.POST("/:rideId/complete", handlers.CompleteTrip(db, hub))
				rides.GET("/:rideId/completion", handlers.GetTripCompletion(db))
				rides.POST("/:rideId/rate", handlers.RateTrip(db))
				rides.GET("/trip-history", handlers.GetClientTripHistory(db))
			}

			// Pricing routes
			pricing := protected.Group("/pricing")
			{
				pricing.POST("/zones", handlers.CreatePricingZone(db))
				pricing.GET("/zones", handlers.GetPricingZones(db))
				pricing.POST("/driver", handlers.SetDriverPricing(db))
				pricing.GET("/driver", handlers.GetDriverPricing(db))
				pricing.GET("/calculate", handlers.CalculateFare(db))
				pricing.GET("/estimate", handlers.GetDynamicFareEstimate(db))
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

			// Notification routes
			notifications := protected.Group("/notifications")
			{
				notifications.POST("/register-token", handlers.RegisterFCMToken(db))
				notifications.DELETE("/remove-token", handlers.RemoveFCMToken(db))
				notifications.POST("/test", handlers.TestNotification(db))
				notifications.POST("/broadcast", handlers.SendBroadcastNotificationHandler(db))
				notifications.POST("/scheduled-rides-available", handlers.NotifyScheduledRidesAvailable(db))

				// Notification preferences
				notifications.GET("/preferences", handlers.GetNotificationPreferences(db))
				notifications.PUT("/preferences", handlers.UpdateNotificationPreferences(db))
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
