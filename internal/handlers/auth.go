package handlers

import (
	"fmt"
	"time"

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
			IsVerified:   false,                           // New users start unverified
		}

		if result := db.Create(&user); result.Error != nil {
			c.JSON(500, gin.H{"error": "Failed to create user: " + result.Error.Error()})
			return
		}

		// Generate and send email verification OTP
		timestamp := time.Now().Format("20060102150405")
		uniqueKey := fmt.Sprintf("%s-verification-%s", user.Email, timestamp)
		otp := utils.GenerateOTP(uniqueKey)

		// Save OTP in database
		otpRecord := models.OTP{
			UserID:    user.ID,
			Code:      otp,
			Type:      models.OTPTypeEmailVerification,
			ExpiresAt: time.Now().Add(utils.OTPExpiration),
		}

		if result := db.Create(&otpRecord); result.Error != nil {
			c.JSON(500, gin.H{"error": "Failed to generate verification OTP"})
			return
		}

		// Send verification email
		if err := utils.SendEmailVerificationOTP(user.Email, otp); err != nil {
			c.JSON(500, gin.H{"error": "Failed to send verification email: " + err.Error()})
			return
		}

		c.JSON(201, gin.H{
			"message": "User created successfully. Please check your email for verification code.",
			"user": gin.H{
				"id":          user.ID,
				"email":       user.Email,
				"username":    user.Username,
				"phoneNumber": user.PhoneNumber,
				"userType":    user.UserType,
				"isVerified":  user.IsVerified,
			},
			"requiresVerification": true,
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

		// Check if email is verified
		if !user.IsVerified {
			// Generate and send email verification OTP
			timestamp := time.Now().Format("20060102150405")
			uniqueKey := fmt.Sprintf("%s-login-verification-%s", user.Email, timestamp)
			otp := utils.GenerateOTP(uniqueKey)

			// Invalidate any existing verification OTPs for this user
			db.Model(&models.OTP{}).
				Where("user_id = ? AND type = ? AND used = ? AND expires_at > ?",
					user.ID, models.OTPTypeEmailVerification, false, time.Now()).
				Update("used", true)

			// Save new OTP in database
			otpRecord := models.OTP{
				UserID:    user.ID,
				Code:      otp,
				Type:      models.OTPTypeEmailVerification,
				ExpiresAt: time.Now().Add(utils.OTPExpiration),
			}

			if result := db.Create(&otpRecord); result.Error != nil {
				c.JSON(500, gin.H{"error": "Failed to generate verification OTP"})
				return
			}

			// Send verification email
			if err := utils.SendEmailVerificationOTP(user.Email, otp); err != nil {
				c.JSON(500, gin.H{"error": "Failed to send verification email: " + err.Error()})
				return
			}

			c.JSON(200, gin.H{
				"message":              "Email verification required. Check your email for verification code.",
				"requiresVerification": true,
				"email":                user.Email,
			})
			return
		}

		// Generate token for verified users
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
				"isVerified":  user.IsVerified,
			},
		})
	}
}

// ResetPasswordRequestInput defines the input for requesting a password reset
type ResetPasswordRequestInput struct {
	Email string `json:"email" binding:"required,email"`
}

// RequestPasswordReset initiates the password reset process
func RequestPasswordReset(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input ResetPasswordRequestInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Find user by email
		var user models.User
		if result := db.Where("email = ?", input.Email).First(&user); result.Error != nil {
			c.JSON(404, gin.H{"error": "User not found"})
			return
		}

		// First, invalidate all previous unused OTPs for this user
		if result := db.Model(&models.OTP{}).
			Where("user_id = ? AND type = ? AND used = ? AND expires_at > ?",
				user.ID, models.OTPTypePasswordReset, false, time.Now()).
			Update("used", true); result.Error != nil {
			c.JSON(500, gin.H{"error": "Failed to invalidate previous OTPs"})
			return
		}

		// Generate a unique OTP with timestamp for this reset request
		// This adds randomness to make each OTP request unique
		timestamp := time.Now().Format("20060102150405")
		uniqueKey := fmt.Sprintf("%s-%s", input.Email, timestamp)
		otp := utils.GenerateOTP(uniqueKey)

		// Save OTP in database
		otpRecord := models.OTP{
			UserID:    user.ID,
			Code:      otp,
			Type:      models.OTPTypePasswordReset,
			ExpiresAt: time.Now().Add(utils.OTPExpiration),
		}

		if result := db.Create(&otpRecord); result.Error != nil {
			c.JSON(500, gin.H{"error": "Failed to generate OTP"})
			return
		}

		// Send OTP via email and SMS
		if err := utils.SendPasswordResetOTP(user.Email, user.PhoneNumber, otp); err != nil {
			c.JSON(500, gin.H{"error": "Failed to send OTP: " + err.Error()})
			return
		}

		// Determine delivery methods based on phone availability
		deliveryMethods := "email"
		if user.PhoneNumber != "" {
			deliveryMethods = "email and SMS"
		}

		c.JSON(200, gin.H{
			"message":          fmt.Sprintf("Password reset OTP sent successfully via %s", deliveryMethods),
			"delivery_methods": deliveryMethods,
		})
	}
}

// VerifyOTPInput defines the input for verifying OTP
type VerifyOTPInput struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required"`
}

// VerifyEmailInput defines the input for email verification
type VerifyEmailInput struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required"`
}

// ResetPasswordInput defines the input for resetting password
type ResetPasswordInput struct {
	Email       string `json:"email" binding:"required,email"`
	OTP         string `json:"otp" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// ResetPassword resets the user's password after OTP verification
func ResetPassword(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input ResetPasswordInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Find user by email
		var user models.User
		if result := db.Where("email = ?", input.Email).First(&user); result.Error != nil {
			c.JSON(404, gin.H{"error": "User not found"})
			return
		}

		// Find valid OTP
		var otpRecord models.OTP
		if result := db.Where("user_id = ? AND code = ? AND type = ? AND used = ? AND expires_at > ?",
			user.ID, input.OTP, models.OTPTypePasswordReset, false, time.Now()).
			First(&otpRecord); result.Error != nil {
			c.JSON(400, gin.H{"error": "Invalid or expired OTP"})
			return
		}

		// Immediately mark OTP as used to prevent race conditions
		if err := db.Model(&models.OTP{}).Where("id = ?", otpRecord.ID).Update("used", true).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to mark OTP as used"})
			return
		}

		// Hash the new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to hash password"})
			return
		}

		// Update user password
		user.PasswordHash = string(hashedPassword)
		if result := db.Save(&user); result.Error != nil {
			c.JSON(500, gin.H{"error": "Failed to update password"})
			return
		}

		c.JSON(200, gin.H{"message": "Password reset successful"})
	}
}

// VerifyOTP verifies if an OTP is valid
func VerifyOTP(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input VerifyOTPInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Find user by email
		var user models.User
		if result := db.Where("email = ?", input.Email).First(&user); result.Error != nil {
			c.JSON(404, gin.H{"error": "User not found"})
			return
		}

		// Find valid OTP
		var otpRecord models.OTP
		if result := db.Where("user_id = ? AND code = ? AND type = ? AND used = ? AND expires_at > ?",
			user.ID, input.OTP, models.OTPTypePasswordReset, false, time.Now()).
			First(&otpRecord); result.Error != nil {
			c.JSON(400, gin.H{"error": "Invalid or expired OTP"})
			return
		}

		// OTP is valid but we don't mark it as used yet
		// It will be used during the actual password reset
		c.JSON(200, gin.H{"message": "OTP verified successfully", "valid": true})
	}
}

// VerifyEmail verifies user's email with OTP and generates login token
func VerifyEmail(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input VerifyEmailInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Find user by email
		var user models.User
		if result := db.Where("email = ?", input.Email).First(&user); result.Error != nil {
			c.JSON(404, gin.H{"error": "User not found"})
			return
		}

		// Find valid email verification OTP
		var otpRecord models.OTP
		if result := db.Where("user_id = ? AND code = ? AND type = ? AND used = ? AND expires_at > ?",
			user.ID, input.OTP, models.OTPTypeEmailVerification, false, time.Now()).
			First(&otpRecord); result.Error != nil {
			c.JSON(400, gin.H{"error": "Invalid or expired verification code"})
			return
		}

		// Mark OTP as used
		if err := db.Model(&models.OTP{}).Where("id = ?", otpRecord.ID).Update("used", true).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to mark OTP as used"})
			return
		}

		// Mark user as verified
		if err := db.Model(&models.User{}).Where("id = ?", user.ID).Update("is_verified", true).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to verify user"})
			return
		}

		// Update user object
		user.IsVerified = true

		// Generate token for verified user
		token, err := utils.GenerateToken(&user)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(200, gin.H{
			"message": "Email verified successfully",
			"token":   token,
			"user": gin.H{
				"id":          user.ID,
				"email":       user.Email,
				"username":    user.Username,
				"phoneNumber": user.PhoneNumber,
				"userType":    user.UserType,
				"isVerified":  user.IsVerified,
			},
		})
	}
}
