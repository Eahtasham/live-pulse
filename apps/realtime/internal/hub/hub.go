package hub

import "log/slog"

// BroadcastMessage carries a message to be sent to all clients in a room.
type BroadcastMessage struct {
	Code           string
	Message        []byte
	CloseAfterSend bool
}

// RoomSubscriber is called when rooms are created/destroyed for Pub/Sub wiring.
type RoomSubscriber interface {
	Subscribe(code string)
	Unsubscribe(code string)
}

// Hub maintains the set of active rooms and routes messages.
type Hub struct {
	rooms       map[string]*Room
	closedRooms map[string]bool
	register    chan *Client
	unregister  chan *Client
	Broadcast   chan BroadcastMessage
	subscriber  RoomSubscriber
}

// NewHub creates and returns a new Hub.
func NewHub(subscriber RoomSubscriber) *Hub {
	return &Hub{
		rooms:       make(map[string]*Room),
		closedRooms: make(map[string]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		Broadcast:   make(chan BroadcastMessage),
		subscriber:  subscriber,
	}
}

// Run starts the hub's main event loop. Should be called as a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if h.closedRooms[client.code] {
				slog.Info("rejecting client for closed room", "code", client.code)
				close(client.send)
				go client.conn.Close()
				continue
			}
			room, ok := h.rooms[client.code]
			if !ok {
				room = newRoom(client.code)
				h.rooms[client.code] = room
				slog.Info("room created", "code", client.code)
				if h.subscriber != nil {
					h.subscriber.Subscribe(client.code)
				}
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
				if h.subscriber != nil {
					h.subscriber.Unsubscribe(client.code)
				}
			}

		case msg := <-h.Broadcast:
			room, ok := h.rooms[msg.Code]
			if !ok {
				continue
			}
			room.broadcast(msg.Message)
			if msg.CloseAfterSend {
				h.closeRoom(msg.Code)
			}
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

// IsRoomClosed returns whether a room has been closed by a session_closed event.
func (h *Hub) IsRoomClosed(code string) bool {
	return h.closedRooms[code]
}

// closeRoom forcefully closes all connections in a room and removes it.
func (h *Hub) closeRoom(code string) {
	room, ok := h.rooms[code]
	if !ok {
		return
	}

	for client := range room.clients {
		close(client.send)
		delete(room.clients, client)
	}

	delete(h.rooms, code)
	h.closedRooms[code] = true
	slog.Info("room closed (session archived)", "code", code)

	if h.subscriber != nil {
		h.subscriber.Unsubscribe(code)
	}
}
