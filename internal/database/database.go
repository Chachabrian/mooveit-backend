
package database

import (
    "fmt"
    "os"

    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "github.com/chachabrian/mooveit-backend/internal/models"
)

func InitDB() (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
        os.Getenv("DB_PORT"),
    )

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, err
    }

    // Auto migrate the schema
    err = db.AutoMigrate(
        &models.User{},
        &models.Booking{},
        &models.Ride{},
    )
    if err != nil {
        return nil, err
    }

    return db, nil
}
