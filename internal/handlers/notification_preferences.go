package handlers

import (
	"context"
	"log"

	"github.com/chachabrian/mooveit-backend/internal/models"
	"github.com/chachabrian/mooveit-backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetNotificationPreferences retrieves user's notification preferences
func GetNotificationPreferences(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("userId")

		var preferences models.NotificationPreference
		if err := db.Where("user_id = ?", userID).First(&preferences).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create default preferences if not found
				defaultPrefs := models.DefaultPreferences(userID)
				if err := db.Create(defaultPrefs).Error; err != nil {
					c.JSON(500, gin.H{"error": "Failed to create default preferences"})
					return
				}
				c.JSON(200, defaultPrefs)
				return
			}
			c.JSON(500, gin.H{"error": "Failed to fetch preferences"})
			return
		}

		c.JSON(200, preferences)
	}
}

// UpdateNotificationPreferences updates user's notification preferences
func UpdateNotificationPreferences(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("userId")

		var input struct {
			PushEnabled         *bool `json:"pushEnabled"`
			AvailableRidesPush  *bool `json:"availableRidesPush"`
			BookingAlerts       *bool `json:"bookingAlerts"`
			RideRequestAlerts   *bool `json:"rideRequestAlerts"`
			RideStatusAlerts    *bool `json:"rideStatusAlerts"`
			PromotionalMessages *bool `json:"promotionalMessages"`
			EmailEnabled        *bool `json:"emailEnabled"`
			SMSEnabled          *bool `json:"smsEnabled"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Get existing preferences or create default
		var preferences models.NotificationPreference
		err := db.Where("user_id = ?", userID).First(&preferences).Error
		if err == gorm.ErrRecordNotFound {
			preferences = *models.DefaultPreferences(userID)
			if err := db.Create(&preferences).Error; err != nil {
				c.JSON(500, gin.H{"error": "Failed to create preferences"})
				return
			}
		} else if err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch preferences"})
			return
		}

		// Track changes for topic subscription
		oldAvailableRidesPush := preferences.AvailableRidesPush

		// Update only provided fields
		if input.PushEnabled != nil {
			preferences.PushEnabled = *input.PushEnabled
		}
		if input.AvailableRidesPush != nil {
			preferences.AvailableRidesPush = *input.AvailableRidesPush
		}
		if input.BookingAlerts != nil {
			preferences.BookingAlerts = *input.BookingAlerts
		}
		if input.RideRequestAlerts != nil {
			preferences.RideRequestAlerts = *input.RideRequestAlerts
		}
		if input.RideStatusAlerts != nil {
			preferences.RideStatusAlerts = *input.RideStatusAlerts
		}
		if input.PromotionalMessages != nil {
			preferences.PromotionalMessages = *input.PromotionalMessages
		}
		if input.EmailEnabled != nil {
			preferences.EmailEnabled = *input.EmailEnabled
		}
		if input.SMSEnabled != nil {
			preferences.SMSEnabled = *input.SMSEnabled
		}

		// Save preferences
		if err := db.Save(&preferences).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to update preferences"})
			return
		}

		// Handle topic subscription for available rides
		if input.AvailableRidesPush != nil && oldAvailableRidesPush != preferences.AvailableRidesPush {
			var user models.User
			if err := db.First(&user, userID).Error; err == nil && user.FCMToken != "" {
				ctx := context.Background()
				tokens := []string{user.FCMToken}

				if preferences.AvailableRidesPush && preferences.PushEnabled {
					// Subscribe to available rides topic
					if err := services.SubscribeToTopic(ctx, tokens, "clients-available-rides"); err != nil {
						log.Printf("Failed to subscribe user %d to available rides topic: %v", userID, err)
					} else {
						log.Printf("User %d subscribed to available rides notifications", userID)
					}
				} else {
					// Unsubscribe from available rides topic
					if err := services.UnsubscribeFromTopic(ctx, tokens, "clients-available-rides"); err != nil {
						log.Printf("Failed to unsubscribe user %d from available rides topic: %v", userID, err)
					} else {
						log.Printf("User %d unsubscribed from available rides notifications", userID)
					}
				}
			}
		}

		c.JSON(200, gin.H{
			"message":     "Preferences updated successfully",
			"preferences": preferences,
		})
	}
}

// SubscribeToAvailableRides subscribes user to available rides topic (called when FCM token is registered)
func SubscribeToAvailableRides(db *gorm.DB, userID uint, fcmToken string) error {
	var preferences models.NotificationPreference
	err := db.Where("user_id = ?", userID).First(&preferences).Error
	if err == gorm.ErrRecordNotFound {
		// User has default preferences, subscribe them
		ctx := context.Background()
		return services.SubscribeToTopic(ctx, []string{fcmToken}, "clients-available-rides")
	} else if err != nil {
		return err
	}

	// Only subscribe if preferences allow it
	if preferences.PushEnabled && preferences.AvailableRidesPush {
		ctx := context.Background()
		return services.SubscribeToTopic(ctx, []string{fcmToken}, "clients-available-rides")
	}

	return nil
}

