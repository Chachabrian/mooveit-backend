package models

import (
	"time"

	"gorm.io/gorm"
)

type Ride struct {
	gorm.Model
	DriverID        uint      `json:"driverId" gorm:"not null"`
	CurrentLocation string    `json:"currentLocation" gorm:"not null"`
	Destination     string    `json:"destination" gorm:"not null"`
	TruckSize       string    `json:"truckSize" gorm:"not null"`
	Price           float64   `json:"price" gorm:"not null"`
	Date            time.Time `json:"date" gorm:"not null"`
	Status          string    `json:"status" gorm:"not null;default:'available'"`
	Driver          *User     `json:"driver,omitempty" gorm:"-"`
}
