package models

import (
	"time"

	"gorm.io/gorm"
)

// DriverLocation represents a driver's current location and status
type DriverLocation struct {
	gorm.Model
	DriverID    uint      `json:"driverId" gorm:"not null;uniqueIndex"`
	Latitude    float64   `json:"lat" gorm:"not null"`
	Longitude   float64   `json:"lng" gorm:"not null"`
	Heading     float64   `json:"heading" gorm:"not null;default:0"`
	IsOnline    bool      `json:"isOnline" gorm:"not null;default:false"`
	IsAvailable bool      `json:"isAvailable" gorm:"not null;default:false"`
	LastSeen    time.Time `json:"lastSeen" gorm:"not null"`
	Driver      *User     `json:"driver,omitempty" gorm:"foreignKey:DriverID"`
}

// TableName specifies the table name
func (DriverLocation) TableName() string {
	return "driver_locations"
}

// RideRequest represents a ride request from a client
type RideRequest struct {
	gorm.Model
	ClientID   uint    `json:"clientId" gorm:"not null"`
	DriverID   *uint   `json:"driverId,omitempty" gorm:"null"`
	PickupLat  float64 `json:"pickupLat" gorm:"not null"`
	PickupLng  float64 `json:"pickupLng" gorm:"not null"`
	PickupAddr string  `json:"pickupAddress" gorm:"not null"`
	DestLat    float64 `json:"destLat" gorm:"not null"`
	DestLng    float64 `json:"destLng" gorm:"not null"`
	DestAddr   string  `json:"destAddress" gorm:"not null"`
	Status     string  `json:"status" gorm:"not null;default:'pending'"` // pending, accepted, started, completed, cancelled
	Price      float64 `json:"price,omitempty"`
	Distance   float64 `json:"distance,omitempty"` // in kilometers
	Duration   int     `json:"duration,omitempty"` // in minutes
	Client     *User   `json:"client,omitempty" gorm:"foreignKey:ClientID"`
	Driver     *User   `json:"driver,omitempty" gorm:"foreignKey:DriverID"`
}

// TableName specifies the table name
func (RideRequest) TableName() string {
	return "ride_requests"
}

// DriverRating represents driver ratings
type DriverRating struct {
	gorm.Model
	DriverID uint    `json:"driverId" gorm:"not null"`
	ClientID uint    `json:"clientId" gorm:"not null"`
	RideID   uint    `json:"rideId" gorm:"not null"`
	Rating   float64 `json:"rating" gorm:"not null;check:rating >= 1 AND rating <= 5"`
	Comment  string  `json:"comment,omitempty"`
	Driver   *User   `json:"driver,omitempty" gorm:"foreignKey:DriverID"`
	Client   *User   `json:"client,omitempty" gorm:"foreignKey:ClientID"`
}

// TableName specifies the table name
func (DriverRating) TableName() string {
	return "driver_ratings"
}

// RideStatus constants
const (
	RideStatusPending   = "pending"
	RideStatusAccepted  = "accepted"
	RideStatusArrived   = "arrived"
	RideStatusStarted   = "started"
	RideStatusCompleted = "completed"
	RideStatusCancelled = "cancelled"
)

// DriverStatus constants
const (
	DriverStatusOnline    = "online"
	DriverStatusOffline   = "offline"
	DriverStatusBusy      = "busy"
	DriverStatusAvailable = "available"
)
