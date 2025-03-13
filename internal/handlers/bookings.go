package handlers

import (
	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/gin-gonic/gin"
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

// GetBookingStatus retrieves detailed booking information
func GetBookingStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookingId := c.Param("id")
		userId := c.GetUint("userId")

		var booking models.Booking
		if err := db.Preload("Ride").
			Preload("Ride.Driver").
			Preload("Client").
			First(&booking, bookingId).Error; err != nil {
			c.JSON(404, gin.H{"error": "Booking not found"})
			return
		}

		if booking.ClientID != userId && booking.Ride.DriverID != userId {
			c.JSON(403, gin.H{"error": "Unauthorized"})
			return
		}

		response := gin.H{
			"id":          booking.ID,
			"status":      booking.Status,
			"clientName":  booking.Client.Username,
			"clientPhone": booking.Client.PhoneNumber,
			"pickup":      booking.Ride.CurrentLocation,
			"destination": booking.Ride.Destination,
			"date":        booking.Ride.Date,
			"price":       booking.Ride.Price,
		}

		if booking.Ride.Driver != nil {
			response["driver"] = gin.H{
				"username":    booking.Ride.Driver.Username,
				"phoneNumber": booking.Ride.Driver.PhoneNumber,
				"carPlate":    booking.Ride.Driver.CarPlate,
				"carMake":     booking.Ride.Driver.CarMake,
				"carColor":    booking.Ride.Driver.CarColor,
			}
		}

		c.JSON(200, response)
	}
}

// GetClientBookings retrieves all bookings for a client
func GetClientBookings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetUint("userId")

		var bookings []models.Booking
		if err := db.Where("client_id = ?", userId).
			Preload("Ride").
			Preload("Ride.Driver").
			Find(&bookings).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch bookings"})
			return
		}

		c.JSON(200, bookings)
	}
}

// GetDriverBookings retrieves all bookings for a driver's rides
func GetDriverBookings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetUint("userId")

		var bookings []models.Booking
		if err := db.Joins("Ride").
			Where("rides.driver_id = ?", userId).
			Preload("Client").
			Preload("Ride").
			Find(&bookings).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch bookings"})
			return
		}

		c.JSON(200, bookings)
	}
}

// UpdateBookingStatus updates the status of a booking
func UpdateBookingStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookingId := c.Param("id")
		userId := c.GetUint("userId")

		var input struct {
			Status string `json:"status" binding:"required,oneof=accepted rejected"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var booking models.Booking
		if err := db.Preload("Ride").First(&booking, bookingId).Error; err != nil {
			c.JSON(404, gin.H{"error": "Booking not found"})
			return
		}

		if booking.Ride.DriverID != userId {
			c.JSON(403, gin.H{"error": "Unauthorized"})
			return
		}

		booking.Status = models.BookingStatus(input.Status)
		if err := db.Save(&booking).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to update booking status"})
			return
		}

		c.JSON(200, booking)
	}
}
