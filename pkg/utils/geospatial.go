package utils

import (
	"math"
)

// HaversineDistance calculates the distance between two points on Earth
// using the Haversine formula. Returns distance in kilometers.
func HaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371 // Earth's radius in kilometers

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlng := lng2Rad - lng1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlng/2)*math.Sin(dlng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// Bearing calculates the initial bearing from point 1 to point 2
// Returns bearing in degrees (0-360)
func Bearing(lat1, lng1, lat2, lng2 float64) float64 {
	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180

	dlng := lng2Rad - lng1Rad

	y := math.Sin(dlng) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) -
		math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(dlng)

	bearing := math.Atan2(y, x) * 180 / math.Pi

	// Normalize to 0-360 degrees
	if bearing < 0 {
		bearing += 360
	}

	return bearing
}

// IsWithinRadius checks if a point is within a specified radius of another point
func IsWithinRadius(centerLat, centerLng, pointLat, pointLng, radiusKm float64) bool {
	distance := HaversineDistance(centerLat, centerLng, pointLat, pointLng)
	return distance <= radiusKm
}

// CalculateETA estimates the time to arrival based on distance and average speed
// distance in kilometers, averageSpeed in km/h
func CalculateETA(distanceKm, averageSpeedKmh float64) int {
	if averageSpeedKmh <= 0 {
		averageSpeedKmh = 30 // Default average speed in city traffic
	}

	etaHours := distanceKm / averageSpeedKmh
	etaMinutes := int(etaHours * 60)

	// Minimum 1 minute
	if etaMinutes < 1 {
		etaMinutes = 1
	}

	return etaMinutes
}

// CalculatePrice estimates the price based on distance and base rate
func CalculatePrice(distanceKm, baseRatePerKm float64) float64 {
	if baseRatePerKm <= 0 {
		baseRatePerKm = 2.0 // Default rate per km
	}

	// Base fare + distance fare
	baseFare := 5.0 // Base fare
	distanceFare := distanceKm * baseRatePerKm

	return baseFare + distanceFare
}

// Point represents a geographical point
type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// BoundingBox represents a rectangular area
type BoundingBox struct {
	NorthEast Point `json:"northEast"`
	SouthWest Point `json:"southWest"`
}

// GetBoundingBox creates a bounding box around a center point
func GetBoundingBox(centerLat, centerLng, radiusKm float64) BoundingBox {
	const earthRadius = 6371 // Earth's radius in kilometers

	// Calculate the angular distance
	angularDistance := radiusKm / earthRadius

	// Calculate the latitude bounds
	latMin := centerLat - (angularDistance * 180 / math.Pi)
	latMax := centerLat + (angularDistance * 180 / math.Pi)

	// Calculate the longitude bounds
	lngMin := centerLng - (angularDistance * 180 / math.Pi / math.Cos(centerLat*math.Pi/180))
	lngMax := centerLng + (angularDistance * 180 / math.Pi / math.Cos(centerLat*math.Pi/180))

	return BoundingBox{
		NorthEast: Point{Lat: latMax, Lng: lngMax},
		SouthWest: Point{Lat: latMin, Lng: lngMin},
	}
}

// IsPointInBoundingBox checks if a point is within a bounding box
func IsPointInBoundingBox(point Point, bbox BoundingBox) bool {
	return point.Lat >= bbox.SouthWest.Lat &&
		point.Lat <= bbox.NorthEast.Lat &&
		point.Lng >= bbox.SouthWest.Lng &&
		point.Lng <= bbox.NorthEast.Lng
}
