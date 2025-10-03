package handlers

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/chachabrian/mooveit-backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CompleteTrip handles trip completion by driver
func CompleteTrip(db *gorm.DB, hub *services.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		rideIDStr := c.Param("rideId")
		driverID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can complete trips"})
			return
		}

		rideID, err := strconv.ParseUint(rideIDStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ride ID"})
			return
		}

		var input struct {
			ActualFare     float64 `json:"actualFare" binding:"required"`
			ActualDistance float64 `json:"actualDistance" binding:"required"`
			ActualDuration int     `json:"actualDuration" binding:"required"`
			DriverNotes    string  `json:"driverNotes,omitempty"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Validate input
		if input.ActualFare < 0 {
			c.JSON(400, gin.H{"error": "Actual fare must be non-negative"})
			return
		}
		if input.ActualDistance < 0 {
			c.JSON(400, gin.H{"error": "Actual distance must be non-negative"})
			return
		}
		if input.ActualDuration < 0 {
			c.JSON(400, gin.H{"error": "Actual duration must be non-negative"})
			return
		}

		// Get ride request
		var rideRequest models.RideRequest
		if err := db.Preload("Client").First(&rideRequest, rideID).Error; err != nil {
			c.JSON(404, gin.H{"error": "Ride not found"})
			return
		}

		// Check if driver is assigned to this ride
		if rideRequest.DriverID == nil || *rideRequest.DriverID != driverID {
			c.JSON(403, gin.H{"error": "Unauthorized to complete this ride"})
			return
		}

		// Check if ride is in correct status
		if rideRequest.Status != models.RideStatusStarted {
			c.JSON(400, gin.H{"error": "Ride must be started before completion"})
			return
		}

		// Check if trip completion already exists
		var existingCompletion models.TripCompletion
		if err := db.Where("ride_id = ?", rideID).First(&existingCompletion).Error; err == nil {
			c.JSON(400, gin.H{"error": "Trip already completed"})
			return
		}

		// Create trip completion record
		tripCompletion := models.TripCompletion{
			RideID:         uint(rideID),
			DriverID:       driverID,
			ClientID:       rideRequest.ClientID,
			ActualFare:     input.ActualFare,
			ActualDistance: input.ActualDistance,
			ActualDuration: input.ActualDuration,
			DriverNotes:    input.DriverNotes,
		}

		// Start transaction
		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// Create trip completion
		if err := tx.Create(&tripCompletion).Error; err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Failed to create trip completion"})
			return
		}

		// Update ride status to completed
		rideRequest.Status = models.RideStatusCompleted
		if err := tx.Save(&rideRequest).Error; err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Failed to update ride status"})
			return
		}

		// Make driver available again
		var driverLocation models.DriverLocation
		if err := tx.Where("driver_id = ?", driverID).First(&driverLocation).Error; err == nil {
			driverLocation.IsAvailable = true
			if err := tx.Save(&driverLocation).Error; err != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"error": "Failed to update driver availability"})
				return
			}
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to complete transaction"})
			return
		}

		// Update Redis
		ctx := context.Background()
		services.SetDriverAvailability(ctx, driverID, true)

		// Get client and driver information for notifications
		var client models.User
		if err := db.Where("id = ?", rideRequest.ClientID).First(&client).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to get client information"})
			return
		}

		var driver models.User
		if err := db.Where("id = ?", driverID).First(&driver).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to get driver information"})
			return
		}

		// Notify client that ride has completed
		completed := services.RideCompleted{
			RideID:         uint(rideID),
			DriverID:       driverID,
			ActualFare:     input.ActualFare,
			ActualDistance: input.ActualDistance,
			ActualDuration: input.ActualDuration,
		}
		hub.SendRideCompleted(rideRequest.ClientID, completed)

		// Also send a general status update notification
		statusUpdate := services.WebSocketMessage{
			Type: "ride_completed",
			Data: gin.H{
				"rideId":         uint(rideID),
				"driverId":       driverID,
				"driverName":     driver.Username,
				"status":         rideRequest.Status,
				"actualFare":     input.ActualFare,
				"actualDistance": input.ActualDistance,
				"actualDuration": input.ActualDuration,
				"driverNotes":    input.DriverNotes,
				"message":        "Ride completed successfully",
			},
		}

		notificationData, _ := json.Marshal(statusUpdate)
		hub.BroadcastToUser(rideRequest.ClientID, notificationData)

		// Notify driver
		driverNotification := services.WebSocketMessage{
			Type: "ride_completed",
			Data: gin.H{
				"rideId":         uint(rideID),
				"clientId":       rideRequest.ClientID,
				"clientName":     client.Username,
				"status":         rideRequest.Status,
				"actualFare":     input.ActualFare,
				"actualDistance": input.ActualDistance,
				"actualDuration": input.ActualDuration,
				"message":        "Ride completed - you are now available for new rides",
			},
		}

		driverData, _ := json.Marshal(driverNotification)
		hub.BroadcastToUser(driverID, driverData)

		c.JSON(200, gin.H{
			"message":    "Trip completed successfully",
			"rideId":     rideID,
			"status":     rideRequest.Status,
			"completion": tripCompletion,
		})
	}
}

// GetTripCompletion gets trip completion details
func GetTripCompletion(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rideIDStr := c.Param("rideId")
		userID := c.GetUint("userId")

		rideID, err := strconv.ParseUint(rideIDStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ride ID"})
			return
		}

		var completion models.TripCompletion
		if err := db.Preload("Driver").Preload("Client").Preload("Ride").
			Where("ride_id = ?", rideID).First(&completion).Error; err != nil {
			c.JSON(404, gin.H{"error": "Trip completion not found"})
			return
		}

		// Check if user is authorized to view this completion
		if completion.DriverID != userID && completion.ClientID != userID {
			c.JSON(403, gin.H{"error": "Unauthorized to view this trip completion"})
			return
		}

		c.JSON(200, completion)
	}
}

// RateTrip allows rating after trip completion
func RateTrip(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rideIDStr := c.Param("rideId")
		userID := c.GetUint("userId")
		userType := c.GetString("userType")

		rideID, err := strconv.ParseUint(rideIDStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ride ID"})
			return
		}

		var input struct {
			Rating float64 `json:"rating" binding:"required"`
			Notes  string  `json:"notes,omitempty"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Validate rating
		if input.Rating < 1 || input.Rating > 5 {
			c.JSON(400, gin.H{"error": "Rating must be between 1 and 5"})
			return
		}

		// Get trip completion
		var completion models.TripCompletion
		if err := db.Where("ride_id = ?", rideID).First(&completion).Error; err != nil {
			c.JSON(404, gin.H{"error": "Trip completion not found"})
			return
		}

		// Check if user is authorized to rate
		if completion.DriverID != userID && completion.ClientID != userID {
			c.JSON(403, gin.H{"error": "Unauthorized to rate this trip"})
			return
		}

		// Update rating based on user type
		if userType == string(models.UserTypeClient) {
			// Client rating driver
			completion.ClientRating = &input.Rating
			completion.ClientNotes = input.Notes
		} else if userType == string(models.UserTypeDriver) {
			// Driver rating client
			completion.DriverRating = &input.Rating
			completion.DriverNotes = input.Notes
		}

		if err := db.Save(&completion).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to update rating"})
			return
		}

		c.JSON(200, gin.H{
			"message": "Rating updated successfully",
			"rideId":  rideID,
			"rating":  input.Rating,
		})
	}
}

// GetDriverTripHistory gets driver's trip history
func GetDriverTripHistory(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can view trip history"})
			return
		}

		pageStr := c.DefaultQuery("page", "1")
		limitStr := c.DefaultQuery("limit", "10")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			limit = 10
		}

		offset := (page - 1) * limit

		var completions []models.TripCompletion
		if err := db.Preload("Client").Preload("Ride").
			Where("driver_id = ?", driverID).
			Order("created_at DESC").
			Offset(offset).
			Limit(limit).
			Find(&completions).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch trip history"})
			return
		}

		var total int64
		db.Model(&models.TripCompletion{}).Where("driver_id = ?", driverID).Count(&total)

		c.JSON(200, gin.H{
			"completions": completions,
			"pagination": gin.H{
				"page":       page,
				"limit":      limit,
				"total":      total,
				"totalPages": (total + int64(limit) - 1) / int64(limit),
			},
		})
	}
}

// GetClientTripHistory gets client's trip history
func GetClientTripHistory(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeClient) {
			c.JSON(403, gin.H{"error": "Only clients can view trip history"})
			return
		}

		pageStr := c.DefaultQuery("page", "1")
		limitStr := c.DefaultQuery("limit", "10")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			limit = 10
		}

		offset := (page - 1) * limit

		var completions []models.TripCompletion
		if err := db.Preload("Driver").Preload("Ride").
			Where("client_id = ?", clientID).
			Order("created_at DESC").
			Offset(offset).
			Limit(limit).
			Find(&completions).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch trip history"})
			return
		}

		var total int64
		db.Model(&models.TripCompletion{}).Where("client_id = ?", clientID).Count(&total)

		c.JSON(200, gin.H{
			"completions": completions,
			"pagination": gin.H{
				"page":       page,
				"limit":      limit,
				"total":      total,
				"totalPages": (total + int64(limit) - 1) / int64(limit),
			},
		})
	}
}
