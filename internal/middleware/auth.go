
package middleware

import (
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/chachabrian/mooveit-backend/pkg/utils"
)

func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(401, gin.H{"error": "Invalid authorization header format"})
            c.Abort()
            return
        }

        token, err := utils.ValidateToken(parts[1])
        if err != nil || !token.Valid {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            c.JSON(401, gin.H{"error": "Invalid token claims"})
            c.Abort()
            return
        }

        c.Set("userId", uint(claims["id"].(float64)))
        c.Set("userType", claims["userType"].(string))
        c.Next()
    }
}
