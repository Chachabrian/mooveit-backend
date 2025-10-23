package services

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Client represents a WebSocket client
type Client struct {
	ID       uint
	UserType string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mutex      sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("Client %d connected", client.ID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mutex.Unlock()
			log.Printf("Client %d disconnected", client.ID)

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastToUser sends a message to a specific user
func (h *Hub) BroadcastToUser(userID uint, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.ID == userID {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}
}

// BroadcastToUserType sends a message to all users of a specific type
func (h *Hub) BroadcastToUserType(userType string, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.UserType == userType {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}
}

// GetConnectedClients returns the number of connected clients
func (h *Hub) GetConnectedClients() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// WebSocket message types
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// DriverLocationUpdate represents a driver location update
type DriverLocationUpdate struct {
	DriverID uint `json:"driverId"`
	Location struct {
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		Heading float64 `json:"heading"`
	} `json:"location"`
}

// DriverStatusUpdate represents a driver status update
type DriverStatusUpdate struct {
	DriverID uint   `json:"driverId"`
	Status   string `json:"status"` // online, offline, busy, available
}

// RideAccepted represents a ride acceptance notification
type RideAccepted struct {
	RideID        uint `json:"rideId"`
	DriverID      uint `json:"driverId"`
	EstimatedTime int  `json:"estimatedTime"` // in minutes
}

// DriverArrived represents a driver arrival notification
type DriverArrived struct {
	RideID   uint `json:"rideId"`
	DriverID uint `json:"driverId"`
}

// RideStarted represents a ride start notification
type RideStarted struct {
	RideID   uint `json:"rideId"`
	DriverID uint `json:"driverId"`
}

// RideCompleted represents a ride completion notification
type RideCompleted struct {
	RideID         uint    `json:"rideId"`
	DriverID       uint    `json:"driverId"`
	ActualFare     float64 `json:"actualFare"`
	ActualDistance float64 `json:"actualDistance"`
	ActualDuration int     `json:"actualDuration"`
}

// RideRejected represents a ride rejection notification
type RideRejected struct {
	RideID uint   `json:"rideId"`
	Reason string `json:"reason"`
}

// RequestRide represents a ride request from client
type RequestRide struct {
	Pickup struct {
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		Address string  `json:"address"`
	} `json:"pickup"`
	Destination struct {
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		Address string  `json:"address"`
	} `json:"destination"`
}

// CancelRide represents a ride cancellation
type CancelRide struct {
	RideID uint `json:"rideId"`
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request, userID uint, userType string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		ID:       userID,
		UserType: userType,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}

	client.Hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages
		var wsMessage WebSocketMessage
		if err := json.Unmarshal(message, &wsMessage); err != nil {
			log.Printf("Error unmarshaling WebSocket message: %v", err)
			continue
		}

		// Process different message types
		switch wsMessage.Type {
		case "request_ride":
			// Handle ride request
			c.handleRideRequest(wsMessage.Data)
		case "cancel_ride":
			// Handle ride cancellation
			c.handleRideCancellation(wsMessage.Data)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	defer c.Conn.Close()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

			// Log message type for debugging
			var msg WebSocketMessage
			if err := json.Unmarshal(message, &msg); err == nil {
				log.Printf("[WS→] Sent to client %d (%s): %s", c.ID, c.UserType, msg.Type)
			}
		}
	}
}

// handleRideRequest processes ride requests
func (c *Client) handleRideRequest(data interface{}) {
	// This would be implemented to handle ride requests
	// For now, just log the request
	log.Printf("Ride request from client %d: %+v", c.ID, data)
}

// handleRideCancellation processes ride cancellations
func (c *Client) handleRideCancellation(data interface{}) {
	// This would be implemented to handle ride cancellations
	// For now, just log the cancellation
	log.Printf("Ride cancellation from client %d: %+v", c.ID, data)
}

// SendDriverLocationUpdate sends a driver location update to all clients
func (hub *Hub) SendDriverLocationUpdate(update DriverLocationUpdate) {
	message := WebSocketMessage{
		Type: "driver_location_update",
		Data: update,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling driver location update: %v", err)
		return
	}

	// Broadcast to all connected clients using the safe method
	hub.BroadcastToAll(data)
}

// SendDriverLocationUpdateToClient sends a driver location update to a specific client
func (hub *Hub) SendDriverLocationUpdateToClient(clientID uint, update DriverLocationUpdate) {
	message := WebSocketMessage{
		Type: "driver_location_update",
		Data: update,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling driver location update: %v", err)
		return
	}

	hub.BroadcastToUser(clientID, data)
	log.Printf("Sent driver location update to client %d: driver %d at (%.6f, %.6f)",
		clientID, update.DriverID, update.Location.Lat, update.Location.Lng)
}

// BroadcastToAll sends a message to all connected clients
func (hub *Hub) BroadcastToAll(message []byte) {
	hub.mutex.RLock()
	defer hub.mutex.RUnlock()

	log.Printf("Broadcasting to %d connected clients", len(hub.clients))
	for client := range hub.clients {
		select {
		case client.Send <- message:
			// Message sent successfully
		default:
			// Client's send channel is full, skip
			log.Printf("Warning: Could not send to client %d (channel full)", client.ID)
		}
	}
}

// SendRideAccepted sends a ride acceptance notification to the client
func (hub *Hub) SendRideAccepted(clientID uint, accepted RideAccepted) {
	message := WebSocketMessage{
		Type: "ride_accepted",
		Data: accepted,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling ride accepted: %v", err)
		return
	}

	hub.BroadcastToUser(clientID, data)
}

// SendDriverArrived sends a driver arrival notification to the client
func (hub *Hub) SendDriverArrived(clientID uint, arrived DriverArrived) {
	message := WebSocketMessage{
		Type: "driver_arrived",
		Data: arrived,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling driver arrived: %v", err)
		return
	}

	hub.BroadcastToUser(clientID, data)
}

// SendRideStarted sends a ride start notification to the client
func (hub *Hub) SendRideStarted(clientID uint, started RideStarted) {
	message := WebSocketMessage{
		Type: "ride_started",
		Data: started,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling ride started: %v", err)
		return
	}

	hub.BroadcastToUser(clientID, data)
}

// SendRideCompleted sends a ride completion notification to the client
func (hub *Hub) SendRideCompleted(clientID uint, completed RideCompleted) {
	message := WebSocketMessage{
		Type: "ride_completed",
		Data: completed,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling ride completed: %v", err)
		return
	}

	hub.BroadcastToUser(clientID, data)
}

// SendRideRejected sends a ride rejection notification to the client
func (hub *Hub) SendRideRejected(clientID uint, rejected RideRejected) {
	message := WebSocketMessage{
		Type: "ride_rejected",
		Data: rejected,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling ride rejected: %v", err)
		return
	}

	hub.BroadcastToUser(clientID, data)
}
