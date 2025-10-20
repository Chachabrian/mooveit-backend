package utils

import (
	"math"
	"time"
)

// FareCalculationResult contains the calculated fare and breakdown
type FareCalculationResult struct {
	TotalFare         float64       `json:"totalFare"`
	Distance          float64       `json:"distance"`
	BaseRate          float64       `json:"baseRate"`
	TrafficMultiplier float64       `json:"trafficMultiplier"`
	HasTraffic        bool          `json:"hasTraffic"`
	MinimumFare       float64       `json:"minimumFare"`
	Breakdown         FareBreakdown `json:"breakdown"`
}

// FareBreakdown provides detailed fare breakdown
type FareBreakdown struct {
	BaseFare         float64 `json:"baseFare"`
	DistanceFare     float64 `json:"distanceFare"`
	TrafficSurcharge float64 `json:"trafficSurcharge"`
	Total            float64 `json:"total"`
}

const (
	// Base rates in KES
	StandardRatePerKm   = 35.0  // Normal rate per km
	TrafficRatePerKm    = 38.0  // Rate per km during high traffic
	MinimumFare         = 150.0 // Minimum fare for distances <= 3km
	MinimumFareDistance = 3.0   // Distance threshold for minimum fare in km
)

// CalculateDynamicFare calculates the fare based on distance and traffic conditions
func CalculateDynamicFare(pickupLat, pickupLng, destLat, destLng float64) FareCalculationResult {
	// Calculate distance using Haversine formula
	distance := HaversineDistance(pickupLat, pickupLng, destLat, destLng)

	// Check if traffic is likely at current time
	hasTraffic := IsLikelyTrafficTime()

	// Determine rate per km
	var ratePerKm float64
	var trafficMultiplier float64

	if hasTraffic {
		ratePerKm = TrafficRatePerKm
		trafficMultiplier = TrafficRatePerKm / StandardRatePerKm
	} else {
		ratePerKm = StandardRatePerKm
		trafficMultiplier = 1.0
	}

	// Calculate fare
	var totalFare float64
	var baseFare float64
	var distanceFare float64
	var trafficSurcharge float64

	// Apply minimum fare for distances <= 3km
	if distance <= MinimumFareDistance {
		totalFare = MinimumFare
		baseFare = MinimumFare
		distanceFare = 0
		trafficSurcharge = 0
	} else {
		// Calculate distance-based fare
		distanceFare = distance * ratePerKm
		totalFare = distanceFare

		// Calculate traffic surcharge (difference from standard rate)
		if hasTraffic {
			standardFare := distance * StandardRatePerKm
			trafficSurcharge = distanceFare - standardFare
		}
	}

	// Round to 2 decimal places
	totalFare = math.Round(totalFare*100) / 100

	return FareCalculationResult{
		TotalFare:         totalFare,
		Distance:          math.Round(distance*100) / 100,
		BaseRate:          ratePerKm,
		TrafficMultiplier: trafficMultiplier,
		HasTraffic:        hasTraffic,
		MinimumFare:       MinimumFare,
		Breakdown: FareBreakdown{
			BaseFare:         baseFare,
			DistanceFare:     math.Round(distanceFare*100) / 100,
			TrafficSurcharge: math.Round(trafficSurcharge*100) / 100,
			Total:            totalFare,
		},
	}
}

// IsLikelyTrafficTime determines if current time is likely to have high traffic
// Based on typical Nairobi traffic patterns
func IsLikelyTrafficTime() bool {
	now := time.Now()

	// Get current time in East Africa Time (EAT - UTC+3)
	// Adjust based on your server timezone
	location, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		// Fallback to local time if timezone not available
		location = time.Local
	}

	currentTime := now.In(location)
	hour := currentTime.Hour()
	dayOfWeek := currentTime.Weekday()

	// Weekend traffic is generally lighter
	isWeekend := dayOfWeek == time.Saturday || dayOfWeek == time.Sunday

	if isWeekend {
		// Weekend peak hours (lighter traffic)
		// Late morning to early evening: 11 AM - 7 PM
		return hour >= 11 && hour < 19
	}

	// Weekday peak hours (heavy traffic)
	// Morning rush: 6 AM - 10 AM
	// Evening rush: 4 PM - 8 PM
	isMorningRush := hour >= 6 && hour < 10
	isEveningRush := hour >= 16 && hour < 20

	return isMorningRush || isEveningRush
}

// IsHighTrafficZone checks if coordinates are in a known high-traffic area
// You can expand this with actual GPS coordinates of busy areas in Nairobi
func IsHighTrafficZone(lat, lng float64) bool {
	// Define high-traffic zones in Nairobi (example coordinates)
	highTrafficZones := []struct {
		centerLat float64
		centerLng float64
		radiusKm  float64
		name      string
	}{
		// Nairobi CBD
		{-1.2864, 36.8172, 3.0, "Nairobi CBD"},
		// Westlands
		{-1.2675, 36.8078, 2.0, "Westlands"},
		// Thika Road Corridor
		{-1.2195, 36.8909, 5.0, "Thika Road"},
		// Mombasa Road Industrial Area
		{-1.3226, 36.8519, 3.0, "Industrial Area"},
		// Ngong Road
		{-1.2964, 36.7821, 2.5, "Ngong Road"},
		// Jogoo Road
		{-1.2845, 36.8562, 4.0, "Jogoo Road"},
		// Outering Road (Northern section near Kasarani)
		{-1.2198, 36.8923, 3.5, "Outering Road North"},
		// Outering Road (Eastern section near Embakasi)
		{-1.3019, 36.9141, 3.0, "Outering Road East"},
		// Outering Road (Southern section near Imara Daima)
		{-1.3431, 36.8889, 2.5, "Outering Road South"},
	}

	// Check if location is within any high-traffic zone
	for _, zone := range highTrafficZones {
		distance := HaversineDistance(lat, lng, zone.centerLat, zone.centerLng)
		if distance <= zone.radiusKm {
			return true
		}
	}

	return false
}

// GetTrafficMultiplier returns the traffic multiplier based on location and time
func GetTrafficMultiplier(pickupLat, pickupLng, destLat, destLng float64) float64 {
	hasTraffic := IsLikelyTrafficTime()

	// Check if either pickup or destination is in high-traffic zone
	pickupInTrafficZone := IsHighTrafficZone(pickupLat, pickupLng)
	destInTrafficZone := IsHighTrafficZone(destLat, destLng)

	if hasTraffic && (pickupInTrafficZone || destInTrafficZone) {
		return TrafficRatePerKm / StandardRatePerKm
	} else if hasTraffic {
		return TrafficRatePerKm / StandardRatePerKm
	}

	return 1.0
}
