package models

import (
    "gorm.io/gorm"
)

type BookingStatus string

const (
    BookingStatusPending   BookingStatus = "pending"
    BookingStatusAccepted BookingStatus = "accepted"
    BookingStatusRejected BookingStatus = "rejected"
     BookingStatusCancelled BookingStatus = "cancelled"
)

type Booking struct {
    gorm.Model
    ClientID    uint          `json:"clientId" gorm:"not null"`
    Client      User          `json:"client"`
    RideID      uint          `json:"rideId" gorm:"not null"`
    Ride        Ride          `json:"ride" gorm:"foreignKey:RideID"`
    Status      BookingStatus `json:"status" gorm:"not null;default:'pending'"`
}
