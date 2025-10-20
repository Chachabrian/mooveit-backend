package handlers

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/chachabrian/mooveit-backend/internal/services"
	"github.com/chachabrian/mooveit-backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AcceptRide allows driver to accept a ride request
func AcceptRide(db *gorm.DB, hub *services.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		rideIDStr := c.Param("rideId")
		driverID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can accept rides"})
			return
		}

		rideID, err := strconv.ParseUint(rideIDStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ride ID"})
			return
		}

		var rideRequest models.RideRequest
		if err := db.Preload("Client").First(&rideRequest, rideID).Error; err != nil {
			c.JSON(404, gin.H{"error": "Ride not found"})
			return
		}

		// Check if ride is still pending
		if rideRequest.Status != models.RideStatusPending {
			c.JSON(400, gin.H{"error": "Ride is no longer available"})
			return
		}

		// Check if driver is available
		var driverLocation models.DriverLocation
		if err := db.Where("driver_id = ?", driverID).First(&driverLocation).Error; err != nil {
			c.JSON(400, gin.H{"error": "Driver location not found"})
			return
		}

		if !driverLocation.IsAvailable {
			c.JSON(400, gin.H{"error": "Driver is not available"})
			return
		}

		// Start transaction
		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// Assign driver to ride
		rideRequest.DriverID = &driverID
		rideRequest.Status = models.RideStatusAccepted
		if err := tx.Save(&rideRequest).Error; err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Failed to accept ride"})
			return
		}

		// Make driver unavailable
		driverLocation.IsAvailable = false
		if err := tx.Save(&driverLocation).Error; err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Failed to update driver availability"})
			return
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to complete transaction"})
			return
		}

		// Update Redis
		ctx := context.Background()
		services.SetDriverAvailability(ctx, driverID, false)

		// Calculate ETA for driver to reach pickup location
		distance := utils.HaversineDistance(
			driverLocation.Latitude, driverLocation.Longitude,
			rideRequest.PickupLat, rideRequest.PickupLng,
		)
		eta := utils.CalculateETA(distance, 30) // Assuming 30 km/h average speed

		// Get client information for notifications
		var client models.User
		if err := db.Where("id = ?", rideRequest.ClientID).First(&client).Error; err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Failed to get client information"})
			return
		}

		// Notify client via WebSocket
		accepted := services.RideAccepted{
			RideID:        rideRequest.ID,
			DriverID:      driverID,
			EstimatedTime: eta,
		}
		hub.SendRideAccepted(rideRequest.ClientID, accepted)

		// Send FCM push notification to client
		ctx = context.Background()
		if client.FCMToken != "" {
			var driver models.User
			if err := db.First(&driver, driverID).Error; err == nil {
				vehicleDetails := driver.CarMake + " " + driver.CarColor + " - " + driver.CarPlate
				go services.SendRideAcceptedNotification(
					ctx,
					client.FCMToken,
					rideRequest.ID,
					driver.Username,
					vehicleDetails,
					eta,
				)
			}
		}

		// Notify driver with pickup details
		driverNotification := services.WebSocketMessage{
			Type: "ride_accepted",
			Data: gin.H{
				"rideId":     rideRequest.ID,
				"clientId":   rideRequest.ClientID,
				"clientName": client.Username,
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
				"eta":      eta,
			},
		}

		notificationData, _ := json.Marshal(driverNotification)
		hub.BroadcastToUser(driverID, notificationData)

		c.JSON(200, gin.H{
			"message": "Ride accepted successfully",
			"rideId":  rideRequest.ID,
			"status":  rideRequest.Status,
			"eta":     eta,
		})
	}
}

// RejectRide allows driver to reject a ride request
func RejectRide(db *gorm.DB, hub *services.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		rideIDStr := c.Param("rideId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can reject rides"})
			return
		}

		rideID, err := strconv.ParseUint(rideIDStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ride ID"})
			return
		}

		var rideRequest models.RideRequest
		if err := db.Preload("Client").First(&rideRequest, rideID).Error; err != nil {
			c.JSON(404, gin.H{"error": "Ride not found"})
			return
		}

		// Check if ride is still pending
		if rideRequest.Status != models.RideStatusPending {
			c.JSON(400, gin.H{"error": "Ride is no longer available"})
			return
		}

		// Update ride status to cancelled
		rideRequest.Status = models.RideStatusCancelled
		if err := db.Save(&rideRequest).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to reject ride"})
			return
		}

		// Notify client via WebSocket
		ctx := context.Background()
		services.PublishRideUpdate(ctx, uint(rideID), "rejected", gin.H{
			"reason": "Driver rejected the ride",
		})

		c.JSON(200, gin.H{
			"message": "Ride rejected successfully",
			"rideId":  rideRequest.ID,
			"status":  rideRequest.Status,
		})
	}
}

// GetDriverAssignedRides gets rides assigned to a driver
func GetDriverAssignedRides(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can view assigned rides"})
			return
		}

		var rides []models.RideRequest
		if err := db.Preload("Client").
			Where("driver_id = ? AND status IN (?)", driverID, []string{
				models.RideStatusAccepted,
				models.RideStatusStarted,
			}).
			Order("created_at DESC").
			Find(&rides).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch assigned rides"})
			return
		}

		c.JSON(200, rides)
	}
}

// DriverArrived allows driver to mark that they have arrived at pickup location
func DriverArrived(db *gorm.DB, hub *services.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		rideIDStr := c.Param("rideId")
		driverID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can mark arrival"})
			return
		}

		rideID, err := strconv.ParseUint(rideIDStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ride ID"})
			return
		}

		var rideRequest models.RideRequest
		if err := db.Preload("Client").First(&rideRequest, rideID).Error; err != nil {
			c.JSON(404, gin.H{"error": "Ride not found"})
			return
		}

		// Check if driver is assigned to this ride
		if rideRequest.DriverID == nil || *rideRequest.DriverID != driverID {
			c.JSON(403, gin.H{"error": "Unauthorized to update this ride"})
			return
		}

		// Check if ride is in accepted status
		if rideRequest.Status != models.RideStatusAccepted {
			c.JSON(400, gin.H{"error": "Ride must be accepted before marking arrival"})
			return
		}

		// Update ride status to arrived
		rideRequest.Status = models.RideStatusArrived
		if err := db.Save(&rideRequest).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to update ride status"})
			return
		}

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

		// Notify client that driver has arrived
		arrived := services.DriverArrived{
			RideID:   rideRequest.ID,
			DriverID: driverID,
		}
		hub.SendDriverArrived(rideRequest.ClientID, arrived)

		// Send FCM push notification to client
		ctx := context.Background()
		if client.FCMToken != "" {
			go services.SendDriverArrivedNotification(
				ctx,
				client.FCMToken,
				rideRequest.ID,
				driver.Username,
			)
		}

		// Also send a general status update notification
		statusUpdate := services.WebSocketMessage{
			Type: "driver_arrived",
			Data: gin.H{
				"rideId":     rideRequest.ID,
				"driverId":   driverID,
				"driverName": driver.Username,
				"status":     rideRequest.Status,
				"message":    "Driver has arrived at pickup location",
			},
		}

		notificationData, _ := json.Marshal(statusUpdate)
		hub.BroadcastToUser(rideRequest.ClientID, notificationData)

		// Notify driver
		driverNotification := services.WebSocketMessage{
			Type: "arrival_confirmed",
			Data: gin.H{
				"rideId":     rideRequest.ID,
				"clientId":   rideRequest.ClientID,
				"clientName": client.Username,
				"status":     rideRequest.Status,
				"message":    "You have arrived at pickup location",
			},
		}

		driverData, _ := json.Marshal(driverNotification)
		hub.BroadcastToUser(driverID, driverData)

		c.JSON(200, gin.H{
			"message": "Driver arrival confirmed successfully",
			"rideId":  rideRequest.ID,
			"status":  rideRequest.Status,
		})
	}
}

// StartRide allows driver to start a ride (arrived at pickup)
func StartRide(db *gorm.DB, hub *services.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		rideIDStr := c.Param("rideId")
		driverID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can start rides"})
			return
		}

		rideID, err := strconv.ParseUint(rideIDStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ride ID"})
			return
		}

		var rideRequest models.RideRequest
		if err := db.Preload("Client").First(&rideRequest, rideID).Error; err != nil {
			c.JSON(404, gin.H{"error": "Ride not found"})
			return
		}

		// Check if driver is assigned to this ride
		if rideRequest.DriverID == nil || *rideRequest.DriverID != driverID {
			c.JSON(403, gin.H{"error": "Unauthorized to start this ride"})
			return
		}

		// Check if ride is in correct status (accepted or arrived)
		if rideRequest.Status != models.RideStatusAccepted && rideRequest.Status != models.RideStatusArrived {
			c.JSON(400, gin.H{"error": "Ride must be accepted before starting"})
			return
		}

		// Update ride status to started
		rideRequest.Status = models.RideStatusStarted
		if err := db.Save(&rideRequest).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to start ride"})
			return
		}

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

		// Notify client that ride has started
		started := services.RideStarted{
			RideID:   rideRequest.ID,
			DriverID: driverID,
		}
		hub.SendRideStarted(rideRequest.ClientID, started)

		// Send FCM push notification to client
		ctx := context.Background()
		if client.FCMToken != "" {
			go services.SendRideStartedNotification(
				ctx,
				client.FCMToken,
				rideRequest.ID,
				driver.Username,
			)
		}

		// Also send a general status update notification
		statusUpdate := services.WebSocketMessage{
			Type: "ride_started",
			Data: gin.H{
				"rideId":     rideRequest.ID,
				"driverId":   driverID,
				"driverName": driver.Username,
				"status":     rideRequest.Status,
				"message":    "Ride has started - proceeding to destination",
			},
		}

		notificationData, _ := json.Marshal(statusUpdate)
		hub.BroadcastToUser(rideRequest.ClientID, notificationData)

		// Notify driver
		driverNotification := services.WebSocketMessage{
			Type: "ride_started",
			Data: gin.H{
				"rideId":     rideRequest.ID,
				"clientId":   rideRequest.ClientID,
				"clientName": client.Username,
				"status":     rideRequest.Status,
				"message":    "Ride started - proceed to destination",
			},
		}

		driverData, _ := json.Marshal(driverNotification)
		hub.BroadcastToUser(driverID, driverData)

		c.JSON(200, gin.H{
			"message": "Ride started successfully",
			"rideId":  rideRequest.ID,
			"status":  rideRequest.Status,
		})
	}
}
