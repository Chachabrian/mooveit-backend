package handlers

import (
	"strings"
	"time"

	"github.com/chachabrian/mooveit-backend/internal/models"
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

		c.JSON(201, ride)
	}
}

// GetAvailableRides retrieves all available rides with optional search
func GetAvailableRides(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		destination := c.Query("destination")
		currentLocation := c.Query("currentLocation")

		query := db.Model(&models.Ride{}).
			Joins("Driver").
			Select("rides.*, users.username, users.phone_number, users.car_plate, users.car_make, users.car_color").
			Where("rides.date > ? AND rides.status = ?", time.Now(), "available")

		if destination != "" {
			query = query.Where("LOWER(rides.destination) LIKE LOWER(?)", "%"+strings.ToLower(destination)+"%")
		}
		if currentLocation != "" {
			query = query.Where("LOWER(rides.current_location) LIKE LOWER(?)", "%"+strings.ToLower(currentLocation)+"%")
		}

		var rides []models.Ride
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
		if err := db.Where("driver_id = ?", userId).Find(&rides).Error; err != nil {
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
		if err := db.Model(&models.Ride{}).
			Joins("Driver").
			Select("rides.*, users.username, users.phone_number, users.car_plate, users.car_make, users.car_color").
			Find(&rides).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch rides"})
			return
		}

		c.JSON(200, rides)
	}
}
