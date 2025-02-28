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
			"username":    user.Username,    // Changed from name to username
			"phoneNumber": user.PhoneNumber, // Changed from phone to phoneNumber
			"userType":    user.UserType,
		})
	}
}

// UpdateProfile updates the user's profile information
func UpdateProfile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetUint("userId")

		var input struct {
			Username    string `json:"username"`    // Changed from name to username
			PhoneNumber string `json:"phoneNumber"` // Changed from phone to phoneNumber
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

		updates := map[string]interface{}{
			"username":     input.Username,    // Changed from name to username
			"phone_number": input.PhoneNumber, // Changed from phone to phone_number
		}

		if err := db.Model(&user).Updates(updates).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to update profile"})
			return
		}

		c.JSON(200, gin.H{
			"id":          user.ID,
			"email":       user.Email,
			"username":    user.Username,    // Changed from name to username
			"phoneNumber": user.PhoneNumber, // Changed from phone to phoneNumber
			"userType":    user.UserType,
		})
	}
}
