
package handlers

import (
    "github.com/gin-gonic/gin"
    "github.com/chachabrian/mooveit-backend/internal/models"
    "gorm.io/gorm"
)

// CreateBooking handles the creation of a new booking
func CreateBooking(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        userId := c.GetUint("userId")
        var input struct {
            RideID uint `json:"rideId" binding:"required"`
        }

        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        var ride models.Ride
        if err := db.First(&ride, input.RideID).Error; err != nil {
            c.JSON(404, gin.H{"error": "Ride not found"})
            return
        }

        booking := models.Booking{
            ClientID: userId,
            RideID:   input.RideID,
            Status:   models.BookingStatusPending,
        }

        if err := db.Create(&booking).Error; err != nil {
            c.JSON(500, gin.H{"error": "Failed to create booking"})
            return
        }

        c.JSON(201, booking)
    }
}

// GetBookingStatus retrieves the status of a specific booking
func GetBookingStatus(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        bookingId := c.Param("id")
        userId := c.GetUint("userId")

        var booking models.Booking
        if err := db.First(&booking, bookingId).Error; err != nil {
            c.JSON(404, gin.H{"error": "Booking not found"})
            return
        }

        if booking.ClientID != userId {
            c.JSON(403, gin.H{"error": "Unauthorized"})
            return
        }

        c.JSON(200, gin.H{"status": booking.Status})
    }
}
