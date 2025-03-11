package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserType string

const (
	UserTypeClient UserType = "client"
	UserTypeDriver UserType = "driver"
)

type User struct {
	gorm.Model
	Username     string   `gorm:"column:username;unique;not null"`
	Email        string   `gorm:"column:email;unique;not null"`
	Password     string    `gorm:"-"`
	PasswordHash string   `gorm:"column:password_hash;not null"`
	PhoneNumber  string   `gorm:"column:phone_number"`
	UserType     UserType `gorm:"column:user_type;type:text;check:user_type IN ('client', 'driver');not null"`
	CarPlate     string   `gorm:"column:car_plate"`
	CarMake      string   `gorm:"column:car_make"`
	CarColor     string   `gorm:"column:car_color"`
}

// TableName specifies the table name
func (User) TableName() string {
	return "users"
}

func (u *User) HashPassword() error {
	if u.Password == "" {
		return nil
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
}
