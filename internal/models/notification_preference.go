package models

import (
	"time"

	"gorm.io/gorm"
)

// NotificationPreference represents user notification preferences
type NotificationPreference struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"uniqueIndex;not null" json:"userId"`
	User      User           `gorm:"foreignKey:UserID" json:"-"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// General push notification toggle
	PushEnabled bool `gorm:"column:push_enabled;default:true" json:"pushEnabled"`

	// Specific notification preferences
	AvailableRidesPush  bool `gorm:"column:available_rides_push;default:true" json:"availableRidesPush"`
	BookingAlerts       bool `gorm:"column:booking_alerts;default:true" json:"bookingAlerts"`
	RideRequestAlerts   bool `gorm:"column:ride_request_alerts;default:true" json:"rideRequestAlerts"`
	RideStatusAlerts    bool `gorm:"column:ride_status_alerts;default:true" json:"rideStatusAlerts"`
	PromotionalMessages bool `gorm:"column:promotional_messages;default:true" json:"promotionalMessages"`

	// Email and SMS preferences
	EmailEnabled bool `gorm:"column:email_enabled;default:true" json:"emailEnabled"`
	SMSEnabled   bool `gorm:"column:sms_enabled;default:true" json:"smsEnabled"`
}

// TableName specifies the table name for NotificationPreference
func (NotificationPreference) TableName() string {
	return "notification_preferences"
}

// DefaultPreferences returns default notification preferences for a new user
func DefaultPreferences(userID uint) *NotificationPreference {
	return &NotificationPreference{
		UserID:              userID,
		PushEnabled:         true,
		AvailableRidesPush:  true,
		BookingAlerts:       true,
		RideRequestAlerts:   true,
		RideStatusAlerts:    true,
		PromotionalMessages: true,
		EmailEnabled:        true,
		SMSEnabled:          true,
	}
}
