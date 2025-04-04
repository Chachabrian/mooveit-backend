package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

		// Use absolute path for uploads
		uploadDir := "/app/uploads/parcels"

		// Create directory if it doesn't exist
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory: " + err.Error()})
			return
		}

		// Generate unique filename using timestamp
		fileExt := filepath.Ext(file.Filename)
		fileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), fileExt)
		filePath := filepath.Join(uploadDir, fileName)

		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
			return
		}

		// Store relative path in database
		dbPath := filepath.Join("uploads/parcels", fileName)

		// First verify if the ride exists
		var ride models.Ride
		if err := db.First(&ride, input.RideID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ride ID"})
			return
		}

		// Create parcel record
		parcel := models.Parcel{
			RideID:            input.RideID,
			ParcelImage:       dbPath,
			ParcelDescription: input.ParcelDescription,
			ReceiverName:      input.ReceiverName,
			ReceiverContact:   input.ReceiverContact,
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
		// Get base URL from environment variable or configuration
		baseURL := os.Getenv("BASE_URL") // e.g., "https://api.yourdomain.com"
		if baseURL == "" {
			baseURL = "http://localhost:8080" // fallback for development
		}

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

		// Construct full URL for parcel image
		fullImageURL := fmt.Sprintf("%s/%s", baseURL, parcel.ParcelImage)

		c.JSON(200, gin.H{
			"parcelImage":       fullImageURL,
			"parcelDescription": parcel.ParcelDescription,
			"receiverName":      parcel.ReceiverName,
			"receiverContact":   parcel.ReceiverContact,
			"destination":       parcel.Destination,
		})
	}
}
