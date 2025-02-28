package models

import (
    "gorm.io/gorm"
    "time"
)

type Ride struct {
    gorm.Model
    DriverID    uint      `json:"driverId" gorm:"not null"`
    Driver      User      `json:"driver"`
    Destination string    `json:"destination" gorm:"not null"`
    TruckSize   string    `json:"truckSize" gorm:"not null"`
    Price       float64   `json:"price" gorm:"not null"`
    Date        time.Time `json:"date" gorm:"not null"`
}