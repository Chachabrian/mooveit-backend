package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/chachabrian/mooveit-backend/internal/services"
	"github.com/chachabrian/mooveit-backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UpdateDriverLocation handles driver location updates
func UpdateDriverLocation(db *gorm.DB, hub *services.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can update location"})
			return
		}

		var input struct {
			Lat     float64 `json:"lat" binding:"required"`
			Lng     float64 `json:"lng" binding:"required"`
			Heading float64 `json:"heading" binding:"required"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Validate coordinates
		if input.Lat < -90 || input.Lat > 90 {
			c.JSON(400, gin.H{"error": "Invalid latitude"})
			return
		}
		if input.Lng < -180 || input.Lng > 180 {
			c.JSON(400, gin.H{"error": "Invalid longitude"})
			return
		}

		ctx := context.Background()

		// Update location in Redis
		if err := services.SetDriverLocation(ctx, driverID, input.Lat, input.Lng, input.Heading); err != nil {
			c.JSON(500, gin.H{"error": "Failed to update location"})
			return
		}

		// Update or create location record in database
		var location models.DriverLocation
		result := db.Where("driver_id = ?", driverID).First(&location)

		if result.Error == gorm.ErrRecordNotFound {
			// Create new location record
			location = models.DriverLocation{
				DriverID:  driverID,
				Latitude:  input.Lat,
				Longitude: input.Lng,
				Heading:   input.Heading,
				IsOnline:  true,
				LastSeen:  time.Now(),
			}
			if err := db.Create(&location).Error; err != nil {
				c.JSON(500, gin.H{"error": "Failed to create location record"})
				return
			}
		} else if result.Error != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch location record"})
			return
		} else {
			// Update existing location record
			location.Latitude = input.Lat
			location.Longitude = input.Lng
			location.Heading = input.Heading
			location.IsOnline = true
			location.LastSeen = time.Now()
			if err := db.Save(&location).Error; err != nil {
				c.JSON(500, gin.H{"error": "Failed to update location record"})
				return
			}
		}

		// Publish location update to WebSocket clients
		update := services.DriverLocationUpdate{
			DriverID: driverID,
		}
		update.Location.Lat = input.Lat
		update.Location.Lng = input.Lng
		update.Location.Heading = input.Heading

		hub.SendDriverLocationUpdate(update)

		c.JSON(200, gin.H{
			"message": "Location updated successfully",
			"location": gin.H{
				"lat":     input.Lat,
				"lng":     input.Lng,
				"heading": input.Heading,
			},
		})
	}
}

// UpdateDriverAvailability handles driver availability updates
func UpdateDriverAvailability(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can update availability"})
			return
		}

		var input struct {
			IsAvailable *bool `json:"isAvailable"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		if input.IsAvailable == nil {
			c.JSON(400, gin.H{"error": "isAvailable field is required"})
			return
		}

		ctx := context.Background()

		// Update availability in Redis
		if err := services.SetDriverAvailability(ctx, driverID, *input.IsAvailable); err != nil {
			c.JSON(500, gin.H{"error": "Failed to update availability"})
			return
		}

		// Update availability in database
		var location models.DriverLocation
		if err := db.Where("driver_id = ?", driverID).First(&location).Error; err != nil {
			c.JSON(404, gin.H{"error": "Driver location not found"})
			return
		}

		location.IsAvailable = *input.IsAvailable
		location.LastSeen = time.Now()

		if err := db.Save(&location).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to update availability"})
			return
		}

		c.JSON(200, gin.H{
			"message":     "Availability updated successfully",
			"isAvailable": *input.IsAvailable,
		})
	}
}

// GetNearbyDrivers finds drivers within a specified radius
func GetNearbyDrivers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		latStr := c.Query("lat")
		lngStr := c.Query("lng")
		radiusStr := c.DefaultQuery("radius", "10") // Default 10km radius

		if latStr == "" || lngStr == "" {
			c.JSON(400, gin.H{"error": "Latitude and longitude are required"})
			return
		}

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid latitude"})
			return
		}

		lng, err := strconv.ParseFloat(lngStr, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid longitude"})
			return
		}

		radius, err := strconv.ParseFloat(radiusStr, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid radius"})
			return
		}

		// Validate coordinates
		if lat < -90 || lat > 90 {
			c.JSON(400, gin.H{"error": "Invalid latitude"})
			return
		}
		if lng < -180 || lng > 180 {
			c.JSON(400, gin.H{"error": "Invalid longitude"})
			return
		}

		// Get all available drivers
		var locations []models.DriverLocation
		if err := db.Preload("Driver").
			Where("is_online = ? AND is_available = ?", true, true).
			Find(&locations).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch drivers"})
			return
		}

		var nearbyDrivers []gin.H
		ctx := context.Background()

		for _, location := range locations {
			// Calculate distance
			distance := utils.HaversineDistance(lat, lng, location.Latitude, location.Longitude)

			// Check if driver is within radius
			if distance <= radius {
				// Calculate ETA (estimated time of arrival)
				eta := utils.CalculateETA(distance, 30) // Assuming 30 km/h average speed

				// Calculate estimated price
				price := utils.CalculatePrice(distance, 2.0) // Assuming 2.0 per km

				driver := gin.H{
					"id":     location.DriverID,
					"name":   location.Driver.Username,
					"rating": 4.5, // Default rating, would be calculated from actual ratings
					"location": gin.H{
						"lat":     location.Latitude,
						"lng":     location.Longitude,
						"heading": location.Heading,
					},
					"vehicle": gin.H{
						"make":  location.Driver.CarMake,
						"color": location.Driver.CarColor,
						"plate": location.Driver.CarPlate,
					},
					"isAvailable":    location.IsAvailable,
					"estimatedTime":  eta,
					"estimatedPrice": price,
					"distance":       distance,
				}

				nearbyDrivers = append(nearbyDrivers, driver)
			}
		}

		// Cache nearby drivers in Redis
		if len(nearbyDrivers) > 0 {
			// Convert gin.H to map[string]interface{}
			drivers := make([]map[string]interface{}, len(nearbyDrivers))
			for i, driver := range nearbyDrivers {
				drivers[i] = map[string]interface{}(driver)
			}
			services.SetNearbyDrivers(ctx, lat, lng, drivers)
		}

		c.JSON(200, gin.H{
			"drivers": nearbyDrivers,
			"count":   len(nearbyDrivers),
		})
	}
}

// GetDriverStatus gets the current status of a driver
func GetDriverStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can check status"})
			return
		}

		var location models.DriverLocation
		if err := db.Where("driver_id = ?", driverID).First(&location).Error; err != nil {
			c.JSON(404, gin.H{"error": "Driver location not found"})
			return
		}

		ctx := context.Background()
		lat, lng, heading, err := services.GetDriverLocation(ctx, driverID)
		if err != nil {
			// Fallback to database values
			lat = location.Latitude
			lng = location.Longitude
			heading = location.Heading
		}

		isAvailable, err := services.GetDriverAvailability(ctx, driverID)
		if err != nil {
			// Fallback to database values
			isAvailable = location.IsAvailable
		}

		status := "offline"
		if location.IsOnline {
			if isAvailable {
				status = "available"
			} else {
				status = "busy"
			}
		}

		c.JSON(200, gin.H{
			"driverId":    driverID,
			"status":      status,
			"isOnline":    location.IsOnline,
			"isAvailable": isAvailable,
			"location": gin.H{
				"lat":     lat,
				"lng":     lng,
				"heading": heading,
			},
			"lastSeen": location.LastSeen,
		})
	}
}
