
package main

import (
    "log"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"
    "github.com/chachabrian/mooveit-backend/internal/database"
    "github.com/chachabrian/mooveit-backend/internal/handlers"
    "github.com/chachabrian/mooveit-backend/internal/middleware"
)

func main() {
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }

    // Initialize database
    db, err := database.InitDB()
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    // Initialize router
    r := gin.Default()

    // CORS middleware
    r.Use(func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

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
            }

            // Bookings routes
            bookings := protected.Group("/bookings")
            {
                bookings.POST("", handlers.CreateBooking(db))
                bookings.GET("/:id/status", handlers.GetBookingStatus(db))
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
