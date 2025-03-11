package database

import (
    "fmt"
    "os"
    "time"

    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
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

    // Configure GORM with custom logger
    config := &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    }

    // Attempt to connect with retries
    var db *gorm.DB
    var err error
    maxRetries := 5
    for i := 0; i < maxRetries; i++ {
        db, err = gorm.Open(postgres.Open(dsn), config)
        if err == nil {
            break
        }
        if i < maxRetries-1 {
            time.Sleep(time.Second * 5)
            continue
        }
        return nil, fmt.Errorf("failed to connect to database after %d attempts: %v", maxRetries, err)
    }

    // Run migrations
    if err := RunMigrations(db); err != nil {
        return nil, fmt.Errorf("failed to run migrations: %v", err)
    }

    return db, nil
}