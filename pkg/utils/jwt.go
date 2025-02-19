
package utils

import (
    "os"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/chachabrian/mooveit-backend/internal/models"
)

func GenerateToken(user *models.User) (string, error) {
    claims := jwt.MapClaims{
        "id":       user.ID,
        "email":    user.Email,
        "userType": user.UserType,
        "exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func ValidateToken(tokenString string) (*jwt.Token, error) {
    return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return []byte(os.Getenv("JWT_SECRET")), nil
    })
}
