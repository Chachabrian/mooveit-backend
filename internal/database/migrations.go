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
		&models.OTP{},
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

	// Handle parcels table separately
	if !db.Migrator().HasTable(&models.Parcel{}) {
		// If table doesn't exist, create it with all columns
		if err := db.AutoMigrate(&models.Parcel{}); err != nil {
			return err
		}
	} else {
		// If table exists, handle receiver_email column carefully
		// First, check if the column exists
		var columnExists bool
		err := db.Raw(`
			SELECT EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_name = 'parcels' 
				AND column_name = 'receiver_email'
			)`).Scan(&columnExists).Error
		if err != nil {
			return err
		}

		if !columnExists {
			// Add column as nullable first
			if err := db.Exec(`ALTER TABLE parcels ADD COLUMN receiver_email text DEFAULT ''`).Error; err != nil {
				return err
			}

			// Update existing records
			if err := db.Exec(`UPDATE parcels SET receiver_email = COALESCE(receiver_contact, '') WHERE receiver_email = ''`).Error; err != nil {
				return err
			}

			// Set default value for new records
			if err := db.Exec(`ALTER TABLE parcels ALTER COLUMN receiver_email SET DEFAULT ''`).Error; err != nil {
				return err
			}

			// Make it not null after setting defaults
			if err := db.Exec(`ALTER TABLE parcels ALTER COLUMN receiver_email SET NOT NULL`).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
