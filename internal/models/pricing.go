package models

import (
	"gorm.io/gorm"
)

// PricingZone represents a geographical zone with specific pricing
type PricingZone struct {
	gorm.Model
	Name        string  `json:"name" gorm:"not null"`
	CenterLat   float64 `json:"centerLat" gorm:"not null"`
	CenterLng   float64 `json:"centerLng" gorm:"not null"`
	Radius      float64 `json:"radius" gorm:"not null"` // in kilometers
	BaseFare    float64 `json:"baseFare" gorm:"not null"`
	PerKmRate   float64 `json:"perKmRate" gorm:"not null"`
	PerMinRate  float64 `json:"perMinRate" gorm:"not null"`
	MinFare     float64 `json:"minFare" gorm:"not null"`
	MaxFare     float64 `json:"maxFare" gorm:"not null"`
	IsActive    bool    `json:"isActive" gorm:"not null;default:true"`
	Description string  `json:"description"`
}

// TableName specifies the table name
func (PricingZone) TableName() string {
	return "pricing_zones"
}

// DriverPricing represents driver-specific pricing overrides
type DriverPricing struct {
	gorm.Model
	DriverID   uint         `json:"driverId" gorm:"not null"`
	ZoneID     uint         `json:"zoneId" gorm:"not null"`
	BaseFare   *float64     `json:"baseFare,omitempty"`   // Override base fare
	PerKmRate  *float64     `json:"perKmRate,omitempty"`  // Override per km rate
	PerMinRate *float64     `json:"perMinRate,omitempty"` // Override per minute rate
	MinFare    *float64     `json:"minFare,omitempty"`    // Override minimum fare
	MaxFare    *float64     `json:"maxFare,omitempty"`    // Override maximum fare
	IsActive   bool         `json:"isActive" gorm:"not null;default:true"`
	Driver     *User        `json:"driver,omitempty" gorm:"foreignKey:DriverID"`
	Zone       *PricingZone `json:"zone,omitempty" gorm:"foreignKey:ZoneID"`
}

// TableName specifies the table name
func (DriverPricing) TableName() string {
	return "driver_pricing"
}

// TripCompletion represents completed trip details
type TripCompletion struct {
	gorm.Model
	RideID         uint            `json:"rideId" gorm:"not null;unique"`
	DriverID       uint            `json:"driverId" gorm:"not null"`
	ClientID       uint            `json:"clientId" gorm:"not null"`
	ActualFare     float64         `json:"actualFare" gorm:"not null"`
	ActualDistance float64         `json:"actualDistance" gorm:"not null"`
	ActualDuration int             `json:"actualDuration" gorm:"not null"` // in minutes
	DriverRating   *float64        `json:"driverRating,omitempty"`
	ClientRating   *float64        `json:"clientRating,omitempty"`
	DriverNotes    string          `json:"driverNotes,omitempty"`
	ClientNotes    string          `json:"clientNotes,omitempty"`
	CompletedAt    *gorm.DeletedAt `json:"completedAt,omitempty"`
	Driver         *User           `json:"driver,omitempty" gorm:"foreignKey:DriverID"`
	Client         *User           `json:"client,omitempty" gorm:"foreignKey:ClientID"`
	Ride           *RideRequest    `json:"ride,omitempty" gorm:"foreignKey:RideID"`
}

// TableName specifies the table name
func (TripCompletion) TableName() string {
	return "trip_completions"
}
