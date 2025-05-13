package models

import (
	"time"

	"gorm.io/gorm"
)

// OTPType defines the purpose of the OTP
type OTPType string

const (
	OTPTypePasswordReset OTPType = "password_reset"
)

// OTP model for storing one-time passwords
type OTP struct {
	gorm.Model
	UserID    uint      `json:"user_id"`
	Code      string    `json:"code"`
	Type      OTPType   `json:"type"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used" gorm:"default:false"`
}

// IsValid checks if the OTP is valid (not expired and not used)
func (o *OTP) IsValid() bool {
	return !o.Used && time.Now().Before(o.ExpiresAt)
}

// MarkAsUsed marks the OTP as used
func (o *OTP) MarkAsUsed(db *gorm.DB) error {
	o.Used = true
	return db.Save(o).Error
} 