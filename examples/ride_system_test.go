package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/chachabrian/mooveit-backend/internal/models"
)

// Example usage of the real-time ride-sharing system
func main() {
	baseURL := "http://localhost:8080/api"
	
	// Example client token (you would get this from login)
	clientToken := "your_client_jwt_token_here"
	driverToken := "your_driver_jwt_token_here"

	fmt.Println("=== Real-Time Ride-Sharing System Examples ===")
	
	// 1. Driver updates location
	fmt.Println("\n1. Driver Location Update:")
	updateDriverLocation(baseURL, driverToken)
	
	// 2. Driver sets availability
	fmt.Println("\n2. Driver Availability Update:")
	updateDriverAvailability(baseURL, driverToken)
	
	// 3. Client finds nearby drivers
	fmt.Println("\n3. Find Nearby Drivers:")
	findNearbyDrivers(baseURL, clientToken)
	
	// 4. Client requests a ride
	fmt.Println("\n4. Request Ride:")
	requestRide(baseURL, clientToken)
	
	// 5. Driver updates ride status
	fmt.Println("\n5. Update Ride Status:")
	updateRideStatus(baseURL, driverToken, 1)
}

func updateDriverLocation(baseURL, token string) {
	url := baseURL + "/driver/location"
	
	locationData := map[string]interface{}{
		"lat":     40.7128,
		"lng":     -74.0060,
		"heading": 45.0,
	}
	
	jsonData, _ := json.Marshal(locationData)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status: %d\n", resp.StatusCode)
}

func updateDriverAvailability(baseURL, token string) {
	url := baseURL + "/driver/availability"
	
	availabilityData := map[string]interface{}{
		"isAvailable": true,
	}
	
	jsonData, _ := json.Marshal(availabilityData)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status: %d\n", resp.StatusCode)
}

func findNearbyDrivers(baseURL, token string) {
	url := baseURL + "/rides/nearby-drivers?lat=40.7128&lng=-74.0060&radius=10"
	
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status: %d\n", resp.StatusCode)
}

func requestRide(baseURL, token string) {
	url := baseURL + "/rides/request"
	
	rideData := map[string]interface{}{
		"pickup": map[string]interface{}{
			"lat":     40.7128,
			"lng":     -74.0060,
			"address": "123 Main St, New York, NY",
		},
		"destination": map[string]interface{}{
			"lat":     40.7589,
			"lng":     -73.9851,
			"address": "456 Broadway, New York, NY",
		},
	}
	
	jsonData, _ := json.Marshal(rideData)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status: %d\n", resp.StatusCode)
}

func updateRideStatus(baseURL, token string, rideID int) {
	url := fmt.Sprintf("%s/rides/%d/status", baseURL, rideID)
	
	statusData := map[string]interface{}{
		"status": models.RideStatusStarted,
	}
	
	jsonData, _ := json.Marshal(statusData)
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status: %d\n", resp.StatusCode)
}
