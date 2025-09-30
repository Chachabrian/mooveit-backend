package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// InitRedis initializes the Redis client
func InitRedis() error {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://redis:6379" // Default Redis address for Docker
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %v", err)
	}

	RedisClient = redis.NewClient(opt)

	// Test the connection
	ctx := context.Background()
	_, err = RedisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return nil
}

// SetDriverLocation stores driver location in Redis
func SetDriverLocation(ctx context.Context, driverID uint, lat, lng, heading float64) error {
	locationData := map[string]interface{}{
		"lat":     lat,
		"lng":     lng,
		"heading": heading,
		"updated": time.Now().Unix(),
	}

	data, err := json.Marshal(locationData)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("driver:location:%d", driverID)
	return RedisClient.Set(ctx, key, data, time.Hour).Err()
}

// GetDriverLocation retrieves driver location from Redis
func GetDriverLocation(ctx context.Context, driverID uint) (lat, lng, heading float64, err error) {
	key := fmt.Sprintf("driver:location:%d", driverID)
	data, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		return 0, 0, 0, err
	}

	var locationData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &locationData); err != nil {
		return 0, 0, 0, err
	}

	lat, _ = locationData["lat"].(float64)
	lng, _ = locationData["lng"].(float64)
	heading, _ = locationData["heading"].(float64)

	return lat, lng, heading, nil
}

// SetDriverAvailability stores driver availability status
func SetDriverAvailability(ctx context.Context, driverID uint, isAvailable bool) error {
	key := fmt.Sprintf("driver:availability:%d", driverID)
	value := "true"
	if !isAvailable {
		value = "false"
	}
	return RedisClient.Set(ctx, key, value, time.Hour).Err()
}

// GetDriverAvailability retrieves driver availability status
func GetDriverAvailability(ctx context.Context, driverID uint) (bool, error) {
	key := fmt.Sprintf("driver:availability:%d", driverID)
	result, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result == "true", nil
}

// SetNearbyDrivers stores nearby drivers for a location
func SetNearbyDrivers(ctx context.Context, lat, lng float64, drivers []map[string]interface{}) error {
	key := fmt.Sprintf("nearby:drivers:%.6f:%.6f", lat, lng)
	data, err := json.Marshal(drivers)
	if err != nil {
		return err
	}
	return RedisClient.Set(ctx, key, data, 5*time.Minute).Err()
}

// GetNearbyDrivers retrieves nearby drivers for a location
func GetNearbyDrivers(ctx context.Context, lat, lng float64) ([]map[string]interface{}, error) {
	key := fmt.Sprintf("nearby:drivers:%.6f:%.6f", lat, lng)
	data, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var drivers []map[string]interface{}
	if err := json.Unmarshal([]byte(data), &drivers); err != nil {
		return nil, err
	}

	return drivers, nil
}

// PublishDriverLocation publishes driver location update to Redis pub/sub
func PublishDriverLocation(ctx context.Context, driverID uint, lat, lng, heading float64) error {
	locationData := map[string]interface{}{
		"driverId": driverID,
		"location": map[string]float64{
			"lat":     lat,
			"lng":     lng,
			"heading": heading,
		},
		"timestamp": time.Now().Unix(),
	}

	data, err := json.Marshal(locationData)
	if err != nil {
		return err
	}

	return RedisClient.Publish(ctx, "driver:location:updates", data).Err()
}

// PublishRideUpdate publishes ride status update to Redis pub/sub
func PublishRideUpdate(ctx context.Context, rideID uint, status string, data map[string]interface{}) error {
	updateData := map[string]interface{}{
		"rideId":    rideID,
		"status":    status,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return err
	}

	return RedisClient.Publish(ctx, "ride:updates", jsonData).Err()
}
