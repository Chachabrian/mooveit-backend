package handlers

import (
	"fmt"
	"net/http"

	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/chachabrian/mooveit-backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateParcel(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			RideID            uint   `form:"rideId" binding:"required"`
			ParcelDescription string `form:"parcelDescription" binding:"required"`
			ReceiverName      string `form:"receiverName" binding:"required"`
			ReceiverContact   string `form:"receiverContact" binding:"required"`
			ReceiverEmail     string `form:"receiverEmail" binding:"required,email"`
			Destination       string `form:"destination" binding:"required"`
		}

		// Parse form data
		if err := c.ShouldBind(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Handle file upload
		file, err := c.FormFile("parcelImage")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parcel image is required"})
			return
		}

		// Upload to S3 or local storage
		imageURL, err := services.UploadImage(file, "parcels")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to upload image",
				"details": err.Error(),
			})
			return
		}

		// First verify if the ride exists
		var ride models.Ride
		if err := db.First(&ride, input.RideID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ride ID"})
			return
		}

		// Create parcel record with image URL
		parcel := models.Parcel{
			RideID:            input.RideID,
			ParcelImage:       imageURL, // Store full URL (S3) or relative path (local)
			ParcelDescription: input.ParcelDescription,
			ReceiverName:      input.ReceiverName,
			ReceiverContact:   input.ReceiverContact,
			ReceiverEmail:     input.ReceiverEmail,
			Destination:       input.Destination,
		}

		if err := db.Create(&parcel).Error; err != nil {
			// Log the actual error
			fmt.Printf("Database error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to create parcel",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, parcel)
	}
}

func GetParcelDetails(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookingId := c.Param("id")

		var booking models.Booking
		if err := db.Preload("Ride").First(&booking, bookingId).Error; err != nil {
			c.JSON(404, gin.H{"error": "Booking not found"})
			return
		}

		var parcel models.Parcel
		if err := db.Where("ride_id = ?", booking.RideID).First(&parcel).Error; err != nil {
			c.JSON(404, gin.H{"error": "Parcel details not found"})
			return
		}

		// Get full image URL (handles both S3 and local storage)
		imageURL := services.GetImageURL(parcel.ParcelImage)

		c.JSON(200, gin.H{
			"parcelImage":       imageURL,
			"parcelDescription": parcel.ParcelDescription,
			"receiverName":      parcel.ReceiverName,
			"receiverContact":   parcel.ReceiverContact,
			"receiverEmail":     parcel.ReceiverEmail,
			"destination":       parcel.Destination,
		})
	}
}
