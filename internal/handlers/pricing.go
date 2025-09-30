package handlers

import (
	"strconv"

	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/chachabrian/mooveit-backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreatePricingZone creates a new pricing zone
func CreatePricingZone(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userType := c.GetString("userType")

		// Only drivers can create pricing zones
		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can create pricing zones"})
			return
		}

		var input struct {
			Name        string  `json:"name" binding:"required"`
			CenterLat   float64 `json:"centerLat" binding:"required"`
			CenterLng   float64 `json:"centerLng" binding:"required"`
			Radius      float64 `json:"radius" binding:"required"`
			BaseFare    float64 `json:"baseFare" binding:"required"`
			PerKmRate   float64 `json:"perKmRate" binding:"required"`
			PerMinRate  float64 `json:"perMinRate" binding:"required"`
			MinFare     float64 `json:"minFare" binding:"required"`
			MaxFare     float64 `json:"maxFare" binding:"required"`
			Description string  `json:"description"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Validate coordinates
		if input.CenterLat < -90 || input.CenterLat > 90 {
			c.JSON(400, gin.H{"error": "Invalid center latitude"})
			return
		}
		if input.CenterLng < -180 || input.CenterLng > 180 {
			c.JSON(400, gin.H{"error": "Invalid center longitude"})
			return
		}

		// Validate pricing
		if input.BaseFare < 0 || input.PerKmRate < 0 || input.PerMinRate < 0 {
			c.JSON(400, gin.H{"error": "Pricing values must be non-negative"})
			return
		}
		if input.MinFare > input.MaxFare {
			c.JSON(400, gin.H{"error": "Minimum fare cannot be greater than maximum fare"})
			return
		}

		pricingZone := models.PricingZone{
			Name:        input.Name,
			CenterLat:   input.CenterLat,
			CenterLng:   input.CenterLng,
			Radius:      input.Radius,
			BaseFare:    input.BaseFare,
			PerKmRate:   input.PerKmRate,
			PerMinRate:  input.PerMinRate,
			MinFare:     input.MinFare,
			MaxFare:     input.MaxFare,
			Description: input.Description,
			IsActive:    true,
		}

		if err := db.Create(&pricingZone).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to create pricing zone"})
			return
		}

		c.JSON(201, pricingZone)
	}
}

// GetPricingZones retrieves all pricing zones
func GetPricingZones(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var zones []models.PricingZone
		if err := db.Where("is_active = ?", true).Find(&zones).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch pricing zones"})
			return
		}

		c.JSON(200, zones)
	}
}

// SetDriverPricing sets driver-specific pricing for a zone
func SetDriverPricing(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can set pricing"})
			return
		}

		var input struct {
			ZoneID     uint     `json:"zoneId" binding:"required"`
			BaseFare   *float64 `json:"baseFare,omitempty"`
			PerKmRate  *float64 `json:"perKmRate,omitempty"`
			PerMinRate *float64 `json:"perMinRate,omitempty"`
			MinFare    *float64 `json:"minFare,omitempty"`
			MaxFare    *float64 `json:"maxFare,omitempty"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Check if zone exists
		var zone models.PricingZone
		if err := db.First(&zone, input.ZoneID).Error; err != nil {
			c.JSON(404, gin.H{"error": "Pricing zone not found"})
			return
		}

		// Validate pricing overrides
		if input.BaseFare != nil && *input.BaseFare < 0 {
			c.JSON(400, gin.H{"error": "Base fare must be non-negative"})
			return
		}
		if input.PerKmRate != nil && *input.PerKmRate < 0 {
			c.JSON(400, gin.H{"error": "Per km rate must be non-negative"})
			return
		}
		if input.PerMinRate != nil && *input.PerMinRate < 0 {
			c.JSON(400, gin.H{"error": "Per minute rate must be non-negative"})
			return
		}
		if input.MinFare != nil && input.MaxFare != nil && *input.MinFare > *input.MaxFare {
			c.JSON(400, gin.H{"error": "Minimum fare cannot be greater than maximum fare"})
			return
		}

		// Check if driver pricing already exists
		var driverPricing models.DriverPricing
		result := db.Where("driver_id = ? AND zone_id = ?", driverID, input.ZoneID).First(&driverPricing)

		if result.Error == gorm.ErrRecordNotFound {
			// Create new driver pricing
			driverPricing = models.DriverPricing{
				DriverID:   driverID,
				ZoneID:     input.ZoneID,
				BaseFare:   input.BaseFare,
				PerKmRate:  input.PerKmRate,
				PerMinRate: input.PerMinRate,
				MinFare:    input.MinFare,
				MaxFare:    input.MaxFare,
				IsActive:   true,
			}
			if err := db.Create(&driverPricing).Error; err != nil {
				c.JSON(500, gin.H{"error": "Failed to create driver pricing"})
				return
			}
		} else if result.Error != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch driver pricing"})
			return
		} else {
			// Update existing driver pricing
			if input.BaseFare != nil {
				driverPricing.BaseFare = input.BaseFare
			}
			if input.PerKmRate != nil {
				driverPricing.PerKmRate = input.PerKmRate
			}
			if input.PerMinRate != nil {
				driverPricing.PerMinRate = input.PerMinRate
			}
			if input.MinFare != nil {
				driverPricing.MinFare = input.MinFare
			}
			if input.MaxFare != nil {
				driverPricing.MaxFare = input.MaxFare
			}

			if err := db.Save(&driverPricing).Error; err != nil {
				c.JSON(500, gin.H{"error": "Failed to update driver pricing"})
				return
			}
		}

		c.JSON(200, driverPricing)
	}
}

// GetDriverPricing retrieves driver's pricing for all zones
func GetDriverPricing(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := c.GetUint("userId")
		userType := c.GetString("userType")

		if userType != string(models.UserTypeDriver) {
			c.JSON(403, gin.H{"error": "Only drivers can view pricing"})
			return
		}

		var pricing []models.DriverPricing
		if err := db.Preload("Zone").
			Where("driver_id = ? AND is_active = ?", driverID, true).
			Find(&pricing).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch driver pricing"})
			return
		}

		c.JSON(200, pricing)
	}
}

// CalculateFare calculates fare for a trip
func CalculateFare(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		latStr := c.Query("lat")
		lngStr := c.Query("lng")
		distanceStr := c.Query("distance")
		durationStr := c.Query("duration")

		if latStr == "" || lngStr == "" || distanceStr == "" || durationStr == "" {
			c.JSON(400, gin.H{"error": "Latitude, longitude, distance, and duration are required"})
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

		distance, err := strconv.ParseFloat(distanceStr, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid distance"})
			return
		}

		duration, err := strconv.Atoi(durationStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid duration"})
			return
		}

		// Find pricing zone for the location
		var zone models.PricingZone
		if err := db.Where("is_active = ?", true).Find(&zone).Error; err != nil {
			c.JSON(500, gin.H{"error": "No pricing zones found"})
			return
		}

		// Check if location is within any zone
		var applicableZone *models.PricingZone
		for _, z := range []models.PricingZone{zone} {
			if utils.IsWithinRadius(lat, lng, z.CenterLat, z.CenterLng, z.Radius) {
				applicableZone = &z
				break
			}
		}

		if applicableZone == nil {
			// Use default pricing
			fare := utils.CalculatePrice(distance, 2.0) // Default 2.0 per km
			c.JSON(200, gin.H{
				"fare":     fare,
				"distance": distance,
				"duration": duration,
				"zone":     nil,
			})
			return
		}

		// Calculate fare based on zone pricing
		baseFare := applicableZone.BaseFare
		distanceFare := distance * applicableZone.PerKmRate
		timeFare := float64(duration) * applicableZone.PerMinRate

		totalFare := baseFare + distanceFare + timeFare

		// Apply minimum and maximum fare constraints
		if totalFare < applicableZone.MinFare {
			totalFare = applicableZone.MinFare
		}
		if totalFare > applicableZone.MaxFare {
			totalFare = applicableZone.MaxFare
		}

		c.JSON(200, gin.H{
			"fare":     totalFare,
			"distance": distance,
			"duration": duration,
			"zone": gin.H{
				"id":   applicableZone.ID,
				"name": applicableZone.Name,
			},
			"breakdown": gin.H{
				"baseFare":     baseFare,
				"distanceFare": distanceFare,
				"timeFare":     timeFare,
			},
		})
	}
}
