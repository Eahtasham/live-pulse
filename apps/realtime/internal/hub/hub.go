package hub

import "log/slog"

// BroadcastMessage carries a message to be sent to all clients in a room.
type BroadcastMessage struct {
	Code    string
	Message []byte
}

// Hub maintains the set of active rooms and routes messages.
type Hub struct {
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	Broadcast  chan BroadcastMessage
}

// NewHub creates and returns a new Hub.
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		Broadcast:  make(chan BroadcastMessage),
	}
}

// Run starts the hub's main event loop. Should be called as a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			room, ok := h.rooms[client.code]
			if !ok {
				room = newRoom(client.code)
				h.rooms[client.code] = room
				slog.Info("room created", "code", client.code)
			}
			room.add(client)

		case client := <-h.unregister:
			room, ok := h.rooms[client.code]
			if !ok {
				continue
			}
			room.remove(client)
			close(client.send)
			if room.isEmpty() {
				delete(h.rooms, client.code)
				slog.Info("room destroyed", "code", client.code)
			}

		case msg := <-h.Broadcast:
			room, ok := h.rooms[msg.Code]
			if !ok {
				continue
			}
			room.broadcast(msg.Message)
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(c *Client) {
	h.register <- c
}

// RoomCount returns the number of active rooms (for testing/monitoring).
func (h *Hub) RoomCount() int {
	return len(h.rooms)
}

// ClientCount returns the number of clients in a specific room.
func (h *Hub) ClientCount(code string) int {
	room, ok := h.rooms[code]
	if !ok {
		return 0
	}
	return room.count()
}
