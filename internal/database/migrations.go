package database

import (
	"github.com/chachabrian/mooveit-backend/internal/models"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	// Create tables if they don't exist
	err := db.AutoMigrate(
		&models.User{},
		&models.Booking{},
		&models.Ride{},
		&models.Parcel{},
	)
	if err != nil {
		return err
	}

	// Update users table
	if db.Migrator().HasTable(&models.User{}) {
		columns := []string{
			"ADD COLUMN IF NOT EXISTS car_plate text DEFAULT ''",
			"ADD COLUMN IF NOT EXISTS car_make text DEFAULT ''",
			"ADD COLUMN IF NOT EXISTS car_color text DEFAULT ''",
			"ADD COLUMN IF NOT EXISTS user_type text DEFAULT 'client'",
		}

		for _, column := range columns {
			if err := db.Exec("ALTER TABLE users " + column).Error; err != nil {
				return err
			}
		}

		// Update constraint
		db.Exec(`ALTER TABLE users DROP CONSTRAINT IF EXISTS users_user_type_check`)
		db.Exec(`ALTER TABLE users ADD CONSTRAINT users_user_type_check CHECK (user_type IN ('client', 'driver'))`)
	}

	return nil
}
