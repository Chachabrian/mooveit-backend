package handlers

import (
	"github.com/chachabrian/mooveit-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// WebSocketHandler handles WebSocket connections
func WebSocketHandler(hub *services.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("userId")
		userType := c.GetString("userType")

		// Convert Gin's ResponseWriter to http.ResponseWriter
		services.HandleWebSocket(hub, c.Writer, c.Request, userID, userType)
	}
}
