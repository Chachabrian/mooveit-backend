package handlers

import (
	"log"

	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/chachabrian/mooveit-backend/pkg/utils"
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

		// Start a transaction
		tx := db.Begin()
		if tx.Error != nil {
			c.JSON(500, gin.H{"error": "Failed to start transaction"})
			return
		}

		// Load the ride with driver information
		var ride models.Ride
		if err := tx.Preload("Driver").First(&ride, input.RideID).Error; err != nil {
			tx.Rollback()
			c.JSON(404, gin.H{"error": "Ride not found"})
			return
		}

		// Load the client information
		var client models.User
		if err := tx.First(&client, userId).Error; err != nil {
			tx.Rollback()
			c.JSON(404, gin.H{"error": "Client not found"})
			return
		}

		booking := models.Booking{
			ClientID: userId,
			RideID:   input.RideID,
			Status:   models.BookingStatusPending,
		}

		if err := tx.Create(&booking).Error; err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Failed to create booking"})
			return
		}

		// Send SMS notification to driver
		if err := utils.SendNewBookingNotificationToDriver(
			ride.Driver.PhoneNumber,
			ride.Destination,
			client.Username,
		); err != nil {
			// Log the error but don't fail the transaction
			log.Printf("Failed to send SMS to driver: %v", err)
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to commit transaction"})
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
		result := db.Debug(). // Add Debug() to see the SQL query
					Preload("Ride").
					Preload("Ride.Driver").
					Preload("Client").
					Joins("JOIN rides ON rides.id = bookings.ride_id").
					Where("rides.driver_id = ?", userId).
					Order("bookings.created_at DESC").
					Find(&bookings)

		if result.Error != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch bookings: " + result.Error.Error()})
			return
		}

		// If no bookings found, return empty array instead of null
		if len(bookings) == 0 {
			c.JSON(200, []gin.H{})
			return
		}

		response := make([]gin.H, 0)
		for _, booking := range bookings {
			// Check if required relationships are loaded
			if booking.Client.ID == 0 || booking.Ride.ID == 0 {
				continue
			}

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
				},
				"createdAt": booking.CreatedAt,
			}

			// Only add driver details if available
			if booking.Ride.Driver != nil {
				bookingDetails["ride"].(gin.H)["driver"] = gin.H{
					"username":    booking.Ride.Driver.Username,
					"phoneNumber": booking.Ride.Driver.PhoneNumber,
					"carPlate":    booking.Ride.Driver.CarPlate,
					"carMake":     booking.Ride.Driver.CarMake,
					"carColor":    booking.Ride.Driver.CarColor,
				}
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

			// Load necessary information for SMS
			var client models.User
			if err := tx.First(&client, booking.ClientID).Error; err != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"error": "Failed to load client information"})
				return
			}

			var driver models.User
			if err := tx.First(&driver, booking.Ride.DriverID).Error; err != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"error": "Failed to load driver information"})
				return
			}

			var parcel models.Parcel
			if err := tx.Where("ride_id = ?", booking.RideID).First(&parcel).Error; err != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"error": "Failed to load parcel information"})
				return
			}

			// Send SMS notifications
			if err := utils.SendBookingAcceptedSMS(
				client.PhoneNumber,
				driver.Username,
				driver.CarPlate,
				parcel.ReceiverContact,
				parcel.ReceiverName,
			); err != nil {
				// Log the error but don't fail the transaction
				log.Printf("Failed to send SMS: %v", err)
			}

		} else if input.Status == "cancelled" || input.Status == "rejected" {
			// Reset ride status to available if booking is cancelled or rejected
			if err := tx.Model(&booking.Ride).Update("status", "available").Error; err != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"error": "Failed to update ride status"})
				return
			}

			if input.Status == "rejected" {
				// Load client information for SMS
				var client models.User
				if err := tx.First(&client, booking.ClientID).Error; err != nil {
					tx.Rollback()
					c.JSON(500, gin.H{"error": "Failed to load client information"})
					return
				}

				// Send rejection SMS to client
				if err := utils.SendBookingRejectedSMS(client.PhoneNumber); err != nil {
					// Log the error but don't fail the transaction
					log.Printf("Failed to send rejection SMS: %v", err)
				}
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
