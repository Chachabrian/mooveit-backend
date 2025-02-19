
package handlers

import (
    "github.com/gin-gonic/gin"
    "github.com/chachabrian/mooveit-backend/internal/models"
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
            "id":       user.ID,
            "email":    user.Email,
            "name":     user.Name,
            "phone":    user.Phone,
            "userType": user.UserType,
        })
    }
}

// UpdateProfile updates the user's profile information
func UpdateProfile(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        userId := c.GetUint("userId")

        var input struct {
            Name  string `json:"name"`
            Phone string `json:"phone"`
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
            "name":  input.Name,
            "phone": input.Phone,
        }

        if err := db.Model(&user).Updates(updates).Error; err != nil {
            c.JSON(500, gin.H{"error": "Failed to update profile"})
            return
        }

        c.JSON(200, gin.H{
            "id":       user.ID,
            "email":    user.Email,
            "name":     user.Name,
            "phone":    user.Phone,
            "userType": user.UserType,
        })
    }
}
