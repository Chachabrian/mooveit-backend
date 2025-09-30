package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/chachabrian/mooveit-backend/internal/services"
	"github.com/chachabrian/mooveit-backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequestRide handles ride requests from clients
func RequestRide(db *gorm.DB, hub *services.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeClient) {
			c.JSON(403, gin.H{"error": "Only clients can request rides"})
			return
		}

		var input struct {
			Pickup struct {
				Lat     float64 `json:"lat" binding:"required"`
				Lng     float64 `json:"lng" binding:"required"`
				Address string  `json:"address" binding:"required"`
			} `json:"pickup" binding:"required"`
			Destination struct {
				Lat     float64 `json:"lat" binding:"required"`
				Lng     float64 `json:"lng" binding:"required"`
				Address string  `json:"address" binding:"required"`
			} `json:"destination" binding:"required"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Validate coordinates
		if input.Pickup.Lat < -90 || input.Pickup.Lat > 90 ||
			input.Destination.Lat < -90 || input.Destination.Lat > 90 {
			c.JSON(400, gin.H{"error": "Invalid latitude"})
			return
		}
		if input.Pickup.Lng < -180 || input.Pickup.Lng > 180 ||
			input.Destination.Lng < -180 || input.Destination.Lng > 180 {
			c.JSON(400, gin.H{"error": "Invalid longitude"})
			return
		}

		// Calculate distance and estimated price
		distance := utils.HaversineDistance(
			input.Pickup.Lat, input.Pickup.Lng,
			input.Destination.Lat, input.Destination.Lng,
		)
		price := utils.CalculatePrice(distance, 2.0) // 2.0 per km
		duration := utils.CalculateETA(distance, 30) // 30 km/h average speed

		// Get client information first to avoid nil pointer issues
		var client models.User
		if err := db.Where("id = ?", clientID).First(&client).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to get client information"})
			return
		}

		// Create ride request
		rideRequest := models.RideRequest{
			ClientID:   clientID,
			PickupLat:  input.Pickup.Lat,
			PickupLng:  input.Pickup.Lng,
			PickupAddr: input.Pickup.Address,
			DestLat:    input.Destination.Lat,
			DestLng:    input.Destination.Lng,
			DestAddr:   input.Destination.Address,
			Status:     models.RideStatusPending,
			Price:      price,
			Distance:   distance,
			Duration:   duration,
		}

		if err := db.Create(&rideRequest).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to create ride request"})
			return
		}

		// Find nearby available drivers
		var locations []models.DriverLocation
		if err := db.Where("is_online = ? AND is_available = ?", true, true).Find(&locations).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to find available drivers"})
			return
		}

		// If no drivers found, return pending status
		if len(locations) == 0 {
			c.JSON(200, gin.H{
				"message": "Ride request created. No drivers available at the moment.",
				"rideId":  rideRequest.ID,
				"status":  rideRequest.Status,
			})
			return
		}

		// Send ride request notifications to nearby drivers
		notificationsSent := 0
		for _, location := range locations {
			driverDistance := utils.HaversineDistance(
				input.Pickup.Lat, input.Pickup.Lng,
				location.Latitude, location.Longitude,
			)

			// Only notify drivers within reasonable distance (e.g., 10km)
			if driverDistance <= 10.0 {
				// Create notification data
				notificationData := gin.H{
					"rideId":     rideRequest.ID,
					"clientId":   clientID,
					"clientName": client.Username,
					"pickup": gin.H{
						"lat":     input.Pickup.Lat,
						"lng":     input.Pickup.Lng,
						"address": input.Pickup.Address,
					},
					"destination": gin.H{
						"lat":     input.Destination.Lat,
						"lng":     input.Destination.Lng,
						"address": input.Destination.Address,
					},
					"price":         price,
					"distance":      driverDistance,
					"duration":      duration,
					"estimatedTime": utils.CalculateETA(driverDistance, 30),
				}

				// Create WebSocket message
				rideNotification := services.WebSocketMessage{
					Type: "ride_request",
					Data: notificationData,
				}

				// Marshal and send notification
				if notificationBytes, err := json.Marshal(rideNotification); err == nil {
					hub.BroadcastToUser(location.DriverID, notificationBytes)
					notificationsSent++
				}
			}
		}

		responseMessage := "Ride request created."
		if notificationsSent > 0 {
			responseMessage = "Ride request sent to nearby drivers. Waiting for acceptance."
		} else {
			responseMessage = "Ride request created. No drivers available at the moment."
		}

		c.JSON(200, gin.H{
			"message": responseMessage,
			"rideId":  rideRequest.ID,
			"status":  rideRequest.Status,
		})
	}
}

// CancelRide handles ride cancellations
func CancelRide(db *gorm.DB, hub *services.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		rideIDStr := c.Param("rideId")
		userID := c.GetUint("userId")
		userType := c.GetString("userType")

		rideID, err := strconv.ParseUint(rideIDStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ride ID"})
			return
		}

		var rideRequest models.RideRequest
		if err := db.First(&rideRequest, rideID).Error; err != nil {
			c.JSON(404, gin.H{"error": "Ride not found"})
			return
		}

		// Check if user is authorized to cancel this ride
		if userType == string(models.UserTypeClient) && rideRequest.ClientID != userID {
			c.JSON(403, gin.H{"error": "Unauthorized to cancel this ride"})
			return
		}
		if userType == string(models.UserTypeDriver) && (rideRequest.DriverID == nil || *rideRequest.DriverID != userID) {
			c.JSON(403, gin.H{"error": "Unauthorized to cancel this ride"})
			return
		}

		// Check if ride can be cancelled
		if rideRequest.Status == models.RideStatusCompleted || rideRequest.Status == models.RideStatusCancelled {
			c.JSON(400, gin.H{"error": "Ride cannot be cancelled"})
			return
		}

		// Update ride status
		rideRequest.Status = models.RideStatusCancelled
		if err := db.Save(&rideRequest).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to cancel ride"})
			return
		}

		// If driver was assigned, make them available again
		if rideRequest.DriverID != nil {
			var driverLocation models.DriverLocation
			if err := db.Where("driver_id = ?", *rideRequest.DriverID).First(&driverLocation).Error; err == nil {
				driverLocation.IsAvailable = true
				db.Save(&driverLocation)
			}
		}

		// Notify relevant parties
		if rideRequest.DriverID != nil {
			// Notify driver
			cancellationData := services.WebSocketMessage{
				Type: "ride_cancelled",
				Data: gin.H{
					"rideId": rideRequest.ID,
					"reason": "Cancelled by client",
				},
			}
			if data, err := json.Marshal(cancellationData); err == nil {
				hub.BroadcastToUser(*rideRequest.DriverID, data)
			}
		}

		// Notify client
		clientNotification := services.WebSocketMessage{
			Type: "ride_cancelled",
			Data: gin.H{
				"rideId": rideRequest.ID,
				"reason": "Ride cancelled successfully",
			},
		}
		if data, err := json.Marshal(clientNotification); err == nil {
			hub.BroadcastToUser(rideRequest.ClientID, data)
		}

		c.JSON(200, gin.H{
			"message": "Ride cancelled successfully",
			"rideId":  rideRequest.ID,
			"status":  rideRequest.Status,
		})
	}
}

// GetRideStatus handles getting ride status
func GetRideStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rideIDStr := c.Param("rideId")
		userID := c.GetUint("userId")
		userType := c.GetString("userType")

		rideID, err := strconv.ParseUint(rideIDStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ride ID"})
			return
		}

		var rideRequest models.RideRequest
		if err := db.Preload("Client").Preload("Driver").First(&rideRequest, rideID).Error; err != nil {
			c.JSON(404, gin.H{"error": "Ride not found"})
			return
		}

		// Check if user is authorized to view this ride
		if userType == string(models.UserTypeClient) && rideRequest.ClientID != userID {
			c.JSON(403, gin.H{"error": "Unauthorized to view this ride"})
			return
		}
		if userType == string(models.UserTypeDriver) && (rideRequest.DriverID == nil || *rideRequest.DriverID != userID) {
			c.JSON(403, gin.H{"error": "Unauthorized to view this ride"})
			return
		}

		// Prepare response
		response := gin.H{
			"rideId": rideRequest.ID,
			"status": rideRequest.Status,
			"pickup": gin.H{
				"lat":     rideRequest.PickupLat,
				"lng":     rideRequest.PickupLng,
				"address": rideRequest.PickupAddr,
			},
			"destination": gin.H{
				"lat":     rideRequest.DestLat,
				"lng":     rideRequest.DestLng,
				"address": rideRequest.DestAddr,
			},
			"price":    rideRequest.Price,
			"distance": rideRequest.Distance,
			"duration": rideRequest.Duration,
		}

		// Add client info for drivers
		if userType == string(models.UserTypeDriver) && rideRequest.Client != nil {
			response["client"] = gin.H{
				"id":       rideRequest.Client.ID,
				"username": rideRequest.Client.Username,
				"phone":    rideRequest.Client.PhoneNumber,
			}
		}

		// Add driver info for clients
		if userType == string(models.UserTypeClient) && rideRequest.Driver != nil {
			response["driver"] = gin.H{
				"id":       rideRequest.Driver.ID,
				"username": rideRequest.Driver.Username,
				"phone":    rideRequest.Driver.PhoneNumber,
			}
		}

		c.JSON(200, response)
	}
}

// UpdateRideStatus handles updating ride status
func UpdateRideStatus(db *gorm.DB, hub *services.Hub) gin.HandlerFunc {
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
			Status string `json:"status" binding:"required"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var rideRequest models.RideRequest
		if err := db.First(&rideRequest, rideID).Error; err != nil {
			c.JSON(404, gin.H{"error": "Ride not found"})
			return
		}

		// Check if user is authorized to update this ride
		if userType == string(models.UserTypeClient) && rideRequest.ClientID != userID {
			c.JSON(403, gin.H{"error": "Unauthorized to update this ride"})
			return
		}
		if userType == string(models.UserTypeDriver) && (rideRequest.DriverID == nil || *rideRequest.DriverID != userID) {
			c.JSON(403, gin.H{"error": "Unauthorized to update this ride"})
			return
		}

		// Update ride status
		rideRequest.Status = input.Status
		if err := db.Save(&rideRequest).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to update ride status"})
			return
		}

		// Notify relevant parties
		statusUpdate := services.WebSocketMessage{
			Type: "ride_status_update",
			Data: gin.H{
				"rideId": rideRequest.ID,
				"status": rideRequest.Status,
			},
		}
		if data, err := json.Marshal(statusUpdate); err == nil {
			// Notify client
			hub.BroadcastToUser(rideRequest.ClientID, data)
			// Notify driver if assigned
			if rideRequest.DriverID != nil {
				hub.BroadcastToUser(*rideRequest.DriverID, data)
			}
		}

		c.JSON(200, gin.H{
			"message": "Ride status updated successfully",
			"rideId":  rideRequest.ID,
			"status":  rideRequest.Status,
		})
	}
}
