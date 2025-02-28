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
    gorm.Model      // This embeds ID, CreatedAt, UpdatedAt, and DeletedAt
    Username     string     `gorm:"column:username;unique;not null"`
    Email        string     `gorm:"column:email;unique;not null"`
    Password     string     `gorm:"-:migration"`                              // Temporary field for password handling
    PasswordHash string     `gorm:"column:password_hash;not null"`
    PhoneNumber  string     `gorm:"column:phone_number"`
    UserType     string     `gorm:"column:user_type;not null"`
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
