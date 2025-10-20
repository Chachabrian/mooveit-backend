package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/chachabrian/mooveit-backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateRide handles the creation of a new ride by a driver
func CreateRide(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can create rides"})
			return
		}

		var input struct {
			CurrentLocation string    `json:"currentLocation" binding:"required"`
			Destination     string    `json:"destination" binding:"required"`
			TruckSize       string    `json:"truckSize" binding:"required"`
			Price           float64   `json:"price" binding:"required"`
			Date            time.Time `json:"date" binding:"required"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Check if the scheduled time is in the future
		if input.Date.Before(time.Now()) {
			c.JSON(400, gin.H{"error": "Ride date must be in the future"})
			return
		}

		ride := models.Ride{
			DriverID:        userId,
			CurrentLocation: input.CurrentLocation,
			Destination:     input.Destination,
			TruckSize:       input.TruckSize,
			Price:           input.Price,
			Date:            input.Date,
			Status:          "available",
		}

		if err := db.Create(&ride).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to create ride"})
			return
		}

		// Send push notification to clients who opted-in for available rides notifications
		go func() {
			ctx := context.Background()
			payload := services.NotificationPayload{
				Title: "New Ride Available! 🚛",
				Body: fmt.Sprintf("From %s to %s - KES %.2f",
					input.CurrentLocation, input.Destination, input.Price),
				Data: map[string]interface{}{
					"type":            "new_ride_available",
					"rideId":          fmt.Sprintf("%d", ride.ID),
					"currentLocation": input.CurrentLocation,
					"destination":     input.Destination,
					"price":           fmt.Sprintf("%.2f", input.Price),
					"date":            input.Date.Format(time.RFC3339),
					"truckSize":       input.TruckSize,
				},
				ChannelID: "mooveit_rides",
				Priority:  "high",
			}
			// Send to clients who subscribed to available rides topic
			if err := services.SendTopicNotification(ctx, "clients-available-rides", payload); err != nil {
				log.Printf("Failed to send new ride notification to subscribed clients: %v", err)
			}
		}()

		c.JSON(201, ride)
	}
}

// GetAvailableRides retrieves all available rides with optional search
func GetAvailableRides(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		destination := c.Query("destination")
		currentLocation := c.Query("currentLocation")

		var rides []models.Ride
		query := db.Preload("Driver").
			Where("rides.date > ? AND rides.date <= ? AND rides.status = ?",
				time.Now(),
				time.Now().Add(24*time.Hour),
				"available")

		if destination != "" {
			query = query.Where("LOWER(rides.destination) LIKE LOWER(?)", "%"+strings.ToLower(destination)+"%")
		}
		if currentLocation != "" {
			query = query.Where("LOWER(rides.current_location) LIKE LOWER(?)", "%"+strings.ToLower(currentLocation)+"%")
		}

		if err := query.Find(&rides).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch rides"})
			return
		}

		c.JSON(200, rides)
	}
}

// GetDriverRides retrieves all rides created by a specific driver
func GetDriverRides(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetUint("userId")

		var rides []models.Ride
		if err := db.Where("driver_id = ?", userId).
			Order("date DESC").
			Find(&rides).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch driver rides"})
			return
		}

		c.JSON(200, rides)
	}
}

// GetAllRides retrieves all rides in the system
func GetAllRides(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rides []models.Ride
		if err := db.Preload("Driver").
			Where("date > ? AND date <= ?",
				time.Now(),
				time.Now().Add(24*time.Hour)).
			Order("date ASC").
			Find(&rides).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch rides"})
			return
		}

		c.JSON(200, rides)
	}
}

// DeleteRide soft deletes a ride
func DeleteRide(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rideID := c.Param("id")
		userId := c.GetUint("userId")

		var ride models.Ride
		if err := db.First(&ride, rideID).Error; err != nil {
			c.JSON(404, gin.H{"error": "Ride not found"})
			return
		}

		// Check if the user is the owner of the ride
		if ride.DriverID != userId {
			c.JSON(403, gin.H{"error": "Unauthorized to delete this ride"})
			return
		}

		// Soft delete the ride
		if err := db.Delete(&ride).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to delete ride"})
			return
		}

		c.JSON(200, gin.H{"message": "Ride successfully deleted"})
	}
}
