package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var (
	// FirebaseApp is the Firebase app instance
	FirebaseApp *firebase.App
	// MessagingClient is the Firebase Cloud Messaging client
	MessagingClient *messaging.Client
)

// InitFirebase initializes Firebase Admin SDK
func InitFirebase() error {
	ctx := context.Background()

	// Check if Firebase is configured
	serviceAccountPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH")
	if serviceAccountPath == "" {
		log.Println("Warning: FIREBASE_SERVICE_ACCOUNT_PATH not set. Push notifications will be disabled.")
		return nil
	}

	// Initialize Firebase app
	opt := option.WithCredentialsFile(serviceAccountPath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return fmt.Errorf("error initializing firebase app: %v", err)
	}

	// Initialize messaging client
	client, err := app.Messaging(ctx)
	if err != nil {
		return fmt.Errorf("error getting messaging client: %v", err)
	}

	FirebaseApp = app
	MessagingClient = client

	log.Println("Firebase Cloud Messaging initialized successfully")
	return nil
}

// NotificationPayload represents the notification data
type NotificationPayload struct {
	Title      string                 `json:"title"`
	Body       string                 `json:"body"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Image      string                 `json:"image,omitempty"`
	ChannelID  string                 `json:"channelId,omitempty"`  // Android notification channel
	Sound      string                 `json:"sound,omitempty"`      // Custom sound file name
	Icon       string                 `json:"icon,omitempty"`       // Android small icon
	Color      string                 `json:"color,omitempty"`      // Android notification color
	Priority   string                 `json:"priority,omitempty"`   // high, normal, low
	BadgeCount *int                   `json:"badgeCount,omitempty"` // iOS badge count
	Tag        string                 `json:"tag,omitempty"`        // Android notification tag
}

// getAndroidConfig returns Android-specific notification configuration
func getAndroidConfig(payload NotificationPayload) *messaging.AndroidConfig {
	channelID := payload.ChannelID
	if channelID == "" {
		channelID = "mooveit_default"
	}

	sound := payload.Sound
	if sound == "" {
		sound = "default"
	}

	icon := payload.Icon
	if icon == "" {
		icon = "ic_stat_logo"
	}

	color := payload.Color
	if color == "" {
		color = "#7FFF00" // MooveIt brand color
	}

	priority := messaging.PriorityHigh
	if payload.Priority == "normal" {
		priority = messaging.PriorityDefault
	}

	return &messaging.AndroidConfig{
		Priority: "high",
		Notification: &messaging.AndroidNotification{
			Sound:                 sound,
			ChannelID:             channelID,
			Priority:              priority,
			DefaultSound:          sound == "default",
			Icon:                  icon,
			Color:                 color,
			Tag:                   payload.Tag,
			DefaultVibrateTimings: true,
		},
	}
}

// getAPNSConfig returns iOS-specific notification configuration
func getAPNSConfig(payload NotificationPayload) *messaging.APNSConfig {
	sound := payload.Sound
	if sound == "" {
		sound = "default"
	}

	badge := 1
	if payload.BadgeCount != nil {
		badge = *payload.BadgeCount
	}

	return &messaging.APNSConfig{
		Payload: &messaging.APNSPayload{
			Aps: &messaging.Aps{
				Sound:            sound,
				Badge:            &badge,
				MutableContent:   true,
				ContentAvailable: true,
			},
		},
	}
}

// SendNotificationToToken sends a notification to a specific FCM token
func SendNotificationToToken(ctx context.Context, token string, payload NotificationPayload) error {
	if MessagingClient == nil {
		log.Println("Warning: Firebase not initialized. Skipping notification.")
		return nil
	}

	// Convert data map to string map (required by FCM)
	dataStrings := make(map[string]string)
	for key, value := range payload.Data {
		// Marshal complex types to JSON strings
		switch v := value.(type) {
		case string:
			dataStrings[key] = v
		case int, int64, float64, bool:
			dataStrings[key] = fmt.Sprintf("%v", v)
		default:
			jsonData, err := json.Marshal(v)
			if err != nil {
				log.Printf("Error marshaling data for key %s: %v", key, err)
				continue
			}
			dataStrings[key] = string(jsonData)
		}
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		},
		Data:  dataStrings,
		Token: token,
	}

	// Add image if provided
	if payload.Image != "" {
		message.Notification.ImageURL = payload.Image
	}

	// Set Android-specific options
	message.Android = getAndroidConfig(payload)

	// Set iOS-specific options
	message.APNS = getAPNSConfig(payload)

	response, err := MessagingClient.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	log.Printf("Successfully sent notification to token: %s, response: %s", token, response)
	return nil
}

// SendNotificationToMultipleTokens sends a notification to multiple FCM tokens
func SendNotificationToMultipleTokens(ctx context.Context, tokens []string, payload NotificationPayload) (*messaging.BatchResponse, error) {
	if MessagingClient == nil {
		log.Println("Warning: Firebase not initialized. Skipping notifications.")
		return nil, nil
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("no tokens provided")
	}

	// Convert data map to string map
	dataStrings := make(map[string]string)
	for key, value := range payload.Data {
		switch v := value.(type) {
		case string:
			dataStrings[key] = v
		case int, int64, float64, bool:
			dataStrings[key] = fmt.Sprintf("%v", v)
		default:
			jsonData, err := json.Marshal(v)
			if err != nil {
				log.Printf("Error marshaling data for key %s: %v", key, err)
				continue
			}
			dataStrings[key] = string(jsonData)
		}
	}

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		},
		Data:   dataStrings,
		Tokens: tokens,
	}

	// Add image if provided
	if payload.Image != "" {
		message.Notification.ImageURL = payload.Image
	}

	// Set Android-specific options
	message.Android = getAndroidConfig(payload)

	// Set iOS-specific options
	message.APNS = getAPNSConfig(payload)

	response, err := MessagingClient.SendEachForMulticast(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("error sending multicast message: %v", err)
	}

	log.Printf("Successfully sent %d messages, %d failures", response.SuccessCount, response.FailureCount)

	// Log any failures
	if response.FailureCount > 0 {
		for idx, resp := range response.Responses {
			if !resp.Success {
				log.Printf("Failed to send to token %s: %v", tokens[idx], resp.Error)
			}
		}
	}

	return response, nil
}

// SendRideRequestNotification sends a ride request notification to a driver
func SendRideRequestNotification(ctx context.Context, driverToken string, rideID uint, clientName, pickupAddress, destAddress string, fare float64) error {
	payload := NotificationPayload{
		Title: "New Ride Request",
		Body:  fmt.Sprintf("%s requested a ride from %s", clientName, pickupAddress),
		Data: map[string]interface{}{
			"type":           "ride_request",
			"rideId":         rideID,
			"clientName":     clientName,
			"pickupAddress":  pickupAddress,
			"destAddress":    destAddress,
			"fare":           fare,
			"notificationId": fmt.Sprintf("ride_request_%d", rideID),
		},
	}

	return SendNotificationToToken(ctx, driverToken, payload)
}

// SendRideAcceptedNotification sends a notification to client when driver accepts
func SendRideAcceptedNotification(ctx context.Context, clientToken string, rideID uint, driverName, vehicleDetails string, eta int) error {
	payload := NotificationPayload{
		Title: "Ride Accepted!",
		Body:  fmt.Sprintf("%s accepted your ride request. ETA: %d minutes", driverName, eta),
		Data: map[string]interface{}{
			"type":           "ride_accepted",
			"rideId":         rideID,
			"driverName":     driverName,
			"vehicleDetails": vehicleDetails,
			"eta":            eta,
			"notificationId": fmt.Sprintf("ride_accepted_%d", rideID),
		},
	}

	return SendNotificationToToken(ctx, clientToken, payload)
}

// SendDriverArrivedNotification sends notification when driver arrives at pickup
func SendDriverArrivedNotification(ctx context.Context, clientToken string, rideID uint, driverName string) error {
	payload := NotificationPayload{
		Title: "Driver Arrived",
		Body:  fmt.Sprintf("%s has arrived at your pickup location", driverName),
		Data: map[string]interface{}{
			"type":           "driver_arrived",
			"rideId":         rideID,
			"driverName":     driverName,
			"notificationId": fmt.Sprintf("driver_arrived_%d", rideID),
		},
	}

	return SendNotificationToToken(ctx, clientToken, payload)
}

// SendRideStartedNotification sends notification when ride starts
func SendRideStartedNotification(ctx context.Context, clientToken string, rideID uint, driverName string) error {
	payload := NotificationPayload{
		Title: "Ride Started",
		Body:  fmt.Sprintf("Your ride with %s has started", driverName),
		Data: map[string]interface{}{
			"type":           "ride_started",
			"rideId":         rideID,
			"driverName":     driverName,
			"notificationId": fmt.Sprintf("ride_started_%d", rideID),
		},
	}

	return SendNotificationToToken(ctx, clientToken, payload)
}

// SendRideCompletedNotification sends notification when ride is completed
func SendRideCompletedNotification(ctx context.Context, clientToken string, rideID uint, fare float64) error {
	payload := NotificationPayload{
		Title: "Ride Completed",
		Body:  fmt.Sprintf("Your ride is complete. Total fare: KES %.2f", fare),
		Data: map[string]interface{}{
			"type":           "ride_completed",
			"rideId":         rideID,
			"fare":           fare,
			"notificationId": fmt.Sprintf("ride_completed_%d", rideID),
		},
	}

	return SendNotificationToToken(ctx, clientToken, payload)
}

// SendScheduledRidesAvailableNotification notifies clients about available scheduled rides
func SendScheduledRidesAvailableNotification(ctx context.Context, clientTokens []string, count int) (*messaging.BatchResponse, error) {
	payload := NotificationPayload{
		Title: "Rides Available!",
		Body:  fmt.Sprintf("%d scheduled rides are now available. Book yours now!", count),
		Data: map[string]interface{}{
			"type":           "scheduled_rides_available",
			"count":          count,
			"notificationId": "scheduled_rides_available",
		},
	}

	return SendNotificationToMultipleTokens(ctx, clientTokens, payload)
}

// SendBroadcastNotification sends a broadcast notification to all users
func SendBroadcastNotification(ctx context.Context, tokens []string, title, body, imageURL string, data map[string]interface{}) (*messaging.BatchResponse, error) {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["type"] = "broadcast"
	data["notificationId"] = "broadcast_" + fmt.Sprintf("%d", ctx.Value("timestamp"))

	payload := NotificationPayload{
		Title: title,
		Body:  body,
		Image: imageURL,
		Data:  data,
	}

	return SendNotificationToMultipleTokens(ctx, tokens, payload)
}

// SubscribeToTopic subscribes tokens to a topic for targeted messaging
func SubscribeToTopic(ctx context.Context, tokens []string, topic string) error {
	if MessagingClient == nil {
		log.Println("Warning: Firebase not initialized. Skipping topic subscription.")
		return nil
	}

	response, err := MessagingClient.SubscribeToTopic(ctx, tokens, topic)
	if err != nil {
		return fmt.Errorf("error subscribing to topic: %v", err)
	}

	log.Printf("Successfully subscribed %d tokens to topic %s, %d failures", response.SuccessCount, topic, response.FailureCount)
	return nil
}

// UnsubscribeFromTopic unsubscribes tokens from a topic
func UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) error {
	if MessagingClient == nil {
		log.Println("Warning: Firebase not initialized. Skipping topic unsubscription.")
		return nil
	}

	response, err := MessagingClient.UnsubscribeFromTopic(ctx, tokens, topic)
	if err != nil {
		return fmt.Errorf("error unsubscribing from topic: %v", err)
	}

	log.Printf("Successfully unsubscribed %d tokens from topic %s, %d failures", response.SuccessCount, topic, response.FailureCount)
	return nil
}

// SendTopicNotification sends a notification to a topic
func SendTopicNotification(ctx context.Context, topic string, payload NotificationPayload) error {
	if MessagingClient == nil {
		log.Println("Warning: Firebase not initialized. Skipping topic notification.")
		return nil
	}

	// Convert data map to string map
	dataStrings := make(map[string]string)
	for key, value := range payload.Data {
		switch v := value.(type) {
		case string:
			dataStrings[key] = v
		case int, int64, float64, bool:
			dataStrings[key] = fmt.Sprintf("%v", v)
		default:
			jsonData, err := json.Marshal(v)
			if err != nil {
				log.Printf("Error marshaling data for key %s: %v", key, err)
				continue
			}
			dataStrings[key] = string(jsonData)
		}
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		},
		Data:  dataStrings,
		Topic: topic,
	}

	if payload.Image != "" {
		message.Notification.ImageURL = payload.Image
	}

	// Set Android-specific options
	message.Android = getAndroidConfig(payload)

	// Set iOS-specific options
	message.APNS = getAPNSConfig(payload)

	response, err := MessagingClient.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("error sending topic message: %v", err)
	}

	log.Printf("Successfully sent notification to topic %s, response: %s", topic, response)
	return nil
}
