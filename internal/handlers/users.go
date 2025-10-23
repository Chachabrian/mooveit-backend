package handlers

import (
	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetProfile retrieves the user's profile
func GetProfile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetUint("userId")

		var user models.User
		if err := db.First(&user, userId).Error; err != nil {
			c.JSON(404, gin.H{"error": "User not found"})
			return
		}

		c.JSON(200, gin.H{
			"id":          user.ID,
			"email":       user.Email,
			"username":    user.Username,
			"phoneNumber": user.PhoneNumber,
			"userType":    user.UserType,
			"carPlate":    user.CarPlate,
			"carMake":     user.CarMake,
			"carColor":    user.CarColor,
		})
	}
}

// UpdateProfile updates the user's profile information
func UpdateProfile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetUint("userId")

		var input struct {
			Username    *string `json:"username"`
			PhoneNumber *string `json:"phoneNumber"`
			CarPlate    *string `json:"carPlate"`
			CarMake     *string `json:"carMake"`
			CarColor    *string `json:"carColor"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var user models.User
		if err := db.First(&user, userId).Error; err != nil {
			c.JSON(404, gin.H{"error": "User not found"})
			return
		}

		// Update fields individually to handle empty strings properly
		if input.Username != nil {
			user.Username = *input.Username
		}
		if input.PhoneNumber != nil {
			user.PhoneNumber = *input.PhoneNumber
		}
		if input.CarPlate != nil {
			user.CarPlate = *input.CarPlate
		}
		if input.CarMake != nil {
			user.CarMake = *input.CarMake
		}
		if input.CarColor != nil {
			user.CarColor = *input.CarColor
		}

		// Use Save() instead of Updates() to persist all fields including empty strings
		if err := db.Save(&user).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to update profile"})
			return
		}

		// Reload user from database to ensure we return the actual saved data
		if err := db.First(&user, userId).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to reload user data"})
			return
		}

		c.JSON(200, gin.H{
			"id":          user.ID,
			"email":       user.Email,
			"username":    user.Username,
			"phoneNumber": user.PhoneNumber,
			"userType":    user.UserType,
			"carPlate":    user.CarPlate,
			"carMake":     user.CarMake,
			"carColor":    user.CarColor,
		})
	}
}
