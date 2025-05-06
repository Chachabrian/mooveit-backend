package models

import "gorm.io/gorm"

type Parcel struct {
	gorm.Model
	RideID            uint   `gorm:"not null"`
	ParcelImage       string `gorm:"not null"`
	ParcelDescription string `gorm:"not null"`
	ReceiverName      string `gorm:"not null"`
	ReceiverContact   string `gorm:"not null"`
	ReceiverEmail     string `gorm:"not null"`
	Destination       string `gorm:"not null"`
	Ride              Ride   `gorm:"foreignKey:RideID"`
}
