package handlers

import (
	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/chachabrian/mooveit-backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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

		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to hash password"})
			return
		}

		user := models.User{
			Username:     input.Username,
			Email:        input.Email,
			PasswordHash: string(hashedPassword),
			PhoneNumber:  input.Phone,
			UserType:     models.UserType(input.UserType), // Convert string to UserType
		}

		if result := db.Create(&user); result.Error != nil {
			c.JSON(500, gin.H{"error": "Failed to create user: " + result.Error.Error()})
			return
		}

		// Generate token for the newly registered user
		token, err := utils.GenerateToken(&user)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(201, gin.H{
			"message": "User created successfully",
			"token": token,
			"user": gin.H{
				"id":          user.ID,
				"email":       user.Email,
				"username":    user.Username,
				"phoneNumber": user.PhoneNumber,
				"userType":    user.UserType,
			},
		})
	}
}

type RegisterInput struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Phone    string `json:"phone"`
	UserType string `json:"userType" binding:"required,oneof=client driver"`
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
				"id":          user.ID,
				"email":       user.Email,
				"username":    user.Username,
				"phoneNumber": user.PhoneNumber,
				"userType":    user.UserType,
			},
		})
	}
}
