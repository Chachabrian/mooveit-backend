package handlers

import (
	"context"
	"time"

	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/chachabrian/mooveit-backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterFCMToken registers or updates a user's FCM token
func RegisterFCMToken(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("userId")

		var input struct {
			FCMToken string `json:"fcmToken" binding:"required"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Update user's FCM token
		if err := db.Model(&models.User{}).Where("id = ?", userID).Update("fcm_token", input.FCMToken).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to register FCM token"})
			return
		}

		// Get user type to subscribe to appropriate topic
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to get user information"})
			return
		}

		// Subscribe to topic based on user type
		ctx := context.Background()
		topics := []string{}

		if user.UserType == models.UserTypeDriver {
			topics = append(topics, "drivers")
		} else {
			topics = append(topics, "clients")
			// For clients, also subscribe to available rides if preferences allow
			if err := SubscribeToAvailableRides(db, userID, input.FCMToken); err != nil {
				// Log but don't fail
				c.JSON(200, gin.H{
					"message": "FCM token registered successfully, but available rides subscription failed",
					"warning": err.Error(),
				})
				return
			}
			topics = append(topics, "available-rides-checked")
		}

		// Subscribe to main topic
		if len(topics) > 0 {
			if err := services.SubscribeToTopic(ctx, []string{input.FCMToken}, topics[0]); err != nil {
				// Log error but don't fail the request
				c.JSON(200, gin.H{
					"message": "FCM token registered successfully, but topic subscription failed",
					"warning": err.Error(),
				})
				return
			}
		}

		c.JSON(200, gin.H{
			"message": "FCM token registered and subscribed to topics",
			"topics":  topics,
		})
	}
}

// RemoveFCMToken removes a user's FCM token
func RemoveFCMToken(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("userId")

		// Clear user's FCM token
		if err := db.Model(&models.User{}).Where("id = ?", userID).Update("fcm_token", "").Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to remove FCM token"})
			return
		}

		c.JSON(200, gin.H{
			"message": "FCM token removed successfully",
		})
	}
}

// SendBroadcastNotification sends a broadcast notification to all users or specific user type
func SendBroadcastNotificationHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("userId")

		// Only allow admins to send broadcast notifications
		// For now, we'll allow any user (you can add admin check later)
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to get user information"})
			return
		}

		var input struct {
			Title    string                 `json:"title" binding:"required"`
			Body     string                 `json:"body" binding:"required"`
			UserType string                 `json:"userType"` // "all", "drivers", "clients"
			ImageURL string                 `json:"imageUrl"`
			Data     map[string]interface{} `json:"data"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Default to all users
		if input.UserType == "" {
			input.UserType = "all"
		}

		// Get tokens based on user type
		var users []models.User
		query := db.Where("fcm_token != ?", "")

		if input.UserType == "drivers" {
			query = query.Where("user_type = ?", models.UserTypeDriver)
		} else if input.UserType == "clients" {
			query = query.Where("user_type = ?", models.UserTypeClient)
		}

		if err := query.Find(&users).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch user tokens"})
			return
		}

		if len(users) == 0 {
			c.JSON(400, gin.H{"error": "No users with FCM tokens found"})
			return
		}

		// Extract tokens
		tokens := make([]string, 0, len(users))
		for _, u := range users {
			if u.FCMToken != "" {
				tokens = append(tokens, u.FCMToken)
			}
		}

		// Send broadcast notification
		ctx := context.Background()
		ctx = context.WithValue(ctx, "timestamp", time.Now().Unix())

		response, err := services.SendBroadcastNotification(ctx, tokens, input.Title, input.Body, input.ImageURL, input.Data)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to send broadcast notification", "details": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message":      "Broadcast notification sent successfully",
			"successCount": response.SuccessCount,
			"failureCount": response.FailureCount,
			"totalTokens":  len(tokens),
		})
	}
}

// NotifyScheduledRidesAvailable notifies clients about available scheduled rides
func NotifyScheduledRidesAvailable(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Count int `json:"count" binding:"required"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Get all client tokens
		var clients []models.User
		if err := db.Where("user_type = ? AND fcm_token != ?", models.UserTypeClient, "").Find(&clients).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch client tokens"})
			return
		}

		if len(clients) == 0 {
			c.JSON(400, gin.H{"error": "No clients with FCM tokens found"})
			return
		}

		// Extract tokens
		tokens := make([]string, 0, len(clients))
		for _, client := range clients {
			if client.FCMToken != "" {
				tokens = append(tokens, client.FCMToken)
			}
		}

		// Send notification
		ctx := context.Background()
		response, err := services.SendScheduledRidesAvailableNotification(ctx, tokens, input.Count)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to send notifications", "details": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message":      "Notifications sent successfully",
			"successCount": response.SuccessCount,
			"failureCount": response.FailureCount,
			"totalClients": len(tokens),
		})
	}
}

// TestNotification sends a test notification to the current user
func TestNotification(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("userId")

		// Get user's FCM token
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to get user information"})
			return
		}

		if user.FCMToken == "" {
			c.JSON(400, gin.H{"error": "No FCM token registered for this user"})
			return
		}

		// Send test notification
		ctx := context.Background()
		payload := services.NotificationPayload{
			Title: "Test Notification",
			Body:  "This is a test notification from MooveIt",
			Data: map[string]interface{}{
				"type":   "test",
				"userId": userID,
			},
		}

		if err := services.SendNotificationToToken(ctx, user.FCMToken, payload); err != nil {
			c.JSON(500, gin.H{"error": "Failed to send test notification", "details": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": "Test notification sent successfully",
		})
	}
}
