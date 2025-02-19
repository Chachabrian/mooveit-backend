
package handlers

import (
    "github.com/gin-gonic/gin"
    "github.com/chachabrian/mooveit-backend/internal/models"
    "gorm.io/gorm"
    "time"
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
            Destination string    `json:"destination" binding:"required"`
            TruckSize   string    `json:"truckSize" binding:"required"`
            Price       float64   `json:"price" binding:"required"`
            Date        time.Time `json:"date" binding:"required"`
        }

        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        ride := models.Ride{
            DriverID:    userId,
            Destination: input.Destination,
            TruckSize:   input.TruckSize,
            Price:       input.Price,
            Date:        input.Date,
        }

        if err := db.Create(&ride).Error; err != nil {
            c.JSON(500, gin.H{"error": "Failed to create ride"})
            return
        }

        c.JSON(201, ride)
    }
}

// GetAvailableRides retrieves all available rides
func GetAvailableRides(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        var rides []models.Ride
        if err := db.Where("date > ?", time.Now()).Find(&rides).Error; err != nil {
            c.JSON(500, gin.H{"error": "Failed to fetch rides"})
            return
        }

        c.JSON(200, rides)
    }
}
