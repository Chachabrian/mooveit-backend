package utils

import (
	"crypto/sha256"
	"fmt"
	"log"
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
// If SMS fails, it logs the error but doesn't fail the entire operation
func SendPasswordResetOTP(email, phone, otp string) error {
	var emailSent bool
	var errors []string

	log.Printf("Attempting to send OTP to email: %s, phone: %s", email, phone)

	// Send via email - this is critical and must succeed
	if err := SendPasswordResetEmail(email, otp); err != nil {
		errors = append(errors, fmt.Sprintf("failed to send OTP via email: %v", err))
	} else {
		emailSent = true
	}

	// Send via SMS if phone is provided - this is optional
	if phone != "" {
		if err := SendPasswordResetSMS(phone, otp); err != nil {
			// Log SMS failure but don't fail the entire operation
			log.Printf("Warning: Failed to send OTP via SMS to %s: %v", phone, err)
			errors = append(errors, fmt.Sprintf("failed to send OTP via SMS: %v", err))
		}
	}

	// If email was sent successfully, consider the operation successful
	if emailSent {
		if len(errors) > 0 {
			// Log warnings but return success
			log.Printf("OTP sent via email successfully. Some issues occurred: %v", errors)
		}
		return nil
	}

	// If email failed, return error
	if len(errors) > 0 {
		return fmt.Errorf("critical error: %s", errors[0])
	}

	return fmt.Errorf("unknown error occurred while sending OTP")
}
