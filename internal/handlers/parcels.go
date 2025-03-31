package handlers

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/chachabrian/mooveit-backend/internal/models"
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

		// Save the file to a directory
		uploadDir := "./uploads/parcels"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
			return
		}

		filePath := filepath.Join(uploadDir, file.Filename)
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}

		// Create parcel record
		parcel := models.Parcel{
			RideID:            input.RideID,
			ParcelImage:       filePath,
			ParcelDescription: input.ParcelDescription,
			ReceiverName:      input.ReceiverName,
			ReceiverContact:   input.ReceiverContact,
			Destination:       input.Destination,
		}

		if err := db.Create(&parcel).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create parcel"})
			return
		}

		c.JSON(http.StatusCreated, parcel)
	}
}

func GetParcelDetails(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        bookingId := c.Param("id") // Use "id" instead of "bookingId"

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

        c.JSON(200, gin.H{
            "parcelImage":      parcel.ParcelImage,
            "parcelDescription": parcel.ParcelDescription,
            "receiverName":     parcel.ReceiverName,
            "receiverContact":  parcel.ReceiverContact,
            "destination":      parcel.Destination,
        })
    }
}
