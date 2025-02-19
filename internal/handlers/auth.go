
package handlers

import (
    "github.com/gin-gonic/gin"
    "github.com/chachabrian/mooveit-backend/internal/models"
    "github.com/chachabrian/mooveit-backend/pkg/utils"
    "gorm.io/gorm"
)

type RegisterInput struct {
    Email    string          `json:"email" binding:"required,email"`
    Password string          `json:"password" binding:"required,min=6"`
    Name     string          `json:"name" binding:"required"`
    Phone    string          `json:"phone" binding:"required"`
    UserType models.UserType `json:"userType" binding:"required"`
}

type LoginInput struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

func Register(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        var input RegisterInput
        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        // Check if user already exists
        var existingUser models.User
        if result := db.Where("email = ?", input.Email).First(&existingUser); result.Error == nil {
            c.JSON(400, gin.H{"error": "User already exists"})
            return
        }

        user := models.User{
            Email:    input.Email,
            Password: input.Password,
            Name:     input.Name,
            Phone:    input.Phone,
            UserType: input.UserType,
        }

        if err := user.HashPassword(); err != nil {
            c.JSON(500, gin.H{"error": "Failed to hash password"})
            return
        }

        if result := db.Create(&user); result.Error != nil {
            c.JSON(500, gin.H{"error": "Failed to create user"})
            return
        }

        token, err := utils.GenerateToken(&user)
        if err != nil {
            c.JSON(500, gin.H{"error": "Failed to generate token"})
            return
        }

        c.JSON(201, gin.H{
            "token": token,
            "user": gin.H{
                "id":       user.ID,
                "email":    user.Email,
                "name":     user.Name,
                "phone":    user.Phone,
                "userType": user.UserType,
            },
        })
    }
}

func Login(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        var input LoginInput
        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        var user models.User
        if result := db.Where("email = ?", input.Email).First(&user); result.Error != nil {
            c.JSON(401, gin.H{"error": "Invalid credentials"})
            return
        }

        if err := user.CheckPassword(input.Password); err != nil {
            c.JSON(401, gin.H{"error": "Invalid credentials"})
            return
        }

        token, err := utils.GenerateToken(&user)
        if err != nil {
            c.JSON(500, gin.H{"error": "Failed to generate token"})
            return
        }

        c.JSON(200, gin.H{
            "token": token,
            "user": gin.H{
                "id":       user.ID,
                "email":    user.Email,
                "name":     user.Name,
                "phone":    user.Phone,
                "userType": user.UserType,
            },
        })
    }
}
