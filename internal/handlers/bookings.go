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
		if err := db.Preload("Ride").
			Preload("Ride.Driver").
			Preload("Client").
			Joins("JOIN rides ON rides.id = bookings.ride_id").
			Where("rides.driver_id = ?", userId).
			Order("bookings.created_at DESC"). // Show newest bookings first
			Find(&bookings).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch bookings: " + err.Error()})
			return
		}

		// Format the response to match the frontend requirements
		var response []gin.H
		for _, booking := range bookings {
			bookingDetails := gin.H{
				"id":     booking.ID,
				"status": booking.Status,
				"client": gin.H{
					"username":    booking.Client.Username,
					"phoneNumber": booking.Client.PhoneNumber,
				},
				"ride": gin.H{
					"id":              booking.Ride.ID,
					"currentLocation": booking.Ride.CurrentLocation,
					"destination":     booking.Ride.Destination,
					"date":            booking.Ride.Date,
					"price":           booking.Ride.Price,
					"status":          booking.Ride.Status,
					"driver": gin.H{
						"username":    booking.Ride.Driver.Username,
						"phoneNumber": booking.Ride.Driver.PhoneNumber,
						"carPlate":    booking.Ride.Driver.CarPlate,
						"carMake":     booking.Ride.Driver.CarMake,
						"carColor":    booking.Ride.Driver.CarColor,
					},
				},
				"createdAt": booking.CreatedAt,
			}
			response = append(response, bookingDetails)
		}

		c.JSON(200, response)
	}
}

// UpdateBookingStatus updates the status of a booking
func UpdateBookingStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookingId := c.Param("id")
		userId := c.GetUint("userId")

		var input struct {
			Status string `json:"status" binding:"required,oneof=accepted rejected cancelled"`
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

		// Check permissions based on user type and action
		if input.Status == "cancelled" {
			// Only clients can cancel their own bookings
			if booking.ClientID != userId {
				c.JSON(403, gin.H{"error": "Only the client can cancel this booking"})
				return
			}
		} else {
			// Only drivers can accept/reject bookings for their rides
			if booking.Ride.DriverID != userId {
				c.JSON(403, gin.H{"error": "Only the driver can accept/reject this booking"})
				return
			}
		}

		// Start a transaction
		tx := db.Begin()

		// Update booking status
		booking.Status = models.BookingStatus(input.Status)
		if err := tx.Save(&booking).Error; err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Failed to update booking status"})
			return
		}

		// Update ride status based on booking status
		if input.Status == "accepted" {
			if err := tx.Model(&booking.Ride).Update("status", "booked").Error; err != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"error": "Failed to update ride status"})
				return
			}
		} else if input.Status == "cancelled" || input.Status == "rejected" {
			// Reset ride status to available if booking is cancelled or rejected
			if err := tx.Model(&booking.Ride).Update("status", "available").Error; err != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"error": "Failed to update ride status"})
				return
			}
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(200, gin.H{
			"id":      booking.ID,
			"status":  booking.Status,
			"ride":    booking.Ride,
			"message": "Booking status updated successfully",
		})
	}
}
