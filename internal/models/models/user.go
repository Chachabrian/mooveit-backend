
package models

import (
    "gorm.io/gorm"
    "golang.org/x/crypto/bcrypt"
)

type UserType string

const (
    UserTypeClient UserType = "client"
    UserTypeDriver UserType = "driver"
)

type User struct {
    gorm.Model
    Email     string   `gorm:"uniqueIndex;not null" json:"email"`
    Password  string   `gorm:"not null" json:"-"`
    Name      string   `json:"name"`
    Phone     string   `json:"phone"`
    UserType  UserType `gorm:"not null" json:"userType"`
}

func (u *User) HashPassword() error {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    u.Password = string(hashedPassword)
    return nil
}

func (u *User) CheckPassword(password string) error {
    return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}
