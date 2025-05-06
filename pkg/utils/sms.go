package utils

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	username = os.Getenv("AT_USERNAME")
	apiKey   = os.Getenv("AT_API_KEY")
)

func sendSMS(message string, recipients []string) error {
	log.Printf("Attempting to send SMS. Username: %s, APIKey length: %d", username, len(apiKey))
	
	if username == "" {
		return fmt.Errorf("africa's talking username not set")
	}

	if apiKey == "" {
		return fmt.Errorf("africa's talking API key not set")
	}

	baseURL := "https://api.africastalking.com/version1/messaging"
	log.Printf("Sending SMS to recipients: %v", recipients)
	log.Printf("Message content: %s", message)

	// Prepare the form data
	data := url.Values{}
	data.Set("username", username)
	data.Set("to", strings.Join(recipients, ","))
	data.Set("message", message)

	// Create the request
	req, err := http.NewRequest("POST", baseURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("apiKey", apiKey)
	req.Header.Set("Accept", "application/json")

	// Send the request
	client := &http.Client{}
	log.Printf("Sending HTTP request to Africa's Talking API...")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	log.Printf("Received response with status code: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send SMS: status code %d", resp.StatusCode)
	}

	log.Printf("Successfully sent SMS to recipients")
	return nil
}

func SendNewBookingNotificationToDriver(driverPhone, destination, clientName string) error {
	msg := fmt.Sprintf("Your ride to %s has been booked by %s. Please log in to accept or reject the booking.",
		destination, clientName)

	return sendSMS(msg, []string{driverPhone})
}

func SendBookingAcceptedSMS(clientPhone, driverName, carPlate, receiverPhone, receiverName string) error {
	// Message to client
	clientMsg := fmt.Sprintf("Your booking has been accepted by driver %s (Car: %s). Your parcel is now ready for delivery.",
		driverName, carPlate)

	// Message to receiver
	receiverMsg := fmt.Sprintf("Hello %s, a parcel is being delivered to you by %s (Car: %s). You will be notified when the parcel arrives.",
		receiverName, driverName, carPlate)

	// Send to client
	if err := sendSMS(clientMsg, []string{clientPhone}); err != nil {
		return fmt.Errorf("failed to send SMS to client: %v", err)
	}

	// Send to receiver
	if err := sendSMS(receiverMsg, []string{receiverPhone}); err != nil {
		return fmt.Errorf("failed to send SMS to receiver: %v", err)
	}

	return nil
}

func SendBookingRejectedSMS(clientPhone string) error {
	msg := "Your booking has been rejected by the driver. Please try booking another available ride."
	return sendSMS(msg, []string{clientPhone})
}
