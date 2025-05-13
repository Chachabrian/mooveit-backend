package utils

import (
	"crypto/sha256"
	"fmt"
	"time"
)

const (
	OTPExpiration = 15 * time.Minute
)

// GenerateOTP generates a 4-digit OTP based on the given unique key
// The unique key should be something that changes with each request
// like email + timestamp to ensure uniqueness
func GenerateOTP(uniqueKey string) string {
	// Create a hash from the unique key
	h := sha256.New()
	h.Write([]byte(uniqueKey))
	hash := h.Sum(nil)
	
	// Get 4 bytes from the hash
	num := uint32(hash[0])<<24 | uint32(hash[1])<<16 | uint32(hash[2])<<8 | uint32(hash[3])
	
	// Convert hash to a 4-digit number (1000-9999)
	otp := 1000 + (num % 9000)
	
	return fmt.Sprintf("%04d", otp)
}

// SendPasswordResetOTP sends OTP via both email and SMS
func SendPasswordResetOTP(email, phone, otp string) error {
	// Send via email
	if err := SendPasswordResetEmail(email, otp); err != nil {
		return fmt.Errorf("failed to send OTP via email: %v", err)
	}

	// Send via SMS if phone is provided
	if phone != "" {
		if err := SendPasswordResetSMS(phone, otp); err != nil {
			return fmt.Errorf("failed to send OTP via SMS: %v", err)
		}
	}

	return nil
} 