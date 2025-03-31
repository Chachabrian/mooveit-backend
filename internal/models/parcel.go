package models

import "gorm.io/gorm"

type Parcel struct {
	gorm.Model
	RideID            uint   `json:"rideId" gorm:"not null"`
	ParcelImage       string `json:"parcelImage" gorm:"not null"`
	ParcelDescription string `json:"parcelDescription" gorm:"not null"`
	ReceiverName      string `json:"receiverName" gorm:"not null"`
	ReceiverContact   string `json:"receiverContact" gorm:"not null"`
	Destination       string `json:"destination" gorm:"not null"`
}
